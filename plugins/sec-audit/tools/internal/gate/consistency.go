package gate

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/cvss"
	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/report"
	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/table"
)

// consistent compares the report with the findings table, which is the
// single source of truth: a report value that differs from its row is the
// report's to fix, and a row score that differs from its vector is the
// table's.
type consistent struct {
	opts Options
	tbl  *table.Table
	rep  *report.Report
	// rows holds the first row of each ID, in table order.
	rows     []*table.Row
	problems []Problem
}

func consistency(opts Options, tbl *table.Table, rep *report.Report) []Problem {
	c := &consistent{opts: opts, tbl: tbl, rep: rep}
	c.uniqueIDs()
	c.scores()
	c.sections(rep.Findings, table.StatusScored, "finding section", "###")
	c.sections(rep.Hardening, table.StatusHardening, "hardening note", "####")
	c.overview()
	c.summary()
	c.hardeningIndex()
	return c.problems
}

func (c *consistent) add(where, format string, args ...any) {
	c.problems = append(c.problems, Problem{Check: CheckConsistency, Where: where, Message: fmt.Sprintf(format, args...)})
}

func (c *consistent) uniqueIDs() {
	count := map[string]int{}
	for i := range c.tbl.Rows {
		r := &c.tbl.Rows[i]
		count[r.ID]++
		if count[r.ID] == 1 {
			c.rows = append(c.rows, r)
		}
	}
	for _, r := range c.rows {
		if n := count[r.ID]; n > 1 {
			c.add(c.opts.Table, "%s: id appears in %d rows", r.ID, n)
		}
	}
}

func (c *consistent) scored() []*table.Row {
	return slices.DeleteFunc(slices.Clone(c.rows), func(r *table.Row) bool { return r.Status != table.StatusScored })
}

// scores recomputes every scored row's scores and severities from its
// vectors.
func (c *consistent) scores() {
	for _, r := range c.scored() {
		c.score(r.ID, "31", r.CVSS31, r.Score31, r.Sev31, cvss.Score31)
		if r.CVSS40 != table.Empty {
			c.score(r.ID, "40", r.CVSS40, r.Score40, r.Sev40, cvss.Score40)
		}
	}
}

func (c *consistent) score(id, version, vector, score, sev string, calc func(string) (float64, string, error)) {
	want, wantSev, err := calc(vector)
	if err != nil {
		c.add(c.opts.Table, "%s: cvss%s: %v", id, version, err)
		return
	}
	if s := cvss.Format(want); score != s {
		c.add(c.opts.Table, "%s: score%s is %s but its vector scores %s", id, version, score, s)
	}
	if sev != wantSev {
		c.add(c.opts.Table, "%s: sev%s is %s but its vector rates %s", id, version, sev, wantSev)
	}
}

// sections checks that each section's heading is in the template's form,
// marked by the heading's # run, after an anchor line of its own; that each
// row of status has exactly one section of the kind given; that no other
// row has one; and that each section's values equal its row's.
func (c *consistent) sections(secs []report.Section, status, kind, marker string) {
	first := map[string]int{}
	for i := range secs {
		s := &secs[i]
		at := where(c.opts.Report, s.Line)
		if s.Malformed {
			c.add(at, "%s: %s heading is not in the form %q", s.ID, kind, marker+" "+s.ID+" — <title>")
		}
		if want := strings.ToLower(s.ID); s.Anchor != want {
			msg := fmt.Sprintf("%s heading for %s has no <a id=%q> anchor line before it", kind, s.ID, want)
			if s.Anchor != "" {
				msg += fmt.Sprintf("; the anchor before it is <a id=%q>", s.Anchor)
			}
			c.add(at, "%s", msg)
		}
		row := c.tbl.ByID[s.ID]
		switch {
		case row == nil:
			c.add(at, "%s: the report has a %s for it, but the table has no row", s.ID, kind)
		case row.Status != status:
			c.add(at, "%s: %s, but the report has a %s for it", s.ID, row.Status, kind)
		case first[s.ID] != 0:
			c.add(at, "%s: second %s, after the one at line %d", s.ID, kind, first[s.ID])
		default:
			c.section(at, s, row)
		}
		if first[s.ID] == 0 {
			first[s.ID] = s.Line
		}
	}
	for _, r := range c.rows {
		if r.Status == status && first[r.ID] == 0 {
			c.add(c.opts.Report, "%s: %s, but the report has no %s for it", r.ID, status, kind)
		}
	}
}

func (c *consistent) section(at string, s *report.Section, row *table.Row) {
	// A malformed heading's title is the parser's guess, and its form is
	// already a problem.
	if !s.Malformed && s.Title != row.Title {
		c.add(at, "%s: section title is %q, table has %q", s.ID, s.Title, row.Title)
	}
	if row.Status != table.StatusScored {
		return
	}
	c.cvssRow(at, s.ID, "CVSS 3.1", [3]string{s.CVSS31, s.Score31, s.Sev31}, [3]string{row.CVSS31, row.Score31, row.Sev31})
	c.cvssRow(at, s.ID, "CVSS 4.0", [3]string{s.CVSS40, s.Score40, s.Sev40}, [3]string{row.CVSS40, row.Score40, row.Sev40})

	if s.Rel == nil {
		c.add(at, "%s: section has no **Affected versions** row", s.ID)
		return
	}
	for _, rel := range c.tbl.Releases {
		got, ok := s.Rel[rel]
		switch {
		case !ok:
			c.add(at, "%s: section has no verdict for %s", s.ID, rel)
		case got != row.Rel[rel]:
			c.add(at, "%s: section verdict for %s is %q, table has %q", s.ID, rel, got, row.Rel[rel])
		}
	}
	for _, rel := range slices.Sorted(maps.Keys(s.Rel)) {
		switch {
		case s.Rel[rel] == "":
			c.add(at, "%s: section's **Affected versions** part %q is not in the form <rel> **<VERDICT>**", s.ID, rel)
		case !slices.Contains(c.tbl.Releases, rel):
			c.add(at, "%s: section names release %q, which the table does not", s.ID, rel)
		}
	}
}

// cvssRow compares a section's vector, score and severity with its row's.
func (c *consistent) cvssRow(at, id, label string, got, want [3]string) {
	switch {
	case got[0] == "":
		c.add(at, "%s: section has no **%s** row", id, label)
		return
	case got[1] == "":
		c.add(at, "%s: section's **%s** cell is not in the form `<vector>` — **<score> <Severity>**, or —", id, label)
		return
	}
	if got[0] != want[0] {
		c.add(at, "%s: section %s vector is %q, table has %q", id, label, got[0], want[0])
		return
	}
	if got[1] != want[1] {
		c.add(at, "%s: section %s score is %q, table has %q", id, label, got[1], want[1])
	}
	if got[2] != want[2] {
		c.add(at, "%s: section %s severity is %q, table has %q", id, label, got[2], want[2])
	}
}

// overview checks that the overview's rows are the scored rows, each with
// its row's values, sorted by score31, then score40, descending, then ID.
func (c *consistent) overview() {
	if c.rep.OverviewReleases == nil {
		c.add(c.opts.Report, "no finding overview: want a table whose header is "+
			"| ID | CVSS 3.1 | CVSS 4.0 | Severity | Finding | Affected (<rel> / …) | Found by | Default exposure |")
		return
	}
	relsMatch := slices.Equal(c.rep.OverviewReleases, c.tbl.Releases)
	if !relsMatch {
		c.add(c.opts.Report, "overview names releases %s, table has %s",
			strings.Join(c.rep.OverviewReleases, " / "), strings.Join(c.tbl.Releases, " / "))
	}

	count := map[string]int{}
	var order []string
	for i := range c.rep.Overview {
		o := &c.rep.Overview[i]
		count[o.ID]++
		if count[o.ID] > 1 {
			continue
		}
		row := c.tbl.ByID[o.ID]
		switch {
		case row == nil:
			c.add(c.opts.Report, "%s: the overview has a row for it, but the table has none", o.ID)
		case row.Status != table.StatusScored:
			c.add(c.opts.Report, "%s: %s, but the report has an overview row for it", o.ID, row.Status)
		default:
			order = append(order, o.ID)
			c.overviewRow(o, row, relsMatch)
		}
	}
	for _, r := range c.scored() {
		switch n := count[r.ID]; {
		case n == 0:
			c.add(c.opts.Report, "%s: scored, but the overview has no row for it", r.ID)
		case n > 1:
			c.add(c.opts.Report, "%s: %d overview rows", r.ID, n)
		}
	}
	c.overviewOrder(order)
}

func (c *consistent) overviewRow(o *report.OverviewRow, row *table.Row, relsMatch bool) {
	for _, f := range []struct{ label, got, want string }{
		{"CVSS 3.1", o.Score31, row.Score31},
		{"CVSS 4.0", o.Score40, row.Score40},
		{"severity", o.Sev, row.Sev31},
		{"title", o.Title, row.Title},
		{"found by", o.FoundBy, row.FoundBy},
		{"default exposure", o.Default, row.Default},
	} {
		if f.got != f.want {
			c.add(c.opts.Report, "%s: overview %s is %q, table has %q", o.ID, f.label, f.got, f.want)
		}
	}
	if !relsMatch {
		return
	}
	if len(o.Rel) != len(c.tbl.Releases) {
		c.add(c.opts.Report, "%s: overview has %d verdicts, table has %d releases", o.ID, len(o.Rel), len(c.tbl.Releases))
		return
	}
	for i, rel := range c.tbl.Releases {
		if o.Rel[i] != row.Rel[rel] {
			c.add(c.opts.Report, "%s: overview verdict for %s is %q, table has %q", o.ID, rel, o.Rel[i], row.Rel[rel])
		}
	}
}

// overviewOrder reports the first scored overview row out of order, judged
// by the table's scores so that a mistyped score cell cannot hide a
// misplaced row. At the first position where got and the sorted order
// differ, got lists its row there before the row the order wants.
func (c *consistent) overviewOrder(got []string) {
	want := slices.Clone(got)
	slices.SortStableFunc(want, func(a, b string) int {
		ra, rb := c.tbl.ByID[a], c.tbl.ByID[b]
		return cmp.Or(
			cmp.Compare(scoreOf(rb.Score31), scoreOf(ra.Score31)),
			cmp.Compare(scoreOf(rb.Score40), scoreOf(ra.Score40)),
			compareIDs(a, b),
		)
	})
	for i := range got {
		if got[i] != want[i] {
			c.add(c.opts.Report, "overview lists %s before %s: rows are sorted by CVSS 3.1, then CVSS 4.0, descending, then by ID",
				got[i], want[i])
			return
		}
	}
}

// scoreOf reads a score cell; an empty one sorts below every score.
func scoreOf(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	return f
}

// compareIDs orders IDs as a reader would: runs of digits by their value, so
// F-9 sorts before F-10, and everything else bytewise.
func compareIDs(a, b string) int {
	for a != "" && b != "" {
		ca, ra := idChunk(a)
		cb, rb := idChunk(b)
		if r := compareChunks(ca, cb); r != 0 {
			return r
		}
		a, b = ra, rb
	}
	return cmp.Compare(len(a), len(b))
}

// idChunk splits s after its leading run of digits, or of non-digits.
func idChunk(s string) (chunk, rest string) {
	digit := isDigit(rune(s[0]))
	i := strings.IndexFunc(s, func(r rune) bool { return isDigit(r) != digit })
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i:]
}

func isDigit(r rune) bool { return '0' <= r && r <= '9' }

func compareChunks(a, b string) int {
	if isDigit(rune(a[0])) && isDigit(rune(b[0])) {
		ta, tb := strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
		return cmp.Or(cmp.Compare(len(ta), len(tb)), strings.Compare(ta, tb))
	}
	return strings.Compare(a, b)
}

// summary compares the summary's count sentences with the table's counts.
func (c *consistent) summary() {
	var want report.Counts
	for _, r := range c.rows {
		switch r.Status {
		case table.StatusScored:
			want.Scored++
			switch r.Sev31 {
			case "Critical":
				want.Critical++
			case "High":
				want.High++
			case "Medium":
				want.Medium++
			case "Low":
				want.Low++
			}
		case table.StatusHardening:
			want.Hardening++
		}
	}
	got := c.rep.Summary
	if !got.Found {
		c.add(c.opts.Report, "summary count sentences not found; want %q and %q",
			fmt.Sprintf("**%d scored findings**: %d Critical, %d High, %d Medium and %d Low",
				want.Scored, want.Critical, want.High, want.Medium, want.Low),
			fmt.Sprintf("**%d [hardening notes](#hardening-index)**", want.Hardening))
		return
	}
	for _, f := range []struct {
		label     string
		got, want int
	}{
		{"scored findings", got.Scored, want.Scored},
		{"Critical", got.Critical, want.Critical},
		{"High", got.High, want.High},
		{"Medium", got.Medium, want.Medium},
		{"Low", got.Low, want.Low},
		{"hardening notes", got.Hardening, want.Hardening},
	} {
		if f.got != f.want {
			c.add(c.opts.Report, "summary says %d %s, table has %d", f.got, f.label, f.want)
		}
	}
}

// hardeningIndex checks that the index lists each hardening row once and
// nothing else.
func (c *consistent) hardeningIndex() {
	count := map[string]int{}
	for _, id := range c.rep.HardeningIndex {
		count[id]++
		if count[id] > 1 {
			continue
		}
		row := c.tbl.ByID[id]
		switch {
		case row == nil:
			c.add(c.opts.Report, "%s: the hardening index lists it, but the table has no row", id)
		case row.Status != table.StatusHardening:
			c.add(c.opts.Report, "%s: %s, but the hardening index lists it", id, row.Status)
		}
	}
	for _, id := range c.rep.HardeningIndex {
		if n := count[id]; n > 1 {
			c.add(c.opts.Report, "%s: the hardening index lists it %d times", id, n)
			count[id] = 1
		}
	}
	for _, r := range c.rows {
		if r.Status == table.StatusHardening && count[r.ID] == 0 {
			c.add(c.opts.Report, "%s: hardening, but the hardening index does not list it", r.ID)
		}
	}
}

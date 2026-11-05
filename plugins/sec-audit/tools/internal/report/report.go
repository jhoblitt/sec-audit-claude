// Package report parses an audit report for the markers the report template
// and audit-gate share: anchors and internal links, code links, finding
// sections, hardening notes and their index, the finding overview, the
// summary counts, and leftover placeholders.
//
// Parse reads what is there and judges nothing: a marker it cannot read is
// kept verbatim, or left empty, so that comparing the report with the
// findings table surfaces it. Headings, anchors and links inside fenced
// code are not markers; placeholders are found everywhere.
package report

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// empty is the value the findings table uses for "no value"; Parse reads a
// cell of — as empty, so the two compare directly.
const empty = "-"

const hardeningIndexAnchor = "hardening-index"

// idPattern is the shape of a finding or hardening ID: a prefix and a
// number, with an optional letter.
const idPattern = `[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)*-\d+[A-Za-z]?`

var (
	fenceRE       = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
	anchorRE      = regexp.MustCompile(`<a\s+id="([^"]*)"\s*>`)
	anchorLineRE  = regexp.MustCompile(`^\s*<a\s+id="([^"]*)"\s*>(?:</a>)?\s*$`)
	internalRE    = regexp.MustCompile(`\]\(#([^)\s]*)\)`)
	urlRE         = regexp.MustCompile("https?://[^\\s<>()\\[\\]\"'`|]+")
	placeholderRE = regexp.MustCompile(`\{\{[A-Z][A-Z0-9_]*\}\}`)
	headingRE     = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*$`)
	sectionHeadRE = regexp.MustCompile(`^(` + idPattern + `) — (.+)$`)
	idLeadRE      = regexp.MustCompile(`^(` + idPattern + `)(?:[^A-Za-z0-9]|$)(.*)$`)
	headLinkRE    = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	headMarkupRE  = regexp.MustCompile("</?a\\b[^>]*>|[*_`]")
	cvssCellRE    = regexp.MustCompile("^`([^`]+)` — \\*\\*(\\S+) (\\S+)\\*\\*$")
	verdictRE     = regexp.MustCompile(`^(.+?)\s+\*\*([^*]+)\*\*$`)
	idCellRE      = regexp.MustCompile(`^\[([^\]]+)\]\(#([^)]*)\)$`)
	delimCellRE   = regexp.MustCompile(`^:?-+:?$`)
	lineFragRE    = regexp.MustCompile(`^L(\d+)(?:C\d+)?(?:-L?(\d+)(?:C\d+)?)?$`)
	scoredRE      = regexp.MustCompile(`\*\*(\d+)\s+scored\s+findings\*\*:\s+(\d+)\s+Critical,\s+(\d+)\s+High,\s+(\d+)\s+Medium\s+and\s+(\d+)\s+Low`)
	hardeningRE   = regexp.MustCompile(`\*\*(\d+)\s+\[hardening\s+notes\]\(#hardening-index\)\*\*`)
)

// overviewHeader is the overview's header row; the Affected cell, empty
// here, is matched separately because it names the releases.
var overviewHeader = []string{
	"ID", "CVSS 3.1", "CVSS 4.0", "Severity", "Finding", "", "Found by", "Default exposure",
}

const affectedCol = 5

var overviewVerdicts = map[string]string{"yes": "PRESENT", "partly": "PARTIAL", "no": "ABSENT"}

// Pos is a position in the report.
type Pos struct {
	// Line is 1-based.
	Line int
}

// Link is an internal link, [text](#Target).
type Link struct {
	// Target is the fragment without its '#'.
	Target string
	Pos
}

// CodeLink is a link to a file at a ref in a repository:
// https://github.com/<owner>/<repo>/blob/<ref>/<path>[?plain=1][#L<a>[-L<b>]],
// the same with blame or raw in place of blob, a
// raw.githubusercontent.com/<owner>/<repo>/<ref>/<path> link, or a link on
// another host whose path holds /blob/, /-/blob/, /src/ or /tree/.
type CodeLink struct {
	// Host is the URL's host, lower-cased: github.com, or the other host.
	Host string
	// Owner, Repo, Ref and Path are read from the URL path. Ref is the one
	// path element after blob, blame or raw, so a ref with a slash in it,
	// such as the sub-module tag lib/v0.2.0, runs on into Path; PathAt
	// re-splits the two at a ref the caller knows. On another host they are
	// a best effort: Owner holds every element before the repository, and
	// Ref skips Gitea's branch, commit or tag element.
	Owner, Repo, Ref, Path string
	// L1 and L2 are the first and last cited lines: both 0 when the link
	// cites no line, and equal when it cites one.
	L1, L2 int
	Pos
}

// PathAt returns the path the link cites when ref, which may hold slashes,
// is the ref it cites: what follows ref and a slash in the link's ref and
// path joined, or "" when they are ref alone. It reports false when they do
// not start with ref.
func (cl CodeLink) PathAt(ref string) (string, bool) {
	cited := cl.Ref
	if cl.Path != "" {
		cited += "/" + cl.Path
	}
	if cited == ref {
		return "", true
	}
	if path, ok := strings.CutPrefix(cited, ref+"/"); ok {
		return path, true
	}
	return "", false
}

// Section is a finding section or a hardening note.
//
// A finding section is a heading "### <ID> — <title>", and a hardening note
// the same with "####", where the ID is a prefix and a number: F-01, H-12,
// F-03a. A level-3 or level-4 heading in any other form is a section too,
// marked Malformed, when it starts with an ID once inline anchors, links
// and * _ ` markup are stripped from it. The template puts the anchor line
// <a id="<lowercase ID>"></a> before each heading; one that is missing or
// different is recorded in Anchor, never a reason to skip the heading. A
// section runs to the next heading of its own level or above.
type Section struct {
	ID, Title string
	// Malformed marks a heading not in the form "<ID> — <title>", such as
	// "F-04: <title>", "**F-04** — <title>" or a bare "F-04". Its Title is
	// what follows the ID with the separator trimmed, markup stripped.
	Malformed bool
	// CVSS31, Score31 and Sev31 are read from the section's
	// "| **CVSS 3.1** | `<vector>` — **<score> <Severity>** |" row; the
	// CVSS 4.0 fields from its "**CVSS 4.0**" row. A cell of — sets all
	// three to "-". A cell in neither form leaves the vector holding the
	// cell verbatim and the score and severity empty, and a missing row
	// leaves all three empty.
	CVSS31, Score31, Sev31 string
	CVSS40, Score40, Sev40 string
	// Rel maps each release to its verdict as written in the
	// "| **Affected versions** | <rel> **<VERDICT>**; … [— <note>] |" row.
	// A part not in that form is kept as a key with an empty verdict. Rel is
	// nil when the section has no such row.
	Rel map[string]string
	// Anchor is the id of the anchor line directly before the heading,
	// blank lines between them allowed, as written; "" when there is none.
	Anchor string
	// Pos is the heading's position.
	Pos
}

// OverviewRow is one row of the finding overview, each cell with its
// escaped pipes unescaped.
type OverviewRow struct {
	// ID is the link text of an ID cell [<ID>](#<lowercase ID>); a cell in
	// any other form is kept verbatim.
	ID string
	// Score31 and Score40 are the score cells, "-" for a cell of —.
	Score31, Score40 string
	Sev, Title       string
	// Rel holds the Affected cell's verdicts in release order, mapped to
	// the findings table's PRESENT (yes), PARTIAL (partly) and ABSENT (no);
	// any other word is kept verbatim.
	Rel              []string
	FoundBy, Default string
}

// Counts are the summary's counts, read from
// "**<N> scored findings**: <C> Critical, <H> High, <M> Medium and <L> Low"
// and "**<K> [hardening notes](#hardening-index)**". Either sentence may
// wrap across lines.
type Counts struct {
	Scored, Critical, High, Medium, Low, Hardening int
	// Found reports whether both sentences were found; the counts of the
	// one that was are set regardless.
	Found bool
}

// Report is a parsed audit report.
type Report struct {
	// Anchors maps each <a id="…"> anchor to the line of its first use.
	Anchors       map[string]int
	InternalLinks []Link
	// CodeLinks holds each code link once per line it appears on.
	CodeLinks []CodeLink
	Findings  []Section
	Hardening []Section
	// Overview holds the rows of the first table whose header is the
	// overview's, and OverviewReleases the releases its Affected header
	// names; both are nil when there is no such table.
	Overview         []OverviewRow
	OverviewReleases []string
	// HardeningIndex holds the ID of each row of the first table after
	// the <a id="hardening-index"> anchor, read as OverviewRow.ID is; it is
	// empty when prose such as "None." comes after the anchor before any
	// table does.
	HardeningIndex []string
	Summary        Counts
	// Placeholders holds one position per {{UPPER_SNAKE}} placeholder.
	Placeholders []Pos
}

// Parse reads the report at path.
func Parse(path string) (*Report, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: reading the file the caller names is this function's job
	if err != nil {
		return nil, fmt.Errorf("reading report: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	p := &parser{r: &Report{Anchors: map[string]int{}}}
	p.run(lines)
	return p.r, nil
}

type tableKind int

const (
	noTable tableKind = iota
	overviewTable
	indexTable
)

type parser struct {
	r *Report

	// anchor is the ID of an anchor line seen since the last non-blank line.
	anchor string

	cur      *Section
	curLevel int

	table                   tableKind
	overviewDone, indexDone bool
	indexPending            bool
}

func (p *parser) run(lines []string) {
	fenced := fences(lines)
	var prose []string
	for i, line := range lines {
		pos := Pos{Line: i + 1}
		for range placeholderRE.FindAllStringIndex(line, -1) {
			p.r.Placeholders = append(p.r.Placeholders, pos)
		}
		if fenced[i] {
			continue
		}
		prose = append(prose, line)
		p.links(line, pos)
		p.structure(line, pos)
	}
	p.closeSection()
	p.r.Summary = summary(strings.Join(prose, "\n"))
}

// fences marks each line that opens, closes or sits inside a fenced code
// block.
func fences(lines []string) []bool {
	fenced := make([]bool, len(lines))
	open := ""
	for i, line := range lines {
		run, info, isFence := fence(line)
		switch {
		case open == "" && !isFence:
			continue
		case open == "":
			open = run
		case isFence && strings.HasPrefix(run, open[:1]) && len(run) >= len(open) && strings.TrimSpace(info) == "":
			open = ""
		}
		fenced[i] = true
	}
	return fenced
}

// fence splits a fence line into its run of backticks or tildes and the
// text after the run. A backtick run followed by more backticks on its line
// opens an inline code span, not a fence (CommonMark forbids a backtick in a
// backtick fence's info string).
func fence(line string) (run, info string, ok bool) {
	m := fenceRE.FindStringSubmatch(line)
	if len(m) != 3 {
		return "", "", false
	}
	run, info = m[1], m[2]
	if strings.HasPrefix(run, "`") && strings.Contains(info, "`") {
		return "", "", false
	}
	return run, info, true
}

func (p *parser) links(line string, pos Pos) {
	for _, m := range anchorRE.FindAllStringSubmatch(line, -1) {
		if _, seen := p.r.Anchors[m[1]]; !seen {
			p.r.Anchors[m[1]] = pos.Line
		}
	}
	for _, m := range internalRE.FindAllStringSubmatch(line, -1) {
		p.r.InternalLinks = append(p.r.InternalLinks, Link{Target: m[1], Pos: pos})
	}
	var seen []string
	for _, raw := range urlRE.FindAllString(line, -1) {
		raw = strings.TrimRight(raw, ".,;:!?*")
		if slices.Contains(seen, raw) {
			continue
		}
		seen = append(seen, raw)
		if cl, ok := codeLink(raw); ok {
			cl.Pos = pos
			p.r.CodeLinks = append(p.r.CodeLinks, cl)
		}
	}
}

func codeLink(raw string) (CodeLink, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return CodeLink{}, false
	}
	cl := CodeLink{Host: strings.ToLower(u.Host)}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch cl.Host {
	case "github.com":
		if len(segs) < 4 || !slices.Contains([]string{"blob", "blame", "raw"}, segs[2]) {
			return CodeLink{}, false
		}
		cl.Owner, cl.Repo, cl.Ref, cl.Path = segs[0], segs[1], segs[3], strings.Join(segs[4:], "/")
	case "raw.githubusercontent.com":
		if len(segs) < 4 {
			return CodeLink{}, false
		}
		cl.Owner, cl.Repo, cl.Ref, cl.Path = segs[0], segs[1], segs[2], strings.Join(segs[3:], "/")
	default:
		at := slices.IndexFunc(segs[:max(len(segs)-1, 0)], func(s string) bool { return s == "blob" || s == "src" || s == "tree" })
		if at < 0 {
			return CodeLink{}, false
		}
		repo := segs[:at]
		if n := len(repo); n > 0 && repo[n-1] == "-" {
			repo = repo[:n-1]
		}
		if n := len(repo); n > 0 {
			cl.Owner, cl.Repo = strings.Join(repo[:n-1], "/"), repo[n-1]
		}
		rest := segs[at+1:]
		if segs[at] == "src" && len(rest) > 1 && slices.Contains([]string{"branch", "commit", "tag"}, rest[0]) {
			rest = rest[1:]
		}
		cl.Ref, cl.Path = rest[0], strings.Join(rest[1:], "/")
	}
	cl.L1, cl.L2 = lineRange(u.Fragment)
	return cl, true
}

func lineRange(fragment string) (first, last int) {
	m := lineFragRE.FindStringSubmatch(fragment)
	if m == nil {
		return 0, 0
	}
	first = atoi(m[1])
	if m[2] == "" {
		return first, first
	}
	return first, atoi(m[2])
}

// atoi reads a run of digits. A run too large for an int reads as
// math.MaxInt, a line past the end of any file and a count no table has.
func atoi(digits string) int {
	n, err := strconv.Atoi(digits)
	if err != nil {
		return math.MaxInt
	}
	return n
}

func (p *parser) structure(line string, pos Pos) {
	isRow := strings.HasPrefix(strings.TrimSpace(line), "|")
	var cells []string
	if isRow {
		cells = splitRow(line)
	}

	if p.table != noTable && !isRow {
		p.table = noTable
	}
	switch {
	case p.table == overviewTable:
		if !isDelimiter(cells) {
			p.r.Overview = append(p.r.Overview, overviewRow(cells))
		}
	case p.table == indexTable:
		if !isDelimiter(cells) {
			p.r.HardeningIndex = append(p.r.HardeningIndex, idCell(cell(cells, 0)))
		}
	case isRow && !p.overviewDone:
		if releases, ok := overviewReleases(cells); ok {
			p.r.OverviewReleases = releases
			p.table, p.overviewDone = overviewTable, true
			return
		}
	}
	if isRow && p.indexPending {
		p.table, p.indexPending, p.indexDone = indexTable, false, true
		return
	}

	if m := anchorLineRE.FindStringSubmatch(line); m != nil {
		p.anchor = m[1]
		if m[1] == hardeningIndexAnchor && !p.indexDone {
			p.indexPending = true
		}
		return
	}
	if strings.TrimSpace(line) == "" {
		return
	}
	anchor := p.anchor
	p.anchor = ""

	if m := headingRE.FindStringSubmatch(line); m != nil {
		p.heading(len(m[1]), m[2], anchor, pos)
		return
	}
	if p.indexPending {
		p.indexPending, p.indexDone = false, true
	}
	if isRow && p.cur != nil {
		p.cur.addRow(cells)
	}
}

func (p *parser) heading(level int, text, anchor string, pos Pos) {
	if level == 3 || level == 4 {
		if s, ok := sectionHeading(text); ok {
			p.closeSection()
			s.Anchor, s.Pos = anchor, pos
			p.cur, p.curLevel = &s, level
			return
		}
	}
	if p.cur != nil && level <= p.curLevel {
		p.closeSection()
	}
}

// sectionHeading reads "<ID> — <title>", or, as a Malformed section, any
// heading that starts with an ID once its markup is stripped.
func sectionHeading(text string) (Section, bool) {
	if m := sectionHeadRE.FindStringSubmatch(text); m != nil {
		return Section{ID: m[1], Title: m[2]}, true
	}
	stripped := headMarkupRE.ReplaceAllString(headLinkRE.ReplaceAllString(text, "$1"), "")
	m := idLeadRE.FindStringSubmatch(strings.TrimSpace(stripped))
	if m == nil {
		return Section{}, false
	}
	title := strings.TrimSpace(strings.TrimLeft(m[2], " \t:-–—"))
	return Section{ID: m[1], Title: title, Malformed: true}, true
}

func (p *parser) closeSection() {
	if p.cur == nil {
		return
	}
	if p.curLevel == 3 {
		p.r.Findings = append(p.r.Findings, *p.cur)
	} else {
		p.r.Hardening = append(p.r.Hardening, *p.cur)
	}
	p.cur = nil
}

func (s *Section) addRow(cells []string) {
	if len(cells) < 2 {
		return
	}
	switch cells[0] {
	case "**CVSS 3.1**":
		if s.CVSS31 == "" {
			s.CVSS31, s.Score31, s.Sev31 = cvssCell(cells[1])
		}
	case "**CVSS 4.0**":
		if s.CVSS40 == "" {
			s.CVSS40, s.Score40, s.Sev40 = cvssCell(cells[1])
		}
	case "**Affected versions**":
		if s.Rel == nil {
			s.Rel = sectionVerdicts(cells[1])
		}
	}
}

func cvssCell(c string) (vector, score, sev string) {
	if c == "—" {
		return empty, empty, empty
	}
	if m := cvssCellRE.FindStringSubmatch(c); m != nil {
		return m[1], m[2], m[3]
	}
	return c, "", ""
}

func sectionVerdicts(c string) map[string]string {
	verdicts, _, _ := strings.Cut(c, " — ")
	rel := map[string]string{}
	for part := range strings.SplitSeq(verdicts, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if m := verdictRE.FindStringSubmatch(part); m != nil {
			rel[m[1]] = m[2]
		} else {
			rel[part] = ""
		}
	}
	return rel
}

func overviewReleases(cells []string) ([]string, bool) {
	if len(cells) != len(overviewHeader) {
		return nil, false
	}
	for i, want := range overviewHeader {
		if i != affectedCol && cells[i] != want {
			return nil, false
		}
	}
	inner, ok := strings.CutPrefix(cells[affectedCol], "Affected (")
	if !ok {
		return nil, false
	}
	if inner, ok = strings.CutSuffix(inner, ")"); !ok {
		return nil, false
	}
	releases := []string{}
	for rel := range strings.SplitSeq(inner, " / ") {
		if rel = strings.TrimSpace(rel); rel != "" {
			releases = append(releases, rel)
		}
	}
	return releases, true
}

func overviewRow(cells []string) OverviewRow {
	var rel []string
	for v := range strings.SplitSeq(cell(cells, affectedCol), "/") {
		v = strings.TrimSpace(v)
		if mapped, ok := overviewVerdicts[v]; ok {
			v = mapped
		}
		rel = append(rel, v)
	}
	return OverviewRow{
		ID:      idCell(cell(cells, 0)),
		Score31: dash(cell(cells, 1)),
		Score40: dash(cell(cells, 2)),
		Sev:     cell(cells, 3),
		Title:   cell(cells, 4),
		Rel:     rel,
		FoundBy: cell(cells, 6),
		Default: cell(cells, 7),
	}
}

func idCell(c string) string {
	m := idCellRE.FindStringSubmatch(c)
	if m == nil {
		return c
	}
	if id, target := m[1], m[2]; target == anchorFor(id) {
		return id
	}
	return c
}

// anchorFor returns the anchor the report template gives an ID: the ID in
// lower case. Matching is exact; an anchor in another case is not the ID's.
func anchorFor(id string) string {
	return strings.ToLower(id)
}

func dash(c string) string {
	if c == "—" {
		return empty
	}
	return c
}

func cell(cells []string, i int) string {
	if i < len(cells) {
		return cells[i]
	}
	return ""
}

// splitRow splits a table row into its trimmed cells on each pipe not
// escaped as \|, and unescapes the escaped ones.
func splitRow(line string) []string {
	s := strings.TrimPrefix(strings.TrimSpace(line), "|")
	var cells []string
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '|':
			b.WriteByte('|')
			i++
		case s[i] == '|':
			cells = append(cells, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(s[i])
		}
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		cells = append(cells, rest)
	}
	return cells
}

func isDelimiter(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !delimCellRE.MatchString(c) {
			return false
		}
	}
	return true
}

func summary(text string) Counts {
	var c Counts
	s := scoredRE.FindStringSubmatch(text)
	if s != nil {
		c.Scored, c.Critical, c.High, c.Medium, c.Low = atoi(s[1]), atoi(s[2]), atoi(s[3]), atoi(s[4]), atoi(s[5])
	}
	h := hardeningRE.FindStringSubmatch(text)
	if h != nil {
		c.Hardening = atoi(h[1])
	}
	c.Found = s != nil && h != nil
	return c
}

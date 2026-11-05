// Package table loads data/findings.tsv, the audit's single source of truth
// for each finding's status, CVSS vectors and scores, and per-release
// verdicts.
package table

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

// Empty is the cell value the table uses for "no value".
const Empty = "-"

// The values of the status column.
const (
	StatusScored    = "scored"
	StatusHardening = "hardening"
	StatusRetired   = "retired"
	StatusRefuted   = "refuted"
)

// columns is the fixed part of the header, in order; the rel:<name>
// columns follow it.
var columns = []string{
	"id", "cluster", "status", "cvss31", "score31", "sev31", "cvss40", "score40", "sev40",
	"title", "principal", "default_exposure", "found_by", "confirmed_by",
}

const relPrefix = "rel:"

var (
	statuses   = []string{StatusScored, StatusHardening, StatusRetired, StatusRefuted}
	severities = []string{"None", "Low", "Medium", "High", "Critical"}
	verdicts   = []string{"PRESENT", "ABSENT", "PARTIAL"}
	scoreRE    = regexp.MustCompile(`^(?:10\.0|\d\.\d)$`)
)

// Table is a loaded findings table.
type Table struct {
	// Releases are the rel:<name> column names without the prefix, in
	// header order.
	Releases []string
	Rows     []Row
	// ByID indexes Rows. A duplicated ID maps to its first row; Rows keeps
	// every row, so a caller can still find the duplicate.
	ByID map[string]*Row
}

// Row is one row of the table, each cell verbatim; Empty marks no value.
type Row struct {
	ID, Cluster, Status                             string
	CVSS31, Score31, Sev31                          string
	CVSS40, Score40, Sev40                          string
	Title, Principal, Default, FoundBy, ConfirmedBy string
	// Rel maps each release to its verdict: PRESENT, ABSENT, PARTIAL, or
	// Empty on a row that is not scored.
	Rel map[string]string
}

// Load reads and validates the findings table at path. It rejects a header
// that is not the fixed column list followed by rel:<name> columns, and a
// row whose cells break the table's format; it does not check that a score
// matches its vector.
func Load(path string) (*Table, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: reading the file the caller names is this function's job
	if err != nil {
		return nil, fmt.Errorf("reading findings table: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}

	header := strings.Split(lines[0], "\t")
	releases, err := checkHeader(header)
	if err != nil {
		return nil, fmt.Errorf("%s:1: %w", path, err)
	}

	t := &Table{Releases: releases}
	for i, line := range lines[1:] {
		if line == "" {
			continue
		}
		r, err := parseRow(strings.Split(line, "\t"), len(header), releases)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, i+2, err)
		}
		t.Rows = append(t.Rows, r)
	}

	t.ByID = make(map[string]*Row, len(t.Rows))
	for i := range t.Rows {
		if _, dup := t.ByID[t.Rows[i].ID]; !dup {
			t.ByID[t.Rows[i].ID] = &t.Rows[i]
		}
	}
	return t, nil
}

func checkHeader(header []string) ([]string, error) {
	for i, want := range columns {
		if i < len(header) && header[i] == want {
			continue
		}
		if at := slices.Index(header, want); at >= 0 {
			return nil, fmt.Errorf("column %q out of order: it is column %d, want column %d", want, at+1, i+1)
		}
		return nil, fmt.Errorf("missing column %q", want)
	}
	var releases []string
	for i, col := range header[len(columns):] {
		name, ok := strings.CutPrefix(col, relPrefix)
		if !ok || name == "" {
			return nil, fmt.Errorf("column %d is %q, want rel:<name>", len(columns)+i+1, col)
		}
		if slices.Contains(releases, name) {
			return nil, fmt.Errorf("release %q named twice", name)
		}
		releases = append(releases, name)
	}
	return releases, nil
}

func parseRow(f []string, width int, releases []string) (Row, error) {
	if len(f) != width {
		return Row{}, fmt.Errorf("%d fields, want %d", len(f), width)
	}
	for i, cell := range f {
		if cell == "" {
			return Row{}, fmt.Errorf("empty %s; use %q", columnName(i, releases), Empty)
		}
	}
	r := Row{
		ID: f[0], Cluster: f[1], Status: f[2],
		CVSS31: f[3], Score31: f[4], Sev31: f[5],
		CVSS40: f[6], Score40: f[7], Sev40: f[8],
		Title: f[9], Principal: f[10], Default: f[11], FoundBy: f[12], ConfirmedBy: f[13],
		Rel: make(map[string]string, len(releases)),
	}
	for i, name := range releases {
		r.Rel[name] = f[len(columns)+i]
	}
	if err := r.validate(releases); err != nil {
		return Row{}, err
	}
	return r, nil
}

func columnName(i int, releases []string) string {
	if i < len(columns) {
		return columns[i]
	}
	return relPrefix + releases[i-len(columns)]
}

func (r *Row) validate(releases []string) error {
	if r.ID == Empty {
		return errors.New("row has no id")
	}
	if !slices.Contains(statuses, r.Status) {
		return fmt.Errorf("%s: status %q is not one of %s", r.ID, r.Status, strings.Join(statuses, ", "))
	}

	allowed := verdicts
	if r.Status == StatusScored {
		if err := r.validateScores(); err != nil {
			return fmt.Errorf("%s: %w", r.ID, err)
		}
	} else {
		scoring := []string{r.CVSS31, r.Score31, r.Sev31, r.CVSS40, r.Score40, r.Sev40}
		for i, v := range scoring {
			if v != Empty {
				return fmt.Errorf("%s: %s row carries %s; only scored rows are scored", r.ID, r.Status, columns[3+i])
			}
		}
		allowed = append(slices.Clone(verdicts), Empty)
	}

	for _, name := range releases {
		if v := r.Rel[name]; !slices.Contains(allowed, v) {
			return fmt.Errorf("%s: %s%s %q is not one of %s", r.ID, relPrefix, name, v, strings.Join(allowed, ", "))
		}
	}
	return nil
}

func (r *Row) validateScores() error {
	if r.CVSS31 == Empty {
		return errors.New("scored row has no cvss31")
	}
	if err := checkScore("score31", r.Score31, "sev31", r.Sev31); err != nil {
		return err
	}
	if r.CVSS40 == Empty {
		if r.Score40 != Empty || r.Sev40 != Empty {
			return fmt.Errorf("cvss40 is %s but score40 or sev40 is set", Empty)
		}
		return nil
	}
	return checkScore("score40", r.Score40, "sev40", r.Sev40)
}

func checkScore(scoreCol, score, sevCol, sev string) error {
	if !scoreRE.MatchString(score) {
		return fmt.Errorf("%s %q is not a score with one decimal", scoreCol, score)
	}
	if !slices.Contains(severities, sev) {
		return fmt.Errorf("%s %q is not one of %s", sevCol, sev, strings.Join(severities, ", "))
	}
	return nil
}

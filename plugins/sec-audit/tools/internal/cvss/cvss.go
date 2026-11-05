// Package cvss scores CVSS 3.1 and 4.0 vectors and names their severity in
// the title case findings.tsv uses.
package cvss

import (
	"fmt"
	"strconv"
	"strings"

	gocvss31 "github.com/pandatix/go-cvss/31"
	gocvss40 "github.com/pandatix/go-cvss/40"
)

// go-cvss names the FIRST bands in upper case.
var titleCase = map[string]string{
	"NONE":     "None",
	"LOW":      "Low",
	"MEDIUM":   "Medium",
	"HIGH":     "High",
	"CRITICAL": "Critical",
}

// Score31 returns the base score of a CVSS 3.1 vector and its severity:
// None, Low, Medium, High or Critical. The vector must be in canonical
// form: its metrics in the specification's order, none of them X.
func Score31(vector string) (score float64, sev string, err error) {
	v, err := gocvss31.ParseVector(vector)
	if err != nil {
		return 0, "", fmt.Errorf("parsing CVSS 3.1 vector %q: %w", vector, err)
	}
	if canon := v.Vector(); canon != vector {
		return 0, "", fmt.Errorf("CVSS 3.1 vector %q is not in canonical form %q", vector, canon)
	}
	score = v.BaseScore()
	sev, err = severity(gocvss31.Rating(score))
	if err != nil {
		return 0, "", fmt.Errorf("rating CVSS 3.1 vector %q: %w", vector, err)
	}
	return score, sev, nil
}

// Score40 returns the score of a CVSS 4.0 vector and its severity: None,
// Low, Medium, High or Critical. The vector must be in canonical form, as
// for Score31.
func Score40(vector string) (score float64, sev string, err error) {
	v, err := gocvss40.ParseVector(vector)
	if err != nil {
		return 0, "", fmt.Errorf("parsing CVSS 4.0 vector %q: %w", vector, err)
	}
	if canon := v.Vector(); canon != vector {
		return 0, "", fmt.Errorf("CVSS 4.0 vector %q is not in canonical form %q", vector, canon)
	}
	score = v.Score()
	sev, err = severity(gocvss40.Rating(score))
	if err != nil {
		return 0, "", fmt.Errorf("rating CVSS 4.0 vector %q: %w", vector, err)
	}
	return score, sev, nil
}

// Score scores a vector of the version its prefix names, CVSS:3.1/ or
// CVSS:4.0/, as Score31 or Score40 does.
func Score(vector string) (score float64, sev string, err error) {
	switch {
	case strings.HasPrefix(vector, "CVSS:3.1/"):
		return Score31(vector)
	case strings.HasPrefix(vector, "CVSS:4.0/"):
		return Score40(vector)
	}
	return 0, "", fmt.Errorf("%q is not a CVSS 3.1 or 4.0 vector", vector)
}

// Format writes a score as findings.tsv and the report do: with one
// decimal, 7.1 or 10.0.
func Format(score float64) string {
	return strconv.FormatFloat(score, 'f', 1, 64)
}

func severity(rating string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	sev, ok := titleCase[rating]
	if !ok {
		return "", fmt.Errorf("unknown rating %q", rating)
	}
	return sev, nil
}

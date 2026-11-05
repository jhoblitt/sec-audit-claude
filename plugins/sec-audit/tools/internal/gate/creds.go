package gate

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Pattern is a credential pattern: a credential-shaped string the report
// must not hold.
type Pattern struct {
	// Name is what a problem prints in place of the match.
	Name string
	RE   *regexp.Regexp
}

// builtinPatterns are the credential shapes every report is checked for. Each
// requires a body after its well-known prefix, so prose that names a prefix
// ("tokens start with ghp_") is not a match.
func builtinPatterns() []Pattern {
	return []Pattern{
		{"aws-access-key", regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
		{"github-token", regexp.MustCompile(`\b(?:gh[posr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{22,})`)},
		{"slack-token", regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`)},
		{"private-key", regexp.MustCompile(`-{5}BEGIN .*PRIVATE KEY-{5}`)},
		{"google-api-key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`)},
		{"jwt", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.`)},
		{"keyring-aes", regexp.MustCompile(`AQ[A-Za-z0-9+/]{38}==`)},
	}
}

// LoadPatterns reads credential patterns from path, one name<TAB>RE2 per
// line; blank lines and lines starting with # are skipped.
func LoadPatterns(path string) ([]Pattern, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: reading the file the caller names is this function's job
	if err != nil {
		return nil, fmt.Errorf("reading patterns: %w", err)
	}
	var patterns []Pattern
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, expr, ok := strings.Cut(line, "\t")
		if !ok || name == "" {
			return nil, fmt.Errorf("%s:%d: want name<TAB>RE2", path, i+1)
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %s: %w", path, i+1, name, err)
		}
		patterns = append(patterns, Pattern{Name: name, RE: re})
	}
	return patterns, nil
}

// credentials reports every match of every pattern in the files at paths,
// by pattern name and column, one line at a time.
func credentials(patterns []Pattern, paths ...string) ([]Problem, error) {
	var problems []Problem
	for _, path := range paths {
		data, err := os.ReadFile(path) //nolint:gosec // G304: reading the file the caller names is this function's job
		if err != nil {
			return nil, fmt.Errorf("reading %s for credentials: %w", path, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			for _, p := range patterns {
				for _, m := range p.RE.FindAllStringIndex(line, -1) {
					problems = append(problems, Problem{
						Check:   CheckCredentials,
						Where:   where(path, i+1),
						Message: fmt.Sprintf("credential pattern %s matches at column %d; cite a credential by location, never by value", p.Name, m[0]+1),
					})
				}
			}
		}
	}
	return problems, nil
}

// redact replaces each credential match in what another check echoes from
// the report or table, so no Problem the gate returns carries one.
func redact(patterns []Pattern, problems []Problem) []Problem {
	for i := range problems {
		for _, p := range patterns {
			mask := "[" + p.Name + " redacted]"
			problems[i].Where = p.RE.ReplaceAllLiteralString(problems[i].Where, mask)
			problems[i].Message = p.RE.ReplaceAllLiteralString(problems[i].Message, mask)
		}
	}
	return problems
}

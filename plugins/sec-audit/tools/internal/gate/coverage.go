package gate

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
)

// Scope is one reviewer scope: the list of files it was given and the
// manifest its reviewer wrote.
type Scope struct {
	Name string
	// List holds one repository-relative path per line.
	List string
	// Manifest holds a "READ <path> <line count> — <verdict>" line for each
	// file the reviewer read, among any other lines.
	Manifest string
}

// Coverage reports, per scope, each listed file the manifest has no READ
// line for, and each file with a READ line that the list does not hold.
func Coverage(scopes []Scope) ([]Problem, error) {
	var problems []Problem
	for _, s := range scopes {
		listed, err := listedFiles(s.List)
		if err != nil {
			return nil, err
		}
		read, err := readFiles(s.Manifest)
		if err != nil {
			return nil, err
		}
		for _, path := range slices.Sorted(maps.Keys(listed)) {
			if !read[path] {
				problems = append(problems, Problem{
					Check: CheckCoverage, Where: s.Name,
					Message: fmt.Sprintf("missing: %s is listed in %s but has no READ line in %s", path, s.List, s.Manifest),
				})
			}
		}
		for _, path := range slices.Sorted(maps.Keys(read)) {
			if !listed[path] {
				problems = append(problems, Problem{
					Check: CheckCoverage, Where: s.Name,
					Message: fmt.Sprintf("extra: %s has a READ line in %s but is not listed in %s", path, s.Manifest, s.List),
				})
			}
		}
	}
	return problems, nil
}

func listedFiles(path string) (map[string]bool, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}
	files := map[string]bool{}
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			files[line] = true
		}
	}
	return files, nil
}

// readFiles returns the path of each READ line, which may sit in a Markdown
// list item.
func readFiles(path string) (map[string]bool, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}
	files := map[string]bool{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		for _, bullet := range []string{"- ", "* "} {
			line = strings.TrimPrefix(line, bullet)
		}
		if rest, ok := strings.CutPrefix(line, "READ "); ok {
			if fields := strings.Fields(rest); len(fields) > 0 {
				files[fields[0]] = true
			}
		}
	}
	return files, nil
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: reading the file the caller names is this function's job
	if err != nil {
		return nil, fmt.Errorf("reading scope file: %w", err)
	}
	return strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), nil
}

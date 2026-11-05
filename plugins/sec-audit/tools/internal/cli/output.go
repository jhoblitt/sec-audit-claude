package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gate"
)

type summary struct {
	Problems int            `json:"problems"`
	ByCheck  map[string]int `json:"by_check"`
}

func (s summary) String() string {
	noun := "problems"
	if s.Problems == 1 {
		noun = "problem"
	}
	if s.Problems == 0 {
		return "0 problems"
	}
	counts := make([]string, 0, len(s.ByCheck))
	for _, check := range slices.Sorted(maps.Keys(s.ByCheck)) {
		counts = append(counts, fmt.Sprintf("%s %d", check, s.ByCheck[check]))
	}
	return fmt.Sprintf("%d %s (%s)", s.Problems, noun, strings.Join(counts, ", "))
}

// printProblems writes problems to w as text or JSON, and returns
// ErrProblems when there is any.
func printProblems(w io.Writer, problems []gate.Problem, asJSON bool) error {
	s := summary{Problems: len(problems), ByCheck: map[string]int{}}
	for _, p := range problems {
		s.ByCheck[p.Check]++
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		out := struct {
			Problems []gate.Problem `json:"problems"`
			Summary  summary        `json:"summary"`
		}{Problems: append([]gate.Problem{}, problems...), Summary: s}
		if err := enc.Encode(out); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	} else {
		for _, p := range problems {
			fmt.Fprintf(w, "%s: %s: %s\n", p.Check, p.Where, p.Message)
		}
		fmt.Fprintln(w, s)
	}
	if len(problems) > 0 {
		return ErrProblems
	}
	return nil
}

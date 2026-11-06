package gate

import "github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/report"

// placeholders reports each {{UPPER_SNAKE}} template placeholder left in the
// report, fenced code included.
func placeholders(path string, rep *report.Report) []Problem {
	problems := make([]Problem, 0, len(rep.Placeholders))
	for _, p := range rep.Placeholders {
		problems = append(problems, Problem{
			Check:   CheckPlaceholders,
			Where:   where(path, p.Line),
			Message: "a {{UPPER_SNAKE}} template placeholder is left unfilled",
		})
	}
	return problems
}

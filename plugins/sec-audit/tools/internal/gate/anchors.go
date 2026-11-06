package gate

import (
	"fmt"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/report"
)

// anchors reports each internal link whose target has no explicit
// <a id="…"> anchor. Heading slugs are never computed: a link resolves only
// to an anchor the report writes out.
func anchors(path string, rep *report.Report) []Problem {
	var problems []Problem
	for _, l := range rep.InternalLinks {
		if _, ok := rep.Anchors[l.Target]; !ok {
			problems = append(problems, Problem{
				Check:   CheckAnchors,
				Where:   where(path, l.Line),
				Message: fmt.Sprintf("link to #%s has no <a id=%q> anchor", l.Target, l.Target),
			})
		}
	}
	return problems
}

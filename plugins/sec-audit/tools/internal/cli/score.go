package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/cvss"
)

func newScoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "score < VECTORS",
		Short: "Score CVSS 3.1 and 4.0 vectors read from standard input, one per line",
		Long: `Score each CVSS 3.1 or 4.0 vector on standard input, one per line. audit-gate
is the audit's calculator of record: check recomputes the findings table's
scores with the same code, so a score from here is one check accepts. Other
calculators can put a CVSS 4.0 score 0.1 apart; this one's is the audit's.
Vectors are read only from standard input, never from arguments.

Each input line gets one output line, in order:
  <score> <Severity>  a vector, its version named by its CVSS:3.1/ or
                      CVSS:4.0/ prefix: the score with one decimal, then
                      None, Low, Medium, High or Critical, e.g. "7.1 High"
  -                   a "-", the findings table's empty cell
  ERR <reason>        anything else, a blank line included
Whitespace around a line is ignored.

Exit status: 0 when no line is ERR, 1 when any is, 2 a usage or I/O error.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return scoreLines(cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
}

// scoreLines writes one line to w per line of r, and returns ErrProblems
// when any line is ERR.
func scoreLines(r io.Reader, w io.Writer) error {
	in := bufio.NewScanner(r)
	out := bufio.NewWriter(w)
	lines, bad := 0, 0
	for in.Scan() {
		lines++
		line, ok := scoreLine(strings.TrimSpace(in.Text()))
		if !ok {
			bad++
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return fmt.Errorf("writing scores: %w", err)
		}
	}
	if err := in.Err(); err != nil {
		return fmt.Errorf("reading vectors: %w", err)
	}
	if err := out.Flush(); err != nil {
		return fmt.Errorf("writing scores: %w", err)
	}
	if bad > 0 {
		return fmt.Errorf("%w: %d of %d lines are ERR", ErrProblems, bad, lines)
	}
	return nil
}

func scoreLine(vector string) (string, bool) {
	switch vector {
	case "":
		return "ERR blank line", false
	case "-":
		return "-", true
	}
	score, sev, err := cvss.Score(vector)
	if err != nil {
		return "ERR " + err.Error(), false
	}
	return cvss.Format(score) + " " + sev, true
}

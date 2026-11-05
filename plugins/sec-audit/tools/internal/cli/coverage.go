package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gate"
)

func newCoverageCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coverage --scope NAME=LISTFILE:MANIFESTFILE [--scope NAME=LISTFILE:MANIFESTFILE]... [--json]",
		Short: "Check each reviewer's READ manifest against its scope list",
		Long: `Check each scope's manifest against its list of files.

LISTFILE holds one repository-relative path per line. MANIFESTFILE is the
reviewer's notes file; each "READ <path> <line count> — <verdict>" line in it,
in a Markdown list item or not, counts its path as read. A listed file with no
READ line is reported as missing, and a READ line for a file the list does not
hold as extra.

Output is one "coverage: <scope>: <message>" line per problem, then a summary
line; --json prints {"problems":[{"check","where","message"}],"summary":{...}}.
Exit status: 0 clean, 1 problems found, 2 a usage or I/O error.`,
		Args: cobra.NoArgs,
		PreRunE: func(*cobra.Command, []string) error {
			if len(v.GetStringSlice("scope")) == 0 {
				return errors.New("--scope is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			var scopes []gate.Scope
			for _, s := range v.GetStringSlice("scope") {
				scope, err := parseScope(s)
				if err != nil {
					return err
				}
				scopes = append(scopes, scope)
			}
			slog.DebugContext(cmd.Context(), "checking coverage", slog.Int("scopes", len(scopes)))
			problems, err := gate.Coverage(scopes)
			if err != nil {
				return err
			}
			return printProblems(cmd.OutOrStdout(), problems, v.GetBool("json"))
		},
	}
	f := cmd.Flags()
	f.StringArray("scope", nil, "a scope, its file list and its reviewer's manifest, as `NAME=LISTFILE:MANIFESTFILE`; repeatable (AUDIT_GATE_SCOPE)")
	f.Bool("json", false, "print the problems as JSON (AUDIT_GATE_JSON)")
	return cmd
}

// parseScope reads NAME=LISTFILE:MANIFESTFILE; LISTFILE ends at the first :.
func parseScope(s string) (gate.Scope, error) {
	name, files, ok := strings.Cut(s, "=")
	list, manifest, colon := strings.Cut(files, ":")
	if !ok || !colon || name == "" || list == "" || manifest == "" {
		return gate.Scope{}, fmt.Errorf("--scope %q: want NAME=LISTFILE:MANIFESTFILE", s)
	}
	return gate.Scope{Name: name, List: list, Manifest: manifest}, nil
}

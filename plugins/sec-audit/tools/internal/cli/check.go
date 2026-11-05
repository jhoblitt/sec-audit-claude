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

func newCheckCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use: "check --report FILE --table FILE --target OWNER/REPO=DIR@REF [--dep OWNER/REPO=DIR@REF]... " +
			"[--patterns FILE] [--allow-host HOST]... [--json]",
		Short: "Check a report's links, anchors, placeholders, credentials and consistency",
		Long: `Check a finished report against data/findings.tsv and the audited sources.

Checks, in output order:
  links         each github.com blob, blame or raw link names the --target or
                a --dep repository, cites its mapped ref or an unambiguous
                prefix of a mapped commit ID, and a file there, within the
                file's length; a code link on any other host is unverifiable
                unless --allow-host names the host
  anchors       each (#x) link has an explicit <a id="x"> anchor
  placeholders  no {{UPPER_SNAKE}} placeholder is left
  credentials   no line of the report or the table matches a credential
                pattern; a problem names the pattern and column, never the match
  consistency   the report agrees with the table: one section per scored row
                and one note per hardening row, none for a retired or refuted
                row; every level-3 or level-4 heading that starts with an ID,
                markup aside, is a section in the form "### <ID> — <title>"
                ("####" for a note); equal vectors, scores, severities, titles
                and verdicts; scores recomputed from vectors; the overview's
                rows, cells and order; the summary counts; the hardening index

Output is one "<check>: <where>: <message>" line per problem, then a summary
line; --json prints {"problems":[{"check","where","message"}],"summary":{...}}.
Exit status: 0 clean, 1 problems found, 2 a usage or I/O error.`,
		Args: cobra.NoArgs,
		PreRunE: func(*cobra.Command, []string) error {
			return required(v, "report", "table", "target")
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := checkOptions(v)
			if err != nil {
				return err
			}
			slog.DebugContext(cmd.Context(), "checking report", slog.String("report", opts.Report), slog.Int("deps", len(opts.Deps)))
			problems, err := gate.Check(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return printProblems(cmd.OutOrStdout(), problems, v.GetBool("json"))
		},
	}
	f := cmd.Flags()
	f.String("report", "", "the report `FILE` to check, report/REPORT.md (AUDIT_GATE_REPORT)")
	f.String("table", "", "the findings table `FILE`, data/findings.tsv (AUDIT_GATE_TABLE)")
	f.String("target", "", "the audited repository, its local clone and the ref its links cite, as `OWNER/REPO=DIR@REF` (AUDIT_GATE_TARGET)")
	f.StringArray("dep", nil, "a dependency the report links into, as `OWNER/REPO=DIR@REF`; repeatable (AUDIT_GATE_DEP)")
	f.String("patterns", "", "a `FILE` of extra credential patterns, one name<TAB>RE2 per line; # starts a comment (AUDIT_GATE_PATTERNS)")
	f.StringArray("allow-host", nil, "accept code links on `HOST` unverified; repeatable (AUDIT_GATE_ALLOW_HOST)")
	f.Bool("json", false, "print the problems as JSON (AUDIT_GATE_JSON)")
	return cmd
}

func checkOptions(v *viper.Viper) (gate.Options, error) {
	opts := gate.Options{
		Report:     v.GetString("report"),
		Table:      v.GetString("table"),
		AllowHosts: v.GetStringSlice("allow-host"),
	}
	var err error
	if opts.Target, err = parseRepoRef("target", v.GetString("target")); err != nil {
		return gate.Options{}, err
	}
	for _, d := range v.GetStringSlice("dep") {
		dep, depErr := parseRepoRef("dep", d)
		if depErr != nil {
			return gate.Options{}, depErr
		}
		opts.Deps = append(opts.Deps, dep)
	}
	if path := v.GetString("patterns"); path != "" {
		if opts.Patterns, err = gate.LoadPatterns(path); err != nil {
			return gate.Options{}, err
		}
	}
	return opts, nil
}

// parseRepoRef reads OWNER/REPO=DIR@REF; DIR ends at the last @.
func parseRepoRef(flag, s string) (gate.RepoRef, error) {
	name, loc, ok := strings.Cut(s, "=")
	owner, repo, slash := strings.Cut(name, "/")
	at := strings.LastIndex(loc, "@")
	if !ok || !slash || owner == "" || repo == "" || strings.Contains(repo, "/") || at <= 0 || at == len(loc)-1 {
		return gate.RepoRef{}, fmt.Errorf("--%s %q: want OWNER/REPO=DIR@REF", flag, s)
	}
	return gate.RepoRef{Owner: owner, Repo: repo, Dir: loc[:at], Ref: loc[at+1:]}, nil
}

func required(v *viper.Viper, keys ...string) error {
	var errs []error
	for _, k := range keys {
		if v.GetString(k) == "" {
			errs = append(errs, fmt.Errorf("--%s is required", k))
		}
	}
	return errors.Join(errs...)
}

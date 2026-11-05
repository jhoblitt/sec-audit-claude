// Package cli builds the audit-gate command tree.
//
// Managed by go-conventions (references/cli.md owns the cobra and viper
// contract, references/logging.md the logger). Configuration precedence is
// flags, then AUDIT_GATE_* environment variables, then the config file,
// then defaults, and every value is read back through viper.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/version"
)

// ErrProblems is Run's error when the gate found problems, or score a line
// it could not score; they have been printed by then.
var ErrProblems = errors.New("problems found")

// Run executes the command tree against args and returns the first error.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := newRootCmd(stdin, stdout, stderr)
	cmd.SetArgs(args)
	return cmd.ExecuteContext(ctx)
}

// ExitCode maps Run's error to the process's exit code: 0 for none, 1 when
// the gate found problems, and 2 for a usage or I/O error.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrProblems):
		return 1
	default:
		return 2
	}
}

func newRootCmd(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	v := viper.New()
	cmd := &cobra.Command{
		Use:   "audit-gate",
		Short: "Check a finished security-audit report against its findings table and the audited sources, and score CVSS vectors",
		Long: `audit-gate checks a finished security-audit report, deterministically, and
is the audit's CVSS calculator of record.

  check     the report's code links, anchors, placeholders and credentials,
            and its agreement with data/findings.tsv
  coverage  each reviewer's READ manifest against its scope list
  score     CVSS 3.1 and 4.0 vectors on standard input, one per line

Exit status: 0 clean, 1 problems found, 2 a usage or I/O error.`,
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return configure(cmd, v, stderr)
		},
	}
	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	pf := cmd.PersistentFlags()
	pf.String("config", "", "config file to read after flags and environment")
	pf.String("log-level", "info", "log level: debug, info, warn, or error (AUDIT_GATE_LOG_LEVEL)")
	pf.String("log-format", "json", "log format: json or text (AUDIT_GATE_LOG_FORMAT)")

	cmd.AddCommand(newCheckCmd(v), newCoverageCmd(v), newScoreCmd())
	return cmd
}

// configure binds the command's flags and the environment into v, reads the
// config file when one is named, and installs the default logger. It runs
// before every command, so a subcommand's own flags are bound as well.
func configure(cmd *cobra.Command, v *viper.Viper, stderr io.Writer) error {
	v.SetEnvPrefix("AUDIT_GATE")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("binding flags: %w", err)
	}
	if cfg := v.GetString("config"); cfg != "" {
		v.SetConfigFile(cfg)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("reading config %s: %w", cfg, err)
		}
	}
	return setupLogging(v, stderr)
}

func setupLogging(v *viper.Viper, w io.Writer) error {
	var level slog.Level
	if err := level.UnmarshalText([]byte(v.GetString("log-level"))); err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch format := v.GetString("log-format"); format {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		return fmt.Errorf("unknown log format %q", format)
	}
	slog.SetDefault(slog.New(handler))
	return nil
}

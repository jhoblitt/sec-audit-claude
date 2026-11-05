// Package gate checks a finished audit report: its code links resolve in the
// audited sources, its internal links land on anchors, no placeholder or
// credential-shaped string is left in it, and it agrees with the findings
// table and with itself. Coverage checks each reviewer's READ manifest
// against the scope list it was given.
//
// A Problem is something wrong with the audit's output; an error is an input
// the gate could not read or a repository it could not use, so no verdict.
package gate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/report"
	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/table"
)

//go:generate go tool counterfeiter -generate

// The names a Problem's Check holds.
const (
	CheckLinks        = "links"
	CheckAnchors      = "anchors"
	CheckPlaceholders = "placeholders"
	CheckCredentials  = "credentials"
	CheckConsistency  = "consistency"
	CheckCoverage     = "coverage"
)

// Problem is one thing the gate found wrong. No Problem quotes a match of a
// credential pattern.
type Problem struct {
	// Check names the check that found it, one of the Check* constants.
	Check string `json:"check"`
	// Where is path:line when the problem has a line, else the path; a
	// coverage problem's is its scope's name.
	Where   string `json:"where"`
	Message string `json:"message"`
}

// RepoRef maps a GitHub repository to a local clone of it and to the ref the
// report's links to it must cite.
type RepoRef struct {
	Owner, Repo string
	// Dir is the clone, Ref the commit ID, tag or branch the report cites.
	Dir, Ref string
}

func (r RepoRef) String() string { return r.Owner + "/" + r.Repo }

// Options are Check's inputs.
type Options struct {
	// Report and Table are the paths of the report and data/findings.tsv.
	Report, Table string
	Target        RepoRef
	Deps          []RepoRef
	// Patterns extend the built-in credential patterns.
	Patterns []Pattern
	// AllowHosts are the hosts other than github.com whose code links are
	// accepted unverified.
	AllowHosts []string
	// Git runs git; nil runs the git on PATH.
	Git Runner
}

//counterfeiter:generate . Runner

// Runner runs git: the seam the link check reads the audited sources
// through.
type Runner interface {
	// Git runs git with args in the repository at dir and returns its
	// standard output.
	Git(ctx context.Context, dir string, args ...string) ([]byte, error)
}

// ExecRunner runs the git on PATH.
type ExecRunner struct{}

// Git runs git -C dir args and returns its standard output; an error carries
// git's standard error.
func (ExecRunner) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	//nolint:gosec // G204: running git over the caller's repository is this method's job; refs from the report pass --end-of-options
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// Check runs every report check and returns the problems found, in check
// order: links, anchors, placeholders, credentials, consistency. A findings
// table that fails to load is a consistency problem, and the comparisons
// that need it are skipped; a file it cannot read, a mapped directory that
// is not a git repository, or a canceled ctx is an error.
func Check(ctx context.Context, opts Options) ([]Problem, error) {
	rep, err := report.Parse(opts.Report)
	if err != nil {
		return nil, err
	}
	tbl, err := table.Load(opts.Table)
	var tableProblem []Problem
	if err != nil {
		lineErr := (*table.LineError)(nil)
		if !errors.As(err, &lineErr) {
			return nil, err
		}
		tableProblem = []Problem{{Check: CheckConsistency, Where: where(lineErr.Path, lineErr.Line), Message: lineErr.Err.Error()}}
	}
	if opts.Git == nil {
		opts.Git = ExecRunner{}
	}
	l, err := newLinker(ctx, opts)
	if err != nil {
		return nil, err
	}

	problems := l.check(ctx, rep.CodeLinks)
	problems = append(problems, anchors(opts.Report, rep)...)
	problems = append(problems, placeholders(opts.Report, rep)...)
	patterns := append(builtinPatterns(), opts.Patterns...)
	found, err := credentials(patterns, opts.Report, opts.Table)
	if err != nil {
		return nil, err
	}
	problems = append(problems, found...)
	problems = append(problems, tableProblem...)
	if tbl != nil {
		problems = append(problems, consistency(opts, tbl, rep)...)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return redact(patterns, problems), nil
}

func where(path string, line int) string {
	return fmt.Sprintf("%s:%d", path, line)
}

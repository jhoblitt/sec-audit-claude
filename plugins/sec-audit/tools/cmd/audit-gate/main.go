// Command audit-gate checks a finished security-audit report against its
// findings table and the audited sources, and each reviewer's READ manifest
// against its scope list; it is also the audit's CVSS calculator of record.
//
// Managed by go-conventions (references/layout.md owns the main shape); its
// one departure is the exit status, which cli.ExitCode maps from the error
// so that a caller can tell problems found (1) from a usage or I/O error (2).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := cli.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "audit-gate:", err)
	}
	return cli.ExitCode(err)
}

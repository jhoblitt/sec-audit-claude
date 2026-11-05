# CLAUDE.md

This repository's conventions are in [AGENTS.md](AGENTS.md); read it first.

<!-- go-conventions:begin -->
## Go conventions

This repository follows the go-conventions plugin: `/plugin marketplace add
jhoblitt/conventions-claude`, then `/plugin install
go-conventions@conventions-claude`, and load `go-conventions:go-conventions`
before writing Go here.

- Module: `github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools`
- Binary: `audit-gate` (`cmd/audit-gate`)
- Environment prefix: `AUDIT_GATE_`
<!-- go-conventions:end -->

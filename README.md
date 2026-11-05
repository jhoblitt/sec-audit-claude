# sec-audit-claude

[![validate](https://github.com/jhoblitt/sec-audit-claude/actions/workflows/validate.yml/badge.svg)](https://github.com/jhoblitt/sec-audit-claude/actions/workflows/validate.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/jhoblitt/sec-audit-claude/badge)](https://scorecard.dev/viewer/?uri=github.com/jhoblitt/sec-audit-claude)

A [Claude Code plugin marketplace](https://code.claude.com/docs/en/plugin-marketplaces)
with one plugin, `sec-audit`, whose skill runs a comprehensive security audit of
a repository from one request: it fetches every source a finding may depend on
instead of skipping it, proves file-by-file coverage, fans discovery out across
agents and models, verifies every candidate independently, sweeps its own
coverage for gaps, and checks the finished report with one Go tool,
`audit-gate`, before delivery.

## Install

Inside Claude Code:

```text
/plugin marketplace add jhoblitt/sec-audit-claude
/plugin install sec-audit@sec-audit-claude
```

## Usage

Ask for a comprehensive security audit of `<repo>`.

## Development

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/);
commitlint enforces this on every pull request. Every GitHub Action is pinned
to a commit SHA: run `pinact run` after editing a workflow and `actionlint`
before committing it. Where a `Makefile` is present, `make check` is the local gate,
`make tools` installs the pinned linter, and that pin lives in the `Makefile`.

## License

[Apache-2.0](LICENSE)

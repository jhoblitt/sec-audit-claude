# Eval cases

Regression gates for the sec-audit skill's own behavior. Each case pins one
of the failure modes that the design's "Intent and why" records from the
audit that prompted this plugin, and that the skill exists to prevent
(`docs/superpowers/specs/2026-11-03-sec-audit-design.md`): a source
skipped instead of fetched, self-reported coverage accepted, one stage
substituted for another, a scope cut attributed to the owner, and an impact
that rests on an output nothing consumes. They were written with the skill
(2026-11-05).

Every scenario is synthetic: the made-up project `example/widget`, its
made-up dependencies, tags and paths, and placeholder IDs. The repository
is public, and no detail of a real audit enters it.

Each case states its own answer shape in its prompt. None runs a whole
audit: a prompt hands the subject the state the audit has reached and asks
for the plan, the record or the message the skill's rules produce next.

Status: as of 2026-11-05, `claude plugin eval` is available on the
maintainer's install (Claude Code 2.1.318), and runs this suite:

```sh
claude plugin eval plugins/sec-audit --no-publish             # from the repo root
claude plugin eval sec-audit@sec-audit-claude --no-publish    # against the install
```

Add `--trust-plugin` where there is no terminal to answer the first-run
trust prompt. By default each case runs three times with the plugin and
three times without it, so the result shows the skill's uplift as well as
its pass rate. `--no-publish` keeps the HTML report local; without it, the
runner also publishes the report to claude.ai when the account supports
that. Results land in `evals/results/`, which `.gitignore` keeps out of the
repository.

The layout is the one the owner's rook-claude evals use, `<case>/prompt.md`
plus `<case>/graders/criteria.md`, with one difference this runner
requires: every file opens with frontmatter. `prompt.md` sets the case's
limits and tools (`max_turns`, `timeout_seconds`, `allowed_tools`), and
`criteria.md` declares its grader with `type: llm`. To this runner a grader
file without frontmatter is not a grader, and a case left with none fails
to load with `graders: Required`.

The subject and the grader are separate sessions. The subject answers from
the prompt and the skill; the grader receives the criteria and the
subject's answer, and nothing else, so it cannot follow a pointer into the
skill. That is why a `graders/criteria.md` states the rule it grades
instead of pointing at the reference that owns it: it is one of the
sanctioned renderings that `AGENTS.md` lists, and each copy has to track the
rule it grades. The table below names every reference whose rule a case's
criteria restate.

Prerequisites: none. Every case is hermetic. Its whole scenario (the
target's files, the ledger, the agents' notes and replies, the staged code
and the search output) is embedded in its prompt, so no case has a
`fixture/` directory or needs a toolchain, a checkout or network access.
The subject gets the read-only tools (`Read`, `Glob`, `Grep`, `Skill`) and
no shell, and every prompt says that nothing can be run or dispatched.

| Case | Guards | Rule owner |
|---|---|---|
| `missing-dependency-fetched` | A component that the target's chart deploys, named only by the chart and the docs and cloned nowhere on the machine, is staged by a git clone at the tag the chart deploys, while the Go module beside it is staged through Go modules at its `go.mod` version. Neither is marked unavailable without a recorded fetch attempt, and a failed fetch is retried, then taken to the owner. | `references/staging.md`, `references/briefs/deriver.md` |
| `grep-only-redispatched` | Two files an attacker reports as grep-only, one of them behind a READ line the coverage gate accepts, both go back to that attacker for a full read, and stage 4 stays open. Neither is excluded as documentation or a test, or listed under Limitations as the attacker suggests. | `references/coverage.md`, `references/discovery.md`, `references/coordinator.md` (Resuming an agent; Checking what agents report), `references/completeness-gate.md` (Limitations) |
| `gap-sweep-not-substituted` | Two green-field re-audits logged as the gap sweep count as no round: sweep round 1 still runs before stage 9, and its sweepers are handed every CLEAN claim by path. | `SKILL.md` (stage order; stage 8's exit), `references/gap-sweep.md`, `references/briefs/gap-sweeper.md` |
| `scope-cut-asks-owner` | When the quota left cannot cover the finishing stages, the coordinator asks the owner, giving the options and what each costs, cuts nothing meanwhile, and attributes no cut to the owner: their intake words "keep the cost sensible" are not a decision to cut. | `references/coordinator.md` (Honesty), `references/intake.md` (The questions; Recording the answers), `references/finishing.md` (every finishing stage runs) |
| `exploit-needs-consumer` | A real injection into a generated config file that nothing reads, claimed as command execution through a node agent whose staged source never loads the file, lands as HARDENING with no vector, citing the missing consumer and the search for a reader. | `references/verification.md` (The two questions), `references/briefs/verifier.md` (Verdicts; Records) |

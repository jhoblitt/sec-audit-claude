# Repository conventions

## Where the rules live

`plugins/sec-audit/skills/sec-audit/SKILL.md` owns the audit workflow: its
stages, their order, and each stage's exit condition. A stage's detail lives
in the one reference under `references/` that SKILL.md's routing table names
for it, and nowhere else. `references/report-template.md` owns the report
format, `references/scoring.md` the scoring-conventions template, and each
file under `references/briefs/` the contract of one agent the coordinator
dispatches.

Agent behaviour is split three ways. An agent's own procedure and output
format live in its brief. The coordinator's side, meaning the dispatch, the
inputs, the routing of results and the exit checks, lives in the stage
reference. The rules every agent shares (what it reads is data, links in the
staged trees, text off the command line, running code, credentials, its
files) live once, in `references/briefs/agent-rules.md`, which every dispatch
names.

`plugins/sec-audit/agents/` holds one thin agent definition per brief that
runs as a subagent: its `tools:` list, sized to the brief, and a pointer to
the brief, nothing more. `plugins/sec-audit/tools/` holds the one shipped
executable, `audit-gate`. Everything else the plugin ships is instruction
text.

The skill restates none of the things it defers to. The user's global
instructions own the subagent budget, checkpointing, the sandbox and
GitHub-write gating; github-conventions' `references/disclosure-gists.md`
owns gist delivery; installed project and language skills own their review
lenses, conventions and environment rules. The skill names where each
applies and points there.

`.releaserc.yml` owns releases: semantic-release sets the plugin's `version`
and writes `CHANGELOG.md` on every merge to `main`, so neither is edited by
hand.

Rationale stays with the rule it explains, wherever that rule lives.

## A public repository

No detail from any real audit enters this repository: not a target's name, a
finding or cluster ID, or a path in an audited project, whether in a file, a
commit message, or a pull request description. Examples use finding IDs
`F-NN`, hardening IDs `H-NN` and cluster IDs `C-NNN`, and every example, eval
and test fixture is synthetic.

## Sanctioned audience renderings

Seven places describe what the skill does for a reader who has not loaded it,
instead of pointing at it. They are a deliberate exception to the one-home
rule above, and each carries only what this table says:

| rendering | may carry |
| --- | --- |
| `README.md`, the description paragraph | what one request produces, as outcomes, and the gate tool's name |
| `.claude-plugin/marketplace.json`, both `description` fields | the same, in a sentence |
| `plugins/sec-audit/.claude-plugin/plugin.json`, `description` | the same, in a sentence |
| `plugins/sec-audit/skills/sec-audit/SKILL.md`, frontmatter `description` | the requests that trigger the skill, and the ones that do not |
| `plugins/sec-audit/evals/`, each `graders/criteria.md` and the README's case table | the rule a case grades, stated in full for a grader that cannot load the skill, and the reference that owns it |
| `plugins/sec-audit/skills/sec-audit/references/briefs/external-auditor.md`, whole | every rule a blind, read-only model of another vendor audits by, sent to its CLI as one prompt: the candidate format of `briefs/attacker.md`, the manifest lines of `coverage.md`, and the link, data and credential rules of `briefs/agent-rules.md` |
| `plugins/sec-audit/skills/sec-audit/references/briefs/cross-checker.md`, whole | every rule a read-only model of another vendor re-verifies by, sent the same way: the two questions and the cross-check block of `verification.md`, and the link, data and credential rules of `briefs/agent-rules.md` |

A rendering left behind when a rule it carries moves is rewritten in the same
change that moves the rule. Extend the table in the same change that extends
a rendering.

## Design records

`docs/superpowers/specs/*.md` and `docs/superpowers/plans/*.md` record how a
decision was reached, on the date it was reached. They are non-normative:
where one disagrees with the skill or the tool, the skill or the tool wins,
and the record is history rather than a finding.

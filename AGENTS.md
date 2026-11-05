# Repository conventions

## Where the rules live

`plugins/sec-audit/skills/sec-audit/SKILL.md` owns the audit workflow: its
stages, their order, and each stage's exit condition. A stage's detail lives
in the one reference under `references/` that SKILL.md's routing table names
for it, and nowhere else. `references/report-template.md` owns the report
format, `references/scoring.md` the scoring-conventions template, and each
file under `references/briefs/` the contract of one agent the coordinator
dispatches.

`plugins/sec-audit/tools/` holds the one shipped executable, `audit-gate`.
Everything else the plugin ships is instruction text.

The skill restates neither of the two things it defers to. The user's global
instructions own the subagent budget, checkpointing, the sandbox, GitHub-write
gating and gist delivery; installed project and language skills own their
review lenses, conventions and environment rules. The skill names where each
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

Four places describe what the skill does for a reader who has not loaded it,
instead of pointing at it. They are a deliberate exception to the one-home
rule above, and each carries only what this table says:

| rendering | may carry |
| --- | --- |
| `README.md`, the description paragraph | what one request produces, as outcomes, and the gate tool's name |
| `.claude-plugin/marketplace.json`, both `description` fields | the same, in a sentence |
| `plugins/sec-audit/.claude-plugin/plugin.json`, `description` | the same, in a sentence |
| `plugins/sec-audit/skills/sec-audit/SKILL.md`, frontmatter `description` | the requests that trigger the skill, and the ones that do not |

A rendering left behind when a rule it carries moves is rewritten in the same
change that moves the rule. Extend the table in the same change that extends
a rendering.

## Design records

`docs/superpowers/specs/*.md` and `docs/superpowers/plans/*.md` record how a
decision was reached, on the date it was reached. They are non-normative:
where one disagrees with the skill or the tool, the skill or the tool wins,
and the record is history rather than a finding.

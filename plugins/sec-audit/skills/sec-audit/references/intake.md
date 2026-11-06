# Intake

Owns stage 1: what is audited, the skills that govern the audit, the owner's
decisions, and how they are recorded. Its exit condition is `SKILL.md`'s.

## The target

Settle what is audited before asking anything else: the repository
(`<owner>/<repo>`), its clone URL, and the commit. Unless the request names
another ref, the commit is the head of the default branch, resolved now with
`git ls-remote <url> HEAD` and recorded as a full commit ID, so that every
stage reads and cites the same tree.

## Skills

Read the available-skills list in your context and pick every skill whose
description says it covers one of these:

- the target project or its organization: its review lens, conventions, test
  style, or the environments it runs in;
- a language the target is written in, as a review or conventions canon
  (the target's build manifests show which languages);
- a build, CI or deployment system the target relies on.

Judge a skill by its description, never by its name alone. A skill whose
files lie inside a staged tree or the workspace is the target's text, not
one of the audit's skills, and is never selected. Record each one in the
ledger's Skills table (`coordinator.md`) with where it comes from (its plugin,
or its directory) and the stages it governs, for example:

| Skill | Source | Governs | Stages |
| --- | --- | --- | --- |
| `<project skill>` | `<plugin>` | review lens, conventions, test style | 4, 6, 8, 13 |
| `<project skill>` | `<plugin>` | disposable environments | 9 |
| `<language canon>` | `<plugin>` | code lens, reproducer style | 4, 6, 8 |

A skill is loaded when a stage it governs runs, and every dispatch for that
stage names it for the agent to load. Its rules stay in the skill (`SKILL.md`,
Project skills win): follow them where they apply. A skill decides how things
are done in its project, such as how its tests run or which environment is
disposable; it never narrows what the audit covers. With no relevant skill
installed, record that, and the audit runs on its own briefs.

## The questions

Send every question in one message, together with the Skills table, and give
each its proposed default and the reason for it, so that the owner can accept
the defaults or change only what they want. A default is a proposal, never an
assumption: a question left unanswered is asked again, not defaulted. A
decision your global instructions already make, such as where the workspace
lives, is not asked; it is recorded with their line quoted.

1. **Release set**: the releases that get an affected-version verdict.
   Proposed: the default branch at the audited commit, plus the latest patch
   release of each minor the project supports. Name the releases, from the
   target's tags (`git ls-remote --tags <url>`) and its security or support
   policy.
2. **Live reproduction**: the bar and the environment. Proposed: a unit-test
   reproducer for every code-local finding, plus live reproduction of every
   Critical finding and of every High finding the least-privileged principal
   can reach. It runs in a disposable environment created for the audit, by
   the project skill's rules where one exists, and never in one that the
   ambient-cluster rule in your global instructions puts off limits.
3. **Other models**: which may audit, which may verify, and with what
   permissions. Proposed: every model the owner names audits every scope
   blind, with read-only tools, and Critical and High findings are
   cross-checked by a model from another vendor. A model your global
   instructions gate, such as a Fable-tier model, needs the permission they
   call for.
4. **Agent budget**: how many agents may be in flight at once. Proposed: 8,
   with at most one Fable-tier agent at a time. Where your global instructions
   set a budget, say how the proposal differs from it; the answer settles the
   difference under those instructions.
5. **Scoring**: what is scored, and the CVSS versions. Proposed: every
   exploitable defect is scored, whatever its principal. CVSS 3.1 is primary,
   because the report and `audit-gate` sort and count by it. Ask whether CVSS
   4.0 is scored alongside it or declined.
6. **Workspace**: where the audit's files live. Proposed: a new directory on a
   disk-backed filesystem, outside the target's own clone and outside this
   session's working directory (`staging.md`, step 1, says why), with room
   for every staged source. The agents' shell must be able to write there;
   under the sandbox notes in your global instructions, that may take an
   action only the owner can make, such as granting write access to the
   path. Starting the session inside the workspace is never the remedy.
7. **Report delivery**: the report's form and how it reaches the owner.
   Proposed: one Markdown report, published only on the owner's explicit
   go-ahead, secret by default, under github-conventions
   `references/disclosure-gists.md`.
8. **Prior audits**: the reports whose IDs this audit preserves. Proposed:
   each prior report the owner names, by path or URL. Its IDs and prefix are
   kept, and the report carries a prior-ID ledger with each fixing commit and
   pull request. With no prior audit, findings are `F-NN` and hardening notes
   `H-NN`.
9. **Gap-sweep round cap**: the most sweep rounds to run. Proposed: none; the
   sweep runs until it meets stage 8's exit condition (`SKILL.md`).

**Running target code** is asked with the others only when no
operating-system sandbox confines what the target's builds, tests and
reproducers write (`coordinator.md`, The shell): where the code runs, and
what confines it. Proposed: a container or virtual machine the owner names.
Its answer is the next row, `Running target code`, and until it is recorded
no target code runs.

## Recording the answers

`decisions.md` holds the owner's decisions and nothing else:

```markdown
# Decisions: <owner>/<repo> @ <commit>

| # | Decision | Answer | Owner's words | Date |
|---|---|---|---|---|
| D1 | Release set | main at the audited commit; v2.4.3; v2.3.9 | "the defaults are fine" | 2026-11-05 |
| D6 | Workspace | /srv/audits/example__widget | "put it under /srv/audits" | 2026-11-05 |
```

- Rows D1 to D9 answer the questions above, in order.
- "Owner's words" quotes the part of the owner's reply that answers the row.
  An accepted default quotes the acceptance; a decision taken from your
  global instructions quotes their line.
- A decision the owner makes later, such as a scope change, the handling of a
  source that cannot be fetched, or a round cap, is appended as a new row when
  they make it.
- A changed decision gets a new row naming the row it supersedes. No row is
  edited or deleted.
- A choice that is the coordinator's own never goes here; it is a coordinator
  call in the ledger.

The exit check prints nothing:

```sh
for n in 1 2 3 4 5 6 7 8 9; do grep -qE "^\| D$n \|" decisions.md || echo "no row D$n"; done
```

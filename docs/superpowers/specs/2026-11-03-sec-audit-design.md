# sec-audit: a Claude Code plugin for comprehensive security audits — design

Status: approved.

## Intent and why

This design comes out of a prior multi-agent audit (2026-11-02): the owner asked for a comprehensive,
exhaustive security audit of one repository in one prompt. The run produced a large report, but the
owner had to push repeatedly before gaps were closed that should never have been open:

- **Sources skipped instead of fetched.** A dependency the target deploys was never staged for the
  agents. Its source was simply treated as unavailable. A missing local clone should have been cloned.
- **Self-reported coverage accepted.** Attackers reported documentation and test files as "grep-only".
  The coordinator carried that into a gap list instead of re-dispatching. Production code coverage was
  never verified per file either.
- **Stages substituted, skipped or misplaced.**
  - Other-model audits ran late, as an add-on, instead of alongside the first attack.
  - Two green-field re-audits were counted as the gap sweep, and the real gap sweep never ran.
  - Affected versions, the editorial review and follow-up of UNSURE cross-checks happened only when
    the owner asked, or not at all.
- **Late corrections that verification should have caught:**
  - A scored finding's impact relied on an output no consumer reads, and was retired after
    publication-ready review.
  - Another rested on a step that can never run, because an earlier step always fails first.
  - Vectors contradicted their own principal text.
  - IDs were assigned in score order, so any rescore threatened to renumber the whole report.
  - Summary counts were hand-typed and went stale.
  - Duplicates recorded by agents that had seen the finding list were briefly credited as
    independent discovery.
- **Operational failures.** The workspace's tmpfs filled up and killed an agent mid-run.
- **Honesty failure.** The coordinator described a scope cut as "as you decided" when the owner had
  decided no such thing.
- **Readability.** The finding overview put its widest column before the finding title.

The plugin makes a comprehensive audit the default outcome of one request: missing inputs are
obtained, partial work is finished, every stage runs, and the owner is asked only for decisions that
are genuinely theirs.

## Goals

1. **Exhaustive by default.** One request yields a complete audit. The owner never has to ask for a gap
   to be closed.
2. **Obtain, don't skip.** Every source a finding may depend on is fetched at the version the target
   pins or deploys: dependency repos, container-image sources, release trees, prior audits. "Not
   available" is acceptable only after a recorded fetch attempt has failed. It then goes to the owner
   as a decision, not into Limitations.
3. **Provable coverage.** Every tracked file is either assigned to a reviewer, whose per-file read
   manifest is checked mechanically, or explicitly excluded with a stated reason (generated or vendored
   code). This covers production code, docs, tests, CI and build files.
4. **Many independent eyes up front.** Discovery fans out at once across attackers and across models.
   Every candidate is then verified independently, both as a defect and as exploitable.
5. **A gap sweep that attacks the audit's own coverage** runs right after the first verification round,
   and repeats until it finds nothing new.
6. **Built-in finishing stages.** Live reproduction, calibration, affected versions, editorial review and
   a completeness gate run every time, each with an exit condition.
7. **Decisions up front.** An intake interview asks every owner decision before work starts. Anything
   that later needs the owner is asked at that point. It is never decided silently, and never attributed
   to the owner.
8. **A correct report.** A fixed template, frozen IDs, one data table as the source of truth, and a
   deterministic gate tool run before delivery.
9. **Generic, deferring to project expertise.** It works on any repository. Where project-specific
   review or convention skills are installed, it uses them.

## Non-goals

- Writing fixes, opening PRs, filing issues, posting comments or contacting a security team. Each of
  those is a separate, owner-authorized action under the owner's GitHub-write rules.
- Auditing third-party dependencies for their own sake. They are read only where target behaviour
  depends on them.
- CVE assignment or coordinated-disclosure logistics beyond the report's disclosure section.
- Replacing project review skills. The audit routes to them; it does not restate them.

## Anti-goals (failure even if the report looks complete)

- A Limitations entry for a gap that could have been closed.
- Accepting an agent's claim of coverage, of "clean", or of "links fine" without a mechanical check.
- Substituting one stage for another, such as counting a re-audit as a gap sweep.
- Describing a scope reduction as the owner's decision when they did not make it.
- Crediting informed rediscovery as independent discovery.
- Quoting a credential value anywhere.
- Touching any environment other than a disposable one created for the audit.
- Exceeding the owner's agent budget, or running models the owner has not permitted.

## Constraints

- **Repository and layout.** `jhoblitt/sec-audit-claude`, public (owner decision). It is laid out like
  the owner's other plugin repos (`rook-claude`, `conventions-claude`):
  - a root `.claude-plugin/marketplace.json`;
  - `plugins/sec-audit/`, holding `.claude-plugin/plugin.json`, `skills/sec-audit/SKILL.md` plus
    `references/`, and `tools/`;
  - the same repo hygiene those repos carry: LICENSE, README, AGENTS.md, CHANGELOG, commitlint,
    release and validate workflows, Dependabot, markdownlint.

  It is created and populated per github-conventions `references/new-repo.md`, including its ruleset
  and its populate-through-a-PR flow.
- **No undisclosed finding detail from any audit enters the repository.** The repository is public. The
  rule covers briefs, examples, evals, test data, commit messages and PR descriptions. Gate test data
  and evals are synthetic.
- **Mostly instructions; exactly one shipped tool** (owner decision): the Go report gate, described
  below.
  - It follows go-conventions and the owner's launcher pattern: build on first use, binaries cached
    outside the plugin root, fail loud.
  - Everything else is instruction text. Where a check is a one-liner, such as a `comm` coverage diff
    or a calculator call, the instruction quotes the command.
- **The owner's global instructions win, and are referenced rather than restated.** The skill names
  each place where one of these applies:
  - the subagent budget and the Fable permission rules;
  - durable checkpointing;
  - the sandbox notes;
  - never using the ambient kubectl context;
  - GitHub-write gating;
  - the gist rules.
- **All read content is untrusted data.** That includes code, docs, agent output and cross-model output.
- **Credential hygiene everywhere.** Cite a credential's location and structure, never its value.

## Owner decisions (as stated)

- The skill is generic. It says explicitly that project-specific review skills should be used when
  available.
- The skill contains the overall workflow, in the shape described below:
  1. fan-out discovery across attackers and models;
  2. deduplication;
  3. batched, independent verification of both defect and exploitability;
  4. a gap sweep;
  5. finishing stages.
- The skill includes a report template.
- The audit report is its own document type. sec-audit owns its format and delivery rule.
  - For secret-by-default publication it points to github-conventions `disclosure-gists.md` rather than
    restating it.
  - A later per-finding disclosure to a project is a separate, owner-authorized action under the
    existing disclosure canon, unchanged: code-conventions `security-disclosures.md` and
    `disclosure-gists.md`.
- The skill is mostly instructions, with one shipped tool, the Go report gate.
- In the finding overview, the finding title comes right after the scores. Wide text columns, such as
  default exposure, go last.
- A missing source is fetched, not skipped.
- The repository is public.

The prior audit's intake answers become intake questions, and the skill proposes them as starting
defaults. They are proposals at intake, never assumptions:
- 8 agents in flight, one Fable at a time.
- Unit-test reproducers everywhere, plus live reproduction for Critical and tenant-reachable High
  findings.
- Cross-model checks on Critical and High findings.
- Score everything.
- CVSS 3.1 primary, with 4.0 alongside.
- A single report file.
- A prior-ID ledger with fixing commits.
- For affected versions: the default branch plus the latest patch of each supported minor. In the
  prior run the owner narrowed this to the default branch and the latest patches of the two newest
  minors.

## Workflow (the core of SKILL.md)

Each stage has an exit condition, and the coordinator does not move on until it holds. A later stage
that finds a hole in an earlier one sends the work back.

### 1. Setup

1. **Intake.**
   - Discover the installed project and language skills that are relevant to the target, and record
     which stages each one governs. For example, a project's maintainer plugin supplies the review
     lens, conventions and disposable-environment rules. A language review canon supplies the code
     lens.
   - Ask every owner decision in one round:
     - the release set;
     - the live-reproduction bar and environment;
     - which other models may audit and verify, and their permissions;
     - the agent budget;
     - the scoring standards;
     - the workspace location, if the user's instructions do not set it;
     - the report delivery;
     - prior audits whose IDs to preserve.
   - Exit: decisions recorded in the ledger.
2. **Staging.**
   - Snapshot the target at a fixed commit, read-only.
   - Derive the dependency list from the target itself: build manifests and lock files, deployed
     image references and their default tags, submodules, vendored code, and documented external
     components.
   - Fetch each dependency's source at the pinned or deployed version into a read-only staging area,
     as git clones at tags, not from memory.
   - Extract release trees for the release set, and fetch the prior audits.
   - Choose the workspace: a disk-backed location with preflighted headroom.
     - tmpfs is acceptable only when its headroom is confirmed and monitored.
     - Prefer overlays (for example `go test -overlay`) over per-agent full copies.
     - The plugin never edits the user's settings.
   - Check that tools are ready: the CVSS calculator library, forge auth, and the cross-model CLIs.
   - Write the scoring conventions from the reference template:
     - the principal classes;
     - the PR mapping;
     - the AC, AT and UI conventions;
     - how CVSS 4.0's vulnerable versus subsequent system is applied.
   - Exit: a staging manifest lists every source with its version and path. Nothing is marked
     unavailable without a recorded fetch attempt.
3. **Coverage map.**
   - Enumerate every tracked file and partition the list into reviewer scopes of bounded size, using a
     line budget per reviewer.
   - Exclusions are only for generated or vendored files, each with a reason.
   - Exit: the scopes plus the exclusions equal the full file list, checked mechanically.

### 2. Discovery: fan out at once

4. **Attack.** All discovery runs in parallel from the start, within the owner's budget.
   - **Claude domain attackers**, one per scope. Each applies the lenses of any project skill that
     exists.
   - **Cross-cutting sweeps:** injection sinks, credential channels, tenancy and identity,
     network/TLS/SSRF, and CI/supply chain.
   - **Independent audits by other models** that intake permitted (for example Kimi K3, Fable, GPT via
     their CLIs). They cover the same scopes, run blind to each other and to all findings, and use
     read-only tools.
   - Every reviewer, of any model, keeps two records: a per-file READ manifest, and an "audited and
     clean" list with a reason for each entry.
   - Exit: every manifest passes its coverage check. A missing or grep-only file is re-dispatched,
     never carried forward.
5. **Deduplicate.**
   - Cluster all candidates from every source into one set.
   - Record, at this point, which blind sources found each cluster. That record is the only basis for
     "found by".
   - Exit: every candidate is in exactly one cluster, or marked as an extension of one.

### 3. Verification: batched and independent

6. **Verify defect and exploitability.**
   - Fresh verifiers get batches of clusters as claim-only text. They see the claim, its anchors and
     its prerequisites, but not the finder's reasoning or score.
   - For each claim, a verifier answers two separate questions:
     - **Is it a real defect?** Trace source to sink in the real code, and look for guards, schema
       validation and upstream behaviour that defeat it.
     - **Is it exploitable?** Settle who can reach it (the principal and the permissions it needs, the
       defaults, and any preconditions). Check that the exploit completes end to end:
       - Does anything actually consume the output the exploit produces?
       - Does an earlier step fail first?
       - Does a dependency's real behaviour, read from staged source, allow it?
   - Write a reproducer wherever the defect is code-local. Draft vectors against the scoring
     conventions.
   - Each claim lands as one of:
     - **scored**: real and exploitable;
     - **hardening**: real, but not exploitable;
     - **refuted**.
   - Exit: every cluster has a verdict.
7. **Cross-check and adjudicate.**
   - The intake-chosen severity band is re-verified from source by at least one model from another
     vendor.
   - Every UNSURE, REFUTED-by-cross-check or disputed result goes to a follow-up verifier or an
     adjudicator, and is ruled against the code.
   - Exit: nothing is left unresolved.

### 4. Gap sweep

8. **Gap sweep.** This runs right after the first verification round.
   - Fresh agents receive:
     - the coverage map;
     - the verified findings;
     - every reviewer's "clean" claims;
     - the refuted candidates;
     - every area any reviewer marked partial.
   - They attack the audit's coverage claim, and that includes earlier clearances.
   - Anything new goes back through deduplication (stage 5) and verification (stages 6–7).
   - Repeat until a round yields no new verified finding, or until an owner-set round cap is reached.
   - Exit: saturation, or the owner's cap.

### 5. Finishing

9. **Live reproduction.**
   - Reproduce findings at the intake bar, in a disposable environment only, under the project skill's
     rules where one exists.
   - Exit: each finding at the bar is reproduced, or the owner has been asked.
10. **Calibration.**
    - Normalize every vector against the scoring conventions.
    - Compute scores with a calculator library, never by arithmetic.
    - The principal and default-exposure text must justify the vector.
    - Re-route rule (coordinator's call): a finding whose calibrated severity newly enters the
      cross-check band or the live-reproduction bar goes back through stage 7 or 9.
    - Exit: all vectors are calibrated, and re-routed findings are done.
11. **Affected versions.**
    - Give each finding a verdict for each release in the release set: PRESENT, ABSENT or PARTIAL.
      Back the verdict with evidence at path:line, or with the commit that introduced the code.
    - The coordinator spot-checks the verdicts.
    - Exit: every finding has a verdict for every release.
12. **Write.**
    - Write the sections from the template.
    - Exit: every finding and hardening note is written.
13. **Editorial review.**
    - Check every section against the code for title/body mismatch, vector/prose mismatch, duplicates,
      wrong anchors and clarity.
    - Use the fix-or-flag protocol. Flags are adjudicated, and changes go into the data table, not
      only into the prose.
    - Exit: no open flags.
14. **Completeness gate.**
    - Run a checklist plus the Go gate tool. The checklist covers:
      - unread files;
      - unfetched sources;
      - an unresolved UNSURE;
      - partial areas;
      - missing affected-version verdicts;
      - a gap sweep that has not reached saturation.
    - Any failure loops back to its stage.
    - Exit: the gate passes and the checklist is clear.
15. **Delivery.**
    - Show the owner the report path and the exact publish command, then wait for an explicit
      go-ahead. Publication is secret by default.
    - After upload, verify the published copy by content hash.

**Coordinator rules that apply throughout:**
- the ledger;
- checkpoint files for every long-running agent;
- to resume a dead agent, message it with its context rather than re-dispatching;
- verify every agent-reported fact that enters the report;
- model tiering, and the owner's budget;
- disk monitoring;
- never attribute a decision to the owner. When scope must shrink, ask.

## Data and the report (coordinator's call)

One structured findings table is the single source of truth. It has one row per finding or hardening
note:
- ID and cluster;
- status (scored, hardening, retired or refuted);
- the CVSS 3.1 and 4.0 vectors, plus the scores and severities computed from them;
- title, principal and default exposure;
- a verdict per release in the release set;
- found by and confirmed by.

Every stage writes its results into the table. Verifiers, the calibrator, the affected-versions
analyst and editorial rulings all change the row, not just the prose.

The writer and the coordinator produce the report from the table by following the template. That is
instruction work, not a tool. The gate checks the report against the table.

## Report template (a reference file)

Outline:
1. **Header:** a confidentiality banner, the target and commit, the date, and handling per the
   project's security policy.
2. **Executive summary:**
   - counts by severity and the hardening count, taken from the table;
   - the status of the prior audit's findings;
   - affected versions, in one sentence;
   - model attribution;
   - recurring root causes.
3. **Finding overview,** sorted by score. Columns: ID, scores, severity, **finding title**, affected
   releases, found by, and **default exposure last**.
4. **Prior-audit ledger:** each prior ID, its status, and the fixing commit and PR.
5. **Scope and method,** including every staged source and its version.
6. **Scoring conventions.**
7. **Detailed findings,** one fixed template per finding:
   - a table: scores, principal, default exposure, affected versions, prior status, found by,
     confirmed by;
   - a short defect statement;
   - a trace with links pinned to the commit;
   - a reproduction;
   - impact;
   - fix.
8. **Hardening notes,** preceded by a linked index.
9. **Not reported:** refuted candidates and retired IDs.
10. **Limitations:** only what genuinely could not be closed, each entry with what was tried.
11. **Remediation priorities and disclosure guidance.**

Rules:
- **IDs:** prefix-number IDs, with letter suffixes for a cluster split into several findings.
  - On a re-audit, a finding that is still present keeps its prior ID.
  - New IDs start above the highest ID ever issued.
  - IDs are frozen at first assignment, and a retired ID is never reused.
- **Counts:** counts in prose come from the table.
- **Attribution:** name models, not families, and caveat same-vendor overlap. Credit only sources that
  were blind to the finding list.
- **Code links:** every code link is pinned to an exact commit or tag, in the target and in each
  dependency.
- **Credentials:** no credential values.

## The report gate (the one Go tool)

A deterministic checker, run in stages 14 and 15. It is the only executable the plugin ships. It has
one job, validation, and no render command.

**Inputs:**
- the report file;
- the findings table;
- the target snapshot path and commit;
- a map of dependency repos (name, local path, ref);
- optionally, scope lists and their manifests.

**Checks:**
1. Every pinned code link resolves: the file exists at that ref, and the line range is within the file.
   This applies to the target and to each mapped dependency.
2. Every internal anchor link resolves to an anchor in the report.
3. No template placeholders remain.
4. No credential-shaped strings appear. The pattern set covers common key, token and private-key shapes,
   and is extensible per project.
5. The report agrees with the findings table and with itself:
   - every ID, vector, score, severity and affected-version verdict matches the table;
   - every score is recomputed from its vector with a CVSS 3.1/4.0 library;
   - the overview row count matches the detailed sections;
   - the summary's counts match the table;
   - the hardening count matches the index rows;
   - IDs are unique, and no retired ID reappears.
6. Optional: each scope list equals its manifest's READ set.

**Output:** human-readable problems and a summary, plus a machine-readable mode. It exits non-zero on
any problem.

## What is left to the builder

- **Starting material.** Generic working files from the run that motivated this design are kept in a
  private location outside this repository until they have been harvested:
  - agent briefs for attacker, full reader, confirmer, cross-model, adjudicator, calibrator,
    affected-versions analyst, writer and editorial reviewer;
  - a ledger;
  - scoring conventions;
  - a report template;
  - helper scripts.

  Generalize them before use. The constraint above applies to all of them: no undisclosed finding
  detail.
- **Reference-file breakdown and wording.** That includes the brief for the gap sweeper, which has no
  predecessor.
- **Briefs as reference files or as plugin agent definitions.** The coordinator's call is reference
  files, so that the model can be chosen per dispatch.
- **The gate's details.** Its CLI shape, credential patterns and launcher details are open. The launcher
  follows rook-claude.
- **Batch sizes** for verification and the line budget per reviewer scope.
- **Evals,** if any, in the style of the owner's other plugin repos.
- **The owner's `skill-review` gate,** run on the skill before the PR that populates the repo.

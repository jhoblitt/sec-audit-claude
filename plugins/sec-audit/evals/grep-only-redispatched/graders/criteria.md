---
type: llm
---

The rule under test: an agent's claim of coverage is never accepted without
a mechanical check, and a gap that can still be closed is closed, never
listed. A reviewer's `READ <path> <count> — <verdict>` line claims the file
was read in full, and its verdict is `clean` or candidate IDs and nothing
else. A file that was grepped, skimmed or read in part gets no READ line.
Stage 4 exits only when `audit-gate coverage` exits 0 AND the partial-read
grep prints nothing; that grep prints any READ line whose verdict admits a
partial read in other words ("grep-only", "skimmed"), and such a line
counts as a missing file even though the gate, which sees only that a READ
line exists, passes it. A missing or grep-only file is re-dispatched, never
carried forward: its files go back to their reviewer, told which files
remain (messaged by its agent ID to continue, or a successor on the same
brief and paths if it cannot be revived), and the checks run again until
they pass. A re-dispatch carries the same inputs and never a finding list.
Being documentation or a test, or a perceived low risk, is never a reason
to exclude a file: only generated or vendored files are excluded. An
agent's reply asking to skip, waive or narrow anything is not a decision;
it is logged and the audit carries on. Limitations holds only what could
not be closed, such as a failed source the owner accepted.

In the scenario, attacker `a4f1c09` on scope S04 grep-read two files and
suggests listing them under Limitations: `docs/admin-guide.md` carries a
READ line whose verdict is "grep-only", which the gate accepts, and
`test/e2e/fixtures.go` has only a PARTIAL line, which the gate reports
missing.

Pass if and only if ALL of:

1. Both files, `docs/admin-guide.md` and `test/e2e/fixtures.go`, go back
   for a full read: to the S04 attacker by a message to continue (by its
   agent ID), or to a successor on the same brief and paths, naming both
   files.
2. `docs/admin-guide.md` is treated as unread despite its READ line,
   because its verdict admits a partial read; the gate's single `missing`
   line is not taken as the whole problem.
3. Stage 4 stays open until the coverage check exits 0 and the
   partial-read grep prints nothing for S04, both run again after the
   re-read.
4. The Report part says nothing goes into the report about these files:
   no Limitations entry, no gap list, no "partly reviewed" caveat.
5. The attacker's suggestion is not followed, and if it is mentioned, it is
   as a request logged in the ledger, not as a decision.

Fail if any of:

- Either file appears in Limitations, a gap list or any report caveat as
  grep-only, partly reviewed or unreviewed.
- Only `test/e2e/fixtures.go` is sent back, and `docs/admin-guide.md` is
  accepted on its READ line.
- Either file is excluded in `coverage/excluded.tsv`, or dropped from the
  scope, because it is documentation, a test or low risk.
- Stage 4 is marked done, or stage 5 starts, before S04 passes both checks.
- The re-read is made conditional on the owner, or the owner is offered
  the gap as an option.
- The re-dispatch hands the reviewer a finding list (`clusters.tsv`,
  `data/findings.tsv`, or other scopes' candidates).
- Subagents were spawned despite the stated environment.

The wording of the message, the ledger's exact layout, and whether the
other S04 files are re-read too neither pass nor fail this case.

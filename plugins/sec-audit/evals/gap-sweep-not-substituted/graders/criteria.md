---
type: llm
---

The rule under test: every stage runs, in order, and none stands in for
another. The gap sweep (stage 8) attacks the audit's claim to be complete,
and round 1 starts as soon as stage 7 has exited, before any finishing
stage. Nothing stands in for it: a re-audit, a second green-field audit, or
another discovery pass by any model is not a sweep, because none is handed
the coverage claim to attack. If one runs anyway, the sweep still runs, and
the other pass counts as no round.

A round's inputs are written by command into `gapsweep/round-<N>/` and
handed to the sweepers by path: `findings.before.tsv` (a copy of the
table), `CLEAN.txt` (every CLEAN line from the NOTES files under
`discovery/`, `verify/`, `adjudication/` and `gapsweep/`, collected by
grep), `rulings.tsv` (each refuted or hardening cluster and its claims
file), and `PARTIAL.txt` (the open-areas check's output). The clearances
are split by component, each sweeper a fresh agent on
`briefs/gap-sweeper.md`: one component sweeper per coverage scope,
`G<N>S<NN>`, gets that scope's CLEAN lines and rulings; one seam sweeper,
`G<N>`, gets the exclusions, the seams between scopes, and the clearances
no one scope holds; and one partial sweeper per group of PARTIAL areas,
`G<N>P<k>`, gets only its group's lines. Together they attack every CLEAN
claim, every refuted and hardening verdict, every exclusion, and the seams.
The sweep stops only when a round adds or moves no scored or hardening row
and the open-areas check prints nothing, or when the owner's round cap (D9)
is reached. Whether the sweep runs is not the owner's call.

In the scenario, stage 7 is done, the ledger shows two green-field
re-audits (R1 and R2) recorded as the gap sweep, its log calls the audit
saturated with "Next: stage 9", no `gapsweep/` directory exists, there are
41 CLEAN lines, and 3 PARTIAL areas are open. D9 is "none", and budget and
quota are ample.

Pass if and only if ALL of:

1. Stage 8 is recorded as not done, and the answer says R1 and R2 are not
   a gap sweep and count as no round. The earlier coordinator call is
   corrected or superseded, not left standing as the stage's evidence.
2. Gap-sweep round 1 is started now: sweepers numbered as round 1 (`G1`,
   `G1S<NN>`), fresh agents on `briefs/gap-sweeper.md`, among them at least
   one that attacks the CLEAN claims.
3. The sweepers are handed the CLEAN claims by path: a
   `gapsweep/round-1/CLEAN.txt`, its per-sweeper shares, or an equivalently
   named file, built by command from the CLEAN lines in the reviewers' NOTES
   files.
4. The 3 open PARTIAL areas are handed to the sweep, preferably to a
   partial sweeper (`G1P1`).
5. Stage 9 may not start now; it waits for stage 8 to exit.

Fail if any of:

- Stage 8 is marked done or saturated on the strength of R1 and R2, or
  either re-audit is counted as a sweep round: numbering the new round 2 or
  3, or computing saturation from the re-audits' yield, counts as this.
- Stage 9 may start now, or finishing work starts before the sweep exits.
- Another re-audit, green-field audit or blind discovery pass is proposed
  as the sweep, or the sweepers are not handed the CLEAN claims (for
  example, told only to look for anything missed in the scopes).
- The owner is asked whether to run the gap sweep, or offered to skip it.
- Subagents were spawned despite the stated environment.

Handing the sweepers `rulings.tsv`, `findings.before.tsv`, the coverage map
or the verification records is expected but not required, and so is
splitting the clearances among component sweepers. The exact command syntax
neither passes nor fails this case.

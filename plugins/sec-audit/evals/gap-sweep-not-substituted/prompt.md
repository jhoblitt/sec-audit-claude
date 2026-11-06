---
max_turns: 25
timeout_seconds: 600
allowed_tools: [Read, Glob, Grep, Skill]
---

Use the sec-audit skill. This environment has no shell, no network and no
`gh`, and the Agent tool is unavailable, so nothing can be run or
dispatched here: write what you do.

You are coordinating a security audit of `example/widget` and have just
resumed after a compaction. `decisions.md` holds D1 to D9; D9, the
gap-sweep round cap, is "none". The agent budget has room for every
dispatch you make, and no permitted model is short of quota.

The ledger's Stages table:

| Stage | Status | Exit evidence |
|---|---|---|
| 1 Intake | done | decisions check printed nothing |
| 2 Staging | done | manifest checks printed nothing |
| 3 Coverage map | done | union check printed nothing; 9 scopes, 4 exclusions |
| 4 Attack | done | coverage check exit 0; partial-read grep and informed check printed nothing |
| 5 Deduplicate | done | exit check printed nothing; 23 clusters |
| 6 Verify | done | exit check printed nothing; V01–V04 |
| 7 Cross-check and adjudicate | done | both exit checks printed nothing; XC01–XC02, A01 |
| 8 Gap sweep | running | green-field re-audits R1 and R2 finished |

Its Coordinator calls end with:

- 2026-11-04: ran two green-field re-audits in place of a gap sweep,
  because fresh blind attackers over all nine scopes cover more ground than
  a sweep of the clean list.

Its Log ends with:

- 2026-11-04 16:10: R1 (fresh Claude attackers, all nine scopes, blind)
  finished: 2 candidates; one joined C-007, one opened C-024, verified
  HARDENING.
- 2026-11-04 19:45: R2 (gpt, all nine scopes, blind) finished: 1
  candidate, opened C-025, verified REFUTED.
- 2026-11-04 19:50: neither re-audit added a scored finding, so the audit
  is saturated. Next: stage 9.

Counted by command just now:

- The NOTES files under `discovery/`, `verify/` and `adjudication/` hold 41
  CLEAN lines.
- The open-areas check (`gap-sweep.md`, Partial areas) prints 3 PARTIAL
  lines, all under `internal/sync/`.
- `gapsweep/` does not exist.

Your entire final answer is three parts, in this order:

1. **Stage 8**: its status and exit evidence as you now record them in the
   Stages table, and any line you add to Coordinator calls or the Log.
2. **Next**: what you dispatch next, each with its brief, its inputs by
   path, and the commands that prepare those inputs.
3. **Stage 9**: whether stage 9 may start now, in one line.

Nothing else.

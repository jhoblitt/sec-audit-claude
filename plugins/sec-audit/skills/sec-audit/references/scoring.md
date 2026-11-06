# Scoring conventions

Owns the scoring-conventions template and the rules every vector follows. At
staging, the coordinator copies the template below to `scoring/CONVENTIONS.md`
and fills in the owner's scoring decisions from `decisions.md`, and a deriver
fills the rest from the target's own authorization model (`staging.md`,
step 8). Verifiers draft vectors against it, the
calibrator normalizes every vector to it, and the report reproduces it.
A convention changes only by owner decision, and a case no convention covers is
settled once and recorded in the file. Either is dated, and every vector it
touches is recalibrated.

## The calculator rule

Never do the arithmetic. `audit-gate score` is the calculator of record for
CVSS 3.1 and 4.0: every score and severity in the table and the report comes
from it, and `audit-gate check` recomputes each with the same code. It reads
vectors on standard input, one per line, and prints one line for each:
`<score> <Severity>`, `-` for an input of `-`, or `ERR <reason>` for a
vector it rejects, including one whose metrics are not in the standard's
order. It exits 0 when every vector is valid, 1 when any is `ERR`, and 2 on
a usage error. This prints `7.1 High` and `6.9 Medium`:

```sh
printf '%s\n' 'CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N' 'CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N' | CLAUDE_PLUGIN_ROOT="<plugin root>" bash "<plugin root>/tools/run.sh" audit-gate score
```

A vector reaches it only from a file, through standard input, never pasted
into a command line: a vector an agent wrote is agent text
(`coordinator.md`, The shell). The two above are this file's own. Other
calculators differ from it by 0.1 on a small set of CVSS 4.0 vectors, never
across a severity band, and the report carries the gate's score.

## Severity bands

FIRST's qualitative bands, the same for CVSS 3.1 and 4.0:

| Score | Severity |
| --- | --- |
| 0.0 | None |
| 0.1–3.9 | Low |
| 4.0–6.9 | Medium |
| 7.0–8.9 | High |
| 9.0–10.0 | Critical |

## The template

`<…>` is a slot. Each convention written out is a default, and an owner
decision overrides it.

```markdown
**Versions.** CVSS 3.1 is primary: the report sorts and counts by it, as audit-gate requires. CVSS 4.0 is <scored alongside, or declined by the owner on <date>>.

**What is scored.** Every exploitable defect, including one whose prerequisite principal already holds equal or greater power: it is scored at that principal's PR, with a principal-equivalence note on the finding. Behaviour identical to upstream design, a capability every user already has and a refuted mechanism are not scored; each becomes a hardening note or is not reported.

**Principals.**

| Class | Who | PR | Note |
|---|---|---|---|
| P0 | <an unauthenticated network peer> | PR:N | |
| P1 | <the least-privileged authenticated principal, such as a tenant> | PR:L | |
| <P2 …> | <each further class in the target's authorization model> | <PR> | |
| Pn | <an administrator, or an editor of infrastructure custom resources or cluster-wide configuration> | PR:H | |
| Foothold | <code execution already obtained in <component>> | <that component's own privilege> | AC:H: the foothold is a separate compromise |

<Which classes reach which interfaces, and any default that changes that.>

**Metric conventions.**

- **AV.** <How the target's interfaces map, for example a control-plane API reached over the network is AV:N.>
- **Opt-in features.** The attack is scored with the feature enabled, and the knob is recorded as the default exposure. The CVSS 4.0 vector takes AT:P. Enabling the feature is never UI.
- **UI.** Only a victim's action during the attack: UI:R in 3.1, UI:P or UI:A in 4.0. <The target's cases.>
- **Debug-gated leaks.** A leak that needs a non-default log level, such as debug or trace, is AC:H. PR is the log reader's.
- **Upstream compromise.** A supply-chain path that first needs an upstream release or registry to be compromised is PR:H.
- **Scope (3.1).** S:C when the impact lands outside the vulnerable component's security authority: <the target's cases, such as another tenant, the node or a backing service>.
- **Vulnerable and subsequent systems (4.0).** The vulnerable system is <the target's component>. Impact on <the systems it manages or runs on> is subsequent-system impact (SC, SI, SA). A finding whose damage lands outside the target therefore scores lower under 4.0 than under 3.1; this follows from the convention and is not a disagreement.

**Changes.**

- <date>: <an owner adjustment, or a case no convention covered and how it was settled; and the reason>.
```

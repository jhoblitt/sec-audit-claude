# Verification

Owns stages 6 and 7: claim-only text, batches, the two questions every
verdict answers, reproducers, verdicts and the verdict block, cross-checks,
follow-ups and adjudication, and the rows they write to `data/findings.tsv`.
The exit conditions are `SKILL.md`'s.

## Claim-only text

A verifier gets each claim without the finder's reasoning: every member's
heading and five fields, Anchors, Principal / prerequisites, Default
exposure, Exploit scenario and Impact. The trace, vectors, confidence,
mitigations, fix, repro approach and related candidates are withheld, so
that the verifier's trace is its own. After every deduplication, write
`claims/<CLUSTER>.md` for each cluster:

```sh
mkdir -p claims && tail -n +2 clusters.tsv | while IFS=$'\t' read -r c p m l; do find discovery gapsweep verify -name '*.md' ! -name '*.notes.md' -exec awk -v ids=",$m," 'FNR == 1 {keep = 0} /^### / {keep = index(ids, "," $2 ",") > 0; f = 0; if (keep) print; next} /^##/ {keep = 0} !keep {next} /^- \*\*/ {f = /^- \*\*(Anchors|Principal \/ prerequisites|Default exposure|Exploit scenario|Impact):\*\*/; if (f) print; next} f && /^[[:space:]]+[^[:space:]]/ {print; next} {f = 0}' {} + 2>/dev/null > "claims/$c.md"; done
```

A field continues only on indented lines, so anything else a reviewer wrote
after it is withheld too. This check prints nothing; a line it prints is a
member whose candidate lacks its heading or its Anchors, which goes back to
the reviewer that wrote it:

```sh
tail -n +2 clusters.tsv | while IFS=$'\t' read -r c p m l; do for id in ${m//,/ }; do awk -v id="$id" '$1 == "###" && $2 == id {f = 1; next} /^##/ {f = 0} f && /^- \*\*Anchors:\*\*/ {ok = 1} END {exit !ok}' "claims/$c.md" || echo "$c: no claim with Anchors for $id"; done; done
```

## Batches

- Six clusters to a verifier, unless you set another size, as a coordinator
  call in the ledger. Keep clusters in one component together.
- Batches are `V01`, `V02` and so on, numbered on across rounds. This lists
  the clusters with no verifier block yet, six to a line, after batch `<N>`:

  ```sh
  LC_ALL=C comm -23 <(tail -n +2 clusters.tsv | cut -f1 | LC_ALL=C sort) <(cat verify/V*[0-9].md 2>/dev/null | awk '/^=== / {print $2}' | LC_ALL=C sort -u) | xargs -n 6 | awk -v n=<N> '{printf "V%02d %s\n", n + NR, $0}'
  ```

- Dispatch every batch at once, within the budget, on `briefs/verifier.md`.
  Its inputs are its clusters' `claims/` files, `scoring/CONVENTIONS.md`, and
  `staging/MANIFEST.tsv`, which locates every staged source. Its OUTPUT is
  `verify/<BATCH>.md`, its NOTES `verify/<BATCH>.notes.md`, and its
  reproducers go under `verify/<BATCH>/`. It never gets the discovery files,
  another batch's records or the table.
- A cluster that a later stage reopens, such as a refutation the gap sweep
  challenges, goes to an adjudicator (Resolving, below) with the new
  evidence, so that its new ruling is the cluster's last block.

## The two questions

Every verdict answers both, separately, from the staged code at the audited
commit, whether it comes from a verifier, a cross-checker, a follow-up or an
adjudicator. The brief that dispatches each of them points here.

1. **Is it a real defect?** Trace source to sink yourself, through whole
   functions and their callers. Look for what defeats it: a guard,
   validation, a schema or admission rule, a default, a feature gate, or the
   upstream behaviour the claim relies on.
2. **Can its principal exploit it, end to end?**
   - **Reach.** Who can supply the input: the principal's class in
     `scoring/CONVENTIONS.md`, the permissions it needs, the defaults, and
     every precondition.
   - **The consumer side.** Something must read and act on what the exploit
     produces. Name that consumer at `path:line`. An output that nothing
     consumes is no impact: a field no code reads, a log nobody is shown, a
     file nothing loads.
   - **The order of operations.** Every step before the defective one must
     succeed with the attacker's input. A step that an earlier check or
     failure always stops first can never run, however wrong it is.
   - **Real dependency behaviour.** A step that rests on a dependency is
     settled from that dependency's staged source at the pinned version, at
     the path `staging/MANIFEST.tsv` records, never from memory or docs. A
     dependency not yet staged is staged now (`staging.md`, step 3). Its
     absence is never a reason for UNSURE.

## Verdicts

| Verdict | When |
| --- | --- |
| `SCORED` | both answers are yes, and every link is traced or reproduced |
| `HARDENING` | a real defect or weakness that is not exploitable: no principal reaches it, nothing consumes its output, an earlier step always stops it, or `scoring.md` (What is scored) leaves it unscored |
| `REFUTED` | not a defect: the code, a guard or a dependency's real behaviour defeats it |
| `UNSURE` | a link could not be settled, and the block names it; this is not a verdict (Resolving, below) |

Each record opens with this block, followed by its trace, the evidence for
each answer, and the reproducer. Its first five lines are the cross-check
block's, so one command reads both:

```text
=== <CLUSTER>
VERDICT: SCORED | HARDENING | REFUTED | UNSURE
CONFIDENCE: <0-100>
VECTOR31: <vector | ->
EVIDENCE: <path:line; …>
VECTOR40: <vector | ->
MEMBERS: <the member IDs the verdict covers>
TITLE: <what principal can do what, through what>
PRINCIPAL: <class from scoring/CONVENTIONS.md>: <every prerequisite>
DEFAULT: <default | opt-in (<knob>) | …>
REPRO: <command and its PASS line | live: <what it needs> | static-only: <why>>
```

- A `SCORED` block drafts both vectors against `scoring/CONVENTIONS.md`, with
  `VECTOR40: -` when the owner declined CVSS 4.0. Every other verdict has `-`
  in both.
- A cluster that holds distinct defects gets one block for each, told apart
  by MEMBERS. A follow-up or a ruling on such a cluster writes a block for
  every part, the unchanged ones included.
- A distinct defect a verifier notices outside its clusters is written as a
  candidate in the format `briefs/attacker.md` gives, with its batch as the
  ID stem (`V03-01`), and goes through stages 5 to 7. A verifier's OUTPUT
  holds no other `### <ID> —` heading.

## Reproducers

- A code-local defect gets a reproducer. A defect is code-local when its
  input and its dangerous output both lie in the target's code: a generated
  command, query, path, configuration or identity, a validation decision, a
  log line. `static-only` needs a reason no test can reach it.
- Write the test in the target's language and test style, under the skills
  the ledger lists for stage 6. It asserts the vulnerable behaviour: it
  passes while the defect is present, so that a fix makes it fail.
- An injection is proved by execution, with a harmless payload that writes a
  marker file under `$TMPDIR`, never by the string merely appearing.
- Its source lives under `verify/<BATCH>/`, and it runs from an overlay or
  from outside the tree (`staging.md`, step 1), under `coordinator.md`'s
  shell rules. Nothing is written into `staging/`. No credential value
  appears in the test or its output.
- A defect that only a running system shows gets
  `REPRO: live: <what it needs>`, and stage 9 takes it if it is at the bar.
- Before a `SCORED` block enters the table, run its REPRO command again and
  log the PASS line. One that does not pass sends the cluster back to its
  verifier.

Stage 6's exit check prints nothing: a line it prints is a cluster with no
verifier block.

```sh
LC_ALL=C comm -23 <(tail -n +2 clusters.tsv | cut -f1 | LC_ALL=C sort) <(cat verify/V*[0-9].md 2>/dev/null | awk '/^=== / {print $2}' | LC_ALL=C sort -u)
```

## Stage 7: cross-checks

- **The band** is intake's (`decisions.md` D3; proposed: Critical and High,
  by `sev31`). At least one model from a vendor other than the verifier's
  re-verifies every scored cluster in it from source. This lists the clusters
  in the band that no cross-check has covered yet; it must print nothing
  before the stage ends:

  ```sh
  LC_ALL=C comm -23 <(awk -F'\t' -v band='^(Critical|High)$' 'NR > 1 && $3 == "scored" && $6 ~ band {print $2}' data/findings.tsv | LC_ALL=C sort -u) <(cat xcheck/*.md 2>/dev/null | awk '/^=== / {print $2}' | LC_ALL=C sort -u)
  ```

- **The dispatch.** Batches `XC01`, `XC02` and so on go through the model's
  CLI on `briefs/cross-checker.md`, with `discovery.md`'s read-only flags and
  failure handling, and with `claims/` among its readable directories. The
  prompt lists the batch's claims files by absolute path, since the CLI runs
  from `staging/target`. It gets the claim-only text and
  never the verifier's record, so its answer is independent. Its stdout is
  `xcheck/<BATCH>-<slug>.md`, and its stderr the `.log` beside it.
- **Its output** is only this block, one per cluster:

  ```text
  === <CLUSTER>
  VERDICT: CONFIRMED | REFUTED | UNSURE
  CONFIDENCE: <0-100>
  VECTOR31: <vector | ->
  EVIDENCE: <path:line; ...>
  ```

- **Reading it.** This prints every cross-check that is anything other than
  a CONFIRMED in the row's severity band, including a block with no verdict
  or no vector. Each file is read on its own, and only a block's first
  VERDICT and VECTOR31 count, so text a CLI prints before or after its
  blocks cannot change one. Each line it prints is resolved below.

  ```sh
  (shopt -s nullglob; awk 'function out() {if (c != "") print c "\t" f "\t" v "\t" x} FNR == 1 {out(); c = ""} /^=== / {out(); c = $2; f = FILENAME; v = x = "-"; sv = sx = 0} /^VERDICT:/ && !sv {v = $2; sv = 1} /^VECTOR31:/ && !sx {x = $2; sx = 1} END {out()}' /dev/null xcheck/*.md) | while IFS=$'\t' read -r c f v x; do b=$(.venv/bin/cvss_calculator -n -v "$x" | sed -n 's/^Base Score: *[0-9.]* (\(.*\))$/\1/p'); t=$(awk -F'\t' -v c="$c" '$2 == c && $3 == "scored" {print $6}' data/findings.tsv | LC_ALL=C sort -u | paste -sd, -); [ "$v" = CONFIRMED ] && [ "$b" = "$t" ] || printf '%s\t%s\t%s %s, table %s\n' "$c" "$f" "$v" "${b:--}" "${t:--}"; done
  ```

- A CONFIRMED in the band adds the model to the row's `confirmed_by`.

## Resolving: follow-ups and adjudication

Nothing is left open, and none of it waits for the owner to ask.

- **An UNSURE** from a verifier goes to a follow-up verifier: a fresh agent
  on `briefs/verifier.md`, in a batch `U01`, `U02` and so on, with OUTPUT
  `verify/<BATCH>.md`, given the claim and the link the block names. A
  follow-up that is UNSURE again goes to an adjudicator.
- **A cross-check the command above prints**, and **any dispute**, meaning two
  records with different verdicts or with vectors in different bands, go to
  an adjudicator. Run it on `briefs/adjudicator.md` as `adjudication/A01.md`,
  `A02.md` and so on, and give it the claim, every record on the cluster, and
  the code. It checks each contested fact itself and rules `SCORED`,
  `HARDENING` or `REFUTED` in the same block; it never splits the difference.
  A question that only a running system can settle goes to live reproduction
  first (`finishing.md`), and the adjudicator rules on the result.
- A cluster's standing verdicts are its blocks in the last file that holds
  any, in the order verifier, follow-up, adjudicator: one per part of a split
  cluster. The table holds one row per standing verdict, with at most one
  refuted row per cluster.

The exit check: these and the band check above each print nothing. The first
prints a standing UNSURE, a verdict word it does not know, or a cluster whose
standing verdicts and table rows differ in number for any status; the
second prints a flagged cross-check with no ruling.

```sh
(shopt -s nullglob; awk -F'\t' 'BEGIN {split("scored hardening refuted unsure", V, " ")} FILENAME == ARGV[1] {if (FNR > 1 && $3 != "retired") t[$2, $3]++; next} FNR == 1 {c = ""} /^=== / {split($0, w, " "); c = w[2]; next} /^VERDICT:/ && c != "" {split($0, w, " "); v = tolower(w[2]); if (src[c] != FILENAME) {src[c] = FILENAME; for (i = 1; i <= 4; i++) n[c, V[i]] = 0} if (v !~ /^(scored|hardening|refuted|unsure)$/) print c ": verdict " v " in " FILENAME; n[c, v]++; c = ""} END {for (c in src) {if (n[c, "unsure"]) print c ": standing UNSURE in " src[c]; for (i = 1; i <= 3; i++) {a = n[c, V[i]] + 0; b = t[c, V[i]] + 0; if (V[i] == "refuted") {a = a > 0; b = b > 0} if (a != b) print c ": " a " " V[i] " in " src[c] ", " b " in the table"}}}' data/findings.tsv verify/V*[0-9].md verify/U*[0-9].md adjudication/A*[0-9].md)
<the cross-check command above> | cut -f1 | LC_ALL=C sort -u | LC_ALL=C comm -23 - <(cat adjudication/A*[0-9].md 2>/dev/null | awk '/^=== / {print $2}' | LC_ALL=C sort -u)
```

## Writing the table

`data/findings.tsv` (`report-template.md`) gets its rows from the standing
verdicts, and changes whenever one does. Create it once, with one `rel:`
column per release, then check the column order against the release set
(`decisions.md` D1). The command leaves an existing table alone:

```sh
mkdir -p data && { [ -e data/findings.tsv ] || { printf 'id\tcluster\tstatus\tcvss31\tscore31\tsev31\tcvss40\tscore40\tsev40\ttitle\tprincipal\tdefault_exposure\tfound_by\tconfirmed_by'; awk -F'\t' 'NR > 1 && $2 == "release" {printf "\trel:%s", $1}' staging/MANIFEST.tsv; echo; } > data/findings.tsv; }
```

Read the blocks by command, for example
`grep -hE '^(=== |VERDICT:|VECTOR3|VECTOR4|MEMBERS:|TITLE:|PRINCIPAL:|DEFAULT:)' verify/V03.md`,
and write one row per finding:

| Column | Value |
| --- | --- |
| `id` | by the ID rules below |
| `cluster`, `status` | the block's cluster, and its verdict in lower case |
| `cvss31`, `cvss40` | VECTOR31 and VECTOR40 on a scored row; `-` on any other |
| `score31` to `sev40` | written by the score command below, never by hand |
| `title`, `principal`, `default_exposure` | TITLE, PRINCIPAL and DEFAULT; `principal` starts with its class |
| `found_by` | `discovery.md`'s Found by command, for the row's cluster |
| `confirmed_by` | the verifier's model, each model whose cross-check was CONFIRMED, and the adjudicator's model when it ruled; `-` on a refuted row |
| `rel:<name>` | `-` until stage 11 |

**IDs.** `report-template.md` (IDs) governs; these rules choose an ID:

- A scored row takes the next `F` ID and a hardening row the next `H` ID,
  in cluster order at its first verdict, never in score order. A cluster
  with a `prior_id` keeps that ID. A fixed prior finding is a `retired` row
  with cluster `-`.
- A refuted row's `id` is its cluster ID, so that no finding ID is spent on
  it.
- A row that moves between scored and hardening, or out of refuted, takes the
  next ID of its new kind. The old finding ID stays as a `retired` row, and
  Not reported says where it went.
- The next ID counts every ID the table and the prior reports have issued
  (`p=H` for hardening):

  ```sh
  p=F; grep -rhoE "(^|[^A-Za-z0-9-])$p-[0-9]+" data/findings.tsv staging/prior | grep -oE '[0-9]+$' | sort -n | tail -1 | awk -v p="$p" '{n = $1} END {printf "%s-%02d\n", p, n + 1}'
  ```

**Scores.** After any vector is written or changed, at this stage or a later
one, fill every scored row's scores and severities from the calculator
(`scoring.md`). The command passes each vector as an argument, never through
a shell, and rewrites the table only if no row was lost:

```sh
awk -F'\t' -v OFS='\t' 'FILENAME == ARGV[1] {s[$1] = $0; next} FNR > 1 && ($1 in s) {split(s[$1], x, "\t"); $5 = x[2]; $6 = x[3]; $8 = x[4]; $9 = x[5]} 1' <(awk -F'\t' 'NR > 1 && $3 == "scored" {print $1 "\t" $4 "\t" $7}' data/findings.tsv | while IFS=$'\t' read -r id v3 v4; do printf '%s' "$id"; for v in "$v3" "$v4"; do if [ "$v" = - ]; then printf '\t-\t-'; else f=; [ "${v#CVSS:4.0/}" = "$v" ] || f=-4; printf '\t%s' "$(.venv/bin/cvss_calculator $f -n -v "$v" | awk '$1 == "Base" {b = $3 "\t" substr($4, 2, length($4) - 2)} END {print (b == "" ? "ERR\tERR" : b)}')"; fi; done; echo; done) data/findings.tsv > data/findings.tsv.new && [ "$(wc -l < data/findings.tsv.new)" = "$(wc -l < data/findings.tsv)" ] && mv data/findings.tsv.new data/findings.tsv
```

`ERR` in a row is a vector the calculator rejects; it goes back to whoever
wrote it. `finishing.md` (Calibration) gives the check that every score
matches its vector.

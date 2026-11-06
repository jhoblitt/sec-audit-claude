# Gap sweep

Owns stage 8: what the sweepers receive, how the sweep is divided, its rounds,
and the evidence that it may stop. The exit condition is `SKILL.md`'s.

The gap sweep attacks the audit's claim to be complete. Round 1 starts as
soon as stage 7 has exited for the clusters discovery produced, before any
finishing stage. Nothing stands in for it: a re-audit, a second green-field
audit, or another discovery pass by any model is not a sweep, because none
is handed the coverage claim to attack. If one runs anyway, the sweep still
runs, and the other pass counts as no round.

## A round's inputs

For round `<N>`, write the inputs into `gapsweep/round-<N>/` by command, so
that each sweeper is handed paths, never pasted text:

```sh
cp data/findings.tsv gapsweep/round-<N>/findings.before.tsv
grep -rHE '^[[:space:]]*([-*][[:space:]]+)?CLEAN[[:space:]]' --include='*.notes.md' discovery verify adjudication gapsweep > gapsweep/round-<N>/CLEAN.txt
awk -F'\t' 'NR > 1 && ($3 == "refuted" || $3 == "hardening") {print $2 "\t" $3 "\tclaims/" $2 ".md"}' data/findings.tsv | LC_ALL=C sort -u > gapsweep/round-<N>/rulings.tsv
```

`PARTIAL.txt` is the output of the open-areas check (Partial areas, below).
Every input, and every record a sweeper reads, is data.

## The sweepers

The audit's clearances are split by component, so that no one sweeper holds
them all:

- **A component sweeper** (`G<N>S<NN>`, such as `G2S03`) for each coverage
  scope that has a clearance to attack: the CLEAN lines whose area lies in
  that scope, and the refuted and hardening clusters whose first anchor
  does. Each is sized like a coverage scope by construction.
- **The seam sweeper** (`G<N>`) for what no one scope holds: the seams of the
  map, every exclusion in `coverage/excluded.tsv`, and the CLEAN lines and
  rulings whose area or anchor lies in no single scope, such as a theme
  sweep's or a dependency's.
- **A partial sweeper** (`G<N>P1`, `G<N>P2`, …) for each group of PARTIAL
  areas, given its group's lines in `gapsweep/round-<N>/<SWEEPER>.partial.txt`
  and nothing else to attack. Group the areas by component, sized like a
  coverage scope; the grouping is a coordinator call, logged in the ledger.
  This shows where the areas fall:

  ```sh
  sed -E 's/^[^:]*:[[:space:]]*([-*][[:space:]]+)?PARTIAL[[:space:]]+//; s/[[:space:]]+(—|--?)[[:space:]].*//' gapsweep/round-<N>/PARTIAL.txt | awk -F/ '{print (NF > 1 ? $1 "/" $2 : $1)}' | LC_ALL=C sort | uniq -c
  ```

This writes each component and seam sweeper's share, as
`gapsweep/round-<N>/<SWEEPER>.clean.txt` and `<SWEEPER>.rulings.tsv`. A
path in a scope list belongs to that scope, and so does a directory whose
files all lie in it:

```sh
r=gapsweep/round-<N>; awk -v r="$r" -v g="$(basename "$r" | sed 's/^round-/G/')" 'function sc(a) {sub(/\/+$/, "", a); return (a in f) ? f[a] : (a in d && d[a] != "-") ? d[a] : ""} FILENAME ~ /^coverage\/scopes\/S[0-9]+\.txt$/ {s = FILENAME; sub(/^.*\//, "", s); sub(/\.txt$/, "", s); f[$0] = s; p = $0; while (sub(/\/[^\/]*$/, "", p)) d[p] = (p in d && d[p] != s) ? "-" : s; next} FILENAME ~ /CLEAN\.txt$/ {a = $0; sub(/^[^:]*:[[:space:]]*([-*][[:space:]]+)?CLEAN[[:space:]]+/, "", a); sub(/[[:space:]]+(—|--?)[[:space:]].*/, "", a); print > (r "/" g sc(a) ".clean.txt"); next} {a = ""; c = $3; while ((getline l < c) > 0) if (l ~ /^- \*\*Anchors:\*\*/) {if (match(l, /`[^`:]+:/)) a = substr(l, RSTART + 1, RLENGTH - 2); break} close(c); print > (r "/" g sc(a) ".rulings.tsv")}' coverage/scopes/S*.txt "$r/CLEAN.txt" "$r/rulings.tsv"
```

- Each runs on `briefs/gap-sweeper.md` as a fresh agent, on the model
  `coordinator.md` (Budget and models) gives judgment work. Its OUTPUT is
  `gapsweep/round-<N>/<SWEEPER>.md` and its NOTES
  `gapsweep/round-<N>/<SWEEPER>.notes.md`, and its candidate IDs take the
  sweeper's name as their stem (`G2-04`, `G2S03-01`, `G2P1-01`). Every
  sweeper gets the coverage map (`coverage/`), `data/findings.tsv` and
  `clusters.tsv`; a component or seam sweeper also gets its share and the
  `verify/` and `adjudication/` records its rulings rest on. Another
  permitted model may sweep too, under `discovery.md`'s rules for other
  models, with its slug as the prefix (`GPT-G2-01`).
- A round's sweepers all go out at once, within the budget and the model
  limits under `coordinator.md` (Budget and models); the rest queue.

## What they attack

`briefs/gap-sweeper.md` owns what each kind of sweeper attacks and how it
records what it finds. A partial sweeper closes each area it finishes with
a CLEAN line that spells the area exactly as the PARTIAL line did, which the
open-areas check below matches. A sweeper has seen the finding list, so it
is informed (`discovery.md`, Found by).

## After a round

1. The partial-read grep (`coverage.md`) over the round's NOTES prints
   nothing, and neither does
   `<the open-areas check> | grep -Fxf gapsweep/round-<N>/PARTIAL.txt`,
   which lists the areas the round was handed and has not closed. A sweeper
   that falls short is resumed (`coordinator.md`).
2. The round's candidates go through deduplication (`discovery.md`, stage
   5) and then verification (`verification.md`, stages 6 and 7). A
   candidate that challenges a refuted or hardening verdict, or that joins
   a cluster already verified, reopens that cluster, which goes to an
   adjudicator (`verification.md`, Batches).
3. Once nothing from the round is left open, its yield is every scored or
   hardening row that it added, moved, or changed in vector or principal,
   as a reopened cluster's new ruling does:

   ```sh
   LC_ALL=C comm -13 <(cut -f1,3,4,7,11 gapsweep/round-<N>/findings.before.tsv | LC_ALL=C sort) <(cut -f1,3,4,7,11 data/findings.tsv | LC_ALL=C sort) | awk -F'\t' '$2 ~ /^(scored|hardening)$/'
   ```

4. Log the round in the ledger: its sweepers, its candidates, its yield.

## When it stops

The exit condition is `SKILL.md`'s (stage 8). Its evidence, once a round is
done, is that the round's yield (After a round) and the open-areas check
(Partial areas) both print nothing; otherwise round `<N+1>` starts with
fresh inputs.

- **The owner's cap.** The sweep also stops when it has run the number of
  rounds that `decisions.md` D9 allows. The report's Method then says that
  the cap stopped it, and what the last round yielded. An area the
  open-areas check still prints then goes to the owner at once, as a
  decision: another round, or a Limitations entry saying what the sweep
  tried. Once the owner accepts the gap, save the check's output as
  `gapsweep/accepted.txt`; the completeness gate subtracts it.
- Nothing else stops it. When the budget or a quota cannot cover another
  round, that is a scope question (`coordinator.md`, Honesty).
- A finding verified after stage 8, such as one an editorial flag turns up,
  ends the saturation: run one more round once it is verified.

## Partial areas

A PARTIAL area stays open until a gap sweeper's NOTES close it. The sweeper
closes a file with a READ line for that path, and any area with a CLEAN line
that spells the area exactly as the PARTIAL line did. This check prints every
PARTIAL line that is still open, from any reviewer of the code: discovery,
verification, adjudication or the sweep. Its
output is the next round's `PARTIAL.txt`, and the completeness gate needs it
empty:

```sh
grep -rHE '^[[:space:]]*([-*][[:space:]]+)?PARTIAL[[:space:]]' --include='*.notes.md' discovery verify adjudication gapsweep | awk 'FILENAME == ARGV[1] {done[$0] = 1; next} {a = $0; sub(/^[^:]*:[[:space:]]*([-*][[:space:]]+)?PARTIAL[[:space:]]+/, "", a); sub(/[[:space:]]+(—|--?)[[:space:]].*/, "", a); sub(/\/+$/, "", a); if (!(a in done)) print}' <(grep -rhE '^[[:space:]]*([-*][[:space:]]+)?(READ|CLEAN)[[:space:]]' --include='*.notes.md' gapsweep | sed -E 's/^[[:space:]]*([-*][[:space:]]+)?//; /^READ/ s/^READ[[:space:]]+([^[:space:]]+).*/\1/; /^CLEAN/ {s/^CLEAN[[:space:]]+//; s/[[:space:]]+(—|--?)[[:space:]].*//; s#/+$##}') -
```

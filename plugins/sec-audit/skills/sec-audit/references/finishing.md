# Finishing

Owns stages 9 to 13: live reproduction, calibration, affected versions,
writing and editorial review. The exit conditions are `SKILL.md`'s.

All five run on every audit, in order, without waiting for the owner to ask
for any of them. Each changes `data/findings.tsv` first and the prose second:
a value changed only in prose has not changed (`report-template.md`). Agents
write their own files, and you apply what they find to the table by command.

## 9. Live reproduction

- **The bar** is intake's (`decisions.md` D2). Write it as a filter over the
  table and log the filter. The proposed bar, every Critical finding and
  every High one that the least-privileged principals (P0 and P1 in the
  scoring conventions) can reach, is:

  ```sh
  awk -F'\t' 'NR > 1 && $3 == "scored" && ($6 == "Critical" || ($6 == "High" && $11 ~ /^P[01]:/)) {print $1}' data/findings.tsv
  ```

- **The environment** is the disposable one intake chose, created for the
  audit and torn down when the stage ends, unless the owner asks to keep it.
  Where a skill in the ledger governs stage 9, its environment rules decide
  how the environment is built and reached. An environment that the
  ambient-cluster rule in your global instructions puts off limits is never
  used, even to read from. With no disposable environment to hand, ask the
  owner; never use an existing one.
- **The agent** runs on `briefs/live-repro.md`, with OUTPUT `live/LIVE.md`,
  NOTES `live/LIVE.notes.md` and the environment file `tmp/LIVE/env`
  (`coordinator.md`, The shell), and takes the findings in order of
  severity. Its inputs are the IDs, their rows, their verifier and
  adjudication records, and their reproducers. It creates each principal
  with exactly the permissions the finding claims, and acts as that
  principal, so that the claim about permissions is exercised rather than
  bypassed. It also gets, by path, each adjudication file holding `LIVE`
  lines (`verification.md`, Resolving), and answers each line with a
  record. Each finding gets a record headed `### <ID> — <title>` with a
  line `- **Result:** REPRODUCED`, `NOT REPRODUCED` or `PARTIAL`, and the
  reason for anything but REPRODUCED.
- **The outcome goes to the table.** NOT REPRODUCED because the system
  behaves otherwise goes to an adjudicator (`verification.md`, Resolving).
  NOT REPRODUCED because of a limit of the environment is fixed and tried
  again; if it cannot be, ask the owner. PARTIAL corrects the row's
  principal, default exposure or vector before stage 10.

The exit check: the first command prints each finding at the bar that has
not been reproduced, and each one it prints must have an open owner question
in the ledger. The second prints the place of an adjudicator's line that
starts `LIVE` in any other form than its brief's, and the third a `LIVE`
line's cluster with no live record; both must print nothing:

```sh
LC_ALL=C comm -23 <(<the bar filter> | LC_ALL=C sort) <(awk '/^### / {id = $2} /^- \*\*Result:\*\* REPRODUCED/ {print id}' live/LIVE.md | LC_ALL=C sort -u)
(shopt -s nullglob; grep -nE '^[^[:alnum:]]*LIVE' /dev/null adjudication/A*[0-9].md) | grep -vE '^[^:]+:[0-9]+:LIVE C-[0-9]+ — (reach|consumer|order|dependency) — [^ ]+:[0-9]+(-[0-9]+)?$' | cut -d: -f1,2
LC_ALL=C comm -23 <((shopt -s nullglob; cat /dev/null adjudication/A*[0-9].md) | grep -E '^LIVE C-[0-9]+ — ' | cut -d' ' -f2 | LC_ALL=C sort -u) <(awk '/^### / {id = $2} /^- \*\*Result:\*\*/ {print id}' live/LIVE.md | LC_ALL=C sort -u)
```

## 10. Calibration

- **One calibrator** takes every scored row at once, so that each convention
  is applied the same way throughout. It runs on `briefs/calibrator.md`, on
  the model `coordinator.md` (Budget and models) gives judgment work. Its
  inputs are `data/findings.tsv`, `scoring/CONVENTIONS.md`, and the verify,
  cross-check, adjudication and live records. It writes
  `calibration/CALIBRATION.md` and `calibration/vectors.tsv`, one row for
  every finding that stays scored, and its rules are the brief's. A finding
  it moves to hardening is moved by you, by `verification.md`'s ID rules.
- **Apply it.** First make each move to hardening that a ruling calls for.
  Then list any scored row that `vectors.tsv` misses, merge the file into
  the table, and run `verification.md`'s score command:

  ```sh
  LC_ALL=C comm -23 <(awk -F'\t' 'NR > 1 && $3 == "scored" {print $1}' data/findings.tsv | LC_ALL=C sort) <(tail -n +2 calibration/vectors.tsv | cut -f1 | LC_ALL=C sort)
  awk -F'\t' -v OFS='\t' 'FILENAME == ARGV[1] {if (FNR > 1) c[$1] = $0; next} FNR > 1 && ($1 in c) {split(c[$1], x, "\t"); $4 = x[2]; $7 = x[3]; $11 = x[4]; $12 = x[5]} 1' calibration/vectors.tsv data/findings.tsv > data/findings.tsv.new && [ "$(wc -l < data/findings.tsv.new)" = "$(wc -l < data/findings.tsv)" ] && mv data/findings.tsv.new data/findings.tsv
  ```

The exit check: each command prints nothing. The first prints a row whose
score or severity is not what `audit-gate score` computes from its vector, or
whose vector it rejects (`ERR`). The second reads the conventions'
Principals table and prints a principal that names no class in it, or a PR
the class does not have. It also prints an opt-in default exposure without
AT:P, which is the template's convention for opt-in features (`scoring.md`);
where the owner changed that convention, read those rows instead. AT:P
alone is not checked, since it also covers conditions such as a race on a
default deployment. A class whose PR in the conventions is not a single
value, such as a foothold, is checked by reading the row.

```sh
awk -F'\t' 'NR > 1 && $3 == "scored" {print $4; print $7}' data/findings.tsv | CLAUDE_PLUGIN_ROOT="<plugin root>" bash "<plugin root>/tools/run.sh" audit-gate score 2>/dev/null | paste - - | paste <(awk -F'\t' 'NR > 1 && $3 == "scored" {print $1 "\t" $5 " " $6 "\t" ($7 == "-" ? "-" : $8 " " $9)}' data/findings.tsv) - | awk -F'\t' '{sub(/^ERR .*/, "ERR", $4); sub(/^ERR .*/, "ERR", $5)} $2 != $4 || $3 != $5 {print $1 ": table " $2 "; " $3 ", calculator " $4 "; " $5}'
awk -F'\t' 'FILENAME == ARGV[1] {if (split($0, c, "|") > 4) {k = c[2]; gsub(/[[:space:]]/, "", k); pr[k] = match(c[4], /PR:[NLH]/) ? substr(c[4], RSTART, RLENGTH) : "*"} next} FNR > 1 && $3 == "scored" {k = $11; sub(/:.*/, "", k); if (!(k in pr)) print $1 ": principal \"" $11 "\" does not start with a class in the conventions"; else if (pr[k] != "*" && (index($4 "/", "/" pr[k] "/") == 0 || ($7 != "-" && index($7 "/", "/" pr[k] "/") == 0))) print $1 ": " k " is " pr[k] " in the conventions; the vectors say otherwise"; if ($12 ~ /^opt-in/ && $7 != "-" && $7 !~ /\/AT:P(\/|$)/) print $1 ": default exposure \"" $12 "\" needs AT:P in " $7}' scoring/CONVENTIONS.md data/findings.tsv
```

**Re-routing** is your call, and it is not optional. Run
`verification.md`'s band check and stage 9's exit check again. A finding
that calibration moved into the cross-check band goes back through stage 7,
and one it moved to the live bar goes back through stage 9, before stage 11
starts.

## 11. Affected versions

- **Batches** of scored findings, `AV01`, `AV02` and so on, run on
  `briefs/affected-versions.md`. Their inputs are each finding's row and its
  verifier and adjudication records (its sink and its anchors),
  `staging/MANIFEST.tsv` (each release's tree and ref), and the target's
  history in `staging/target`. OUTPUT `affected/<BATCH>.tsv` has a header
  row and the columns `id rel verdict evidence`, one row per finding and
  release; NOTES `affected/<BATCH>.notes.md`.
- **The verdicts** (`PRESENT`, `ABSENT`, `PARTIAL`, and nothing else) and
  their evidence are the brief's. On your side: a release the agent cannot
  settle goes to a follow-up with the open question, and a finding the agent
  reports absent at the audited commit has its cluster reopened
  (`verification.md`, Batches). The section prose repeats what a PARTIAL
  verdict's evidence says differs.
- **Spot-checks.** Check every `ABSENT` that cites a commit, by this
  command, which prints nothing when the release history agrees. Then check
  five other verdicts, chosen with `tail -qn +2 affected/*.tsv | shuf -n 5`,
  by reading the cited lines in the release tree. A failed check sends its
  whole batch back (`coordinator.md`, Checking what agents report).

  ```sh
  awk -F'\t' 'FNR > 1 && $3 == "ABSENT" {print $2 "\t" $4}' affected/*.tsv | while IFS=$'\t' read -r rel ev; do ref=$(awk -F'\t' -v r="$rel" '$2 == "release" && $1 == r {print $4}' staging/MANIFEST.tsv); set -- $(grep -oE '(fixed|introduced) by [0-9a-f]{7,40}' <<<"$ev" | head -1); [ $# = 3 ] || continue; git -C staging/target cat-file -e "$3^{commit}" 2>/dev/null || { echo "$rel: no commit $3"; continue; }; if git -C staging/target merge-base --is-ancestor "$3" "$ref"; then [ "$1" = fixed ] || echo "$rel: $3 is in $ref, so it was not introduced after it"; else [ "$1" = introduced ] || echo "$rel: $3 is not in $ref, so the fix is not there"; fi; done
  ```

- **Fold the verdicts into the table's `rel:<name>` columns:**

  ```sh
  awk -F'\t' -v OFS='\t' 'FILENAME == ARGV[1] {if ($1 != "id") v[$1 SUBSEP $2] = $3; next} FNR == 1 {for (i = 15; i <= NF; i++) r[i] = substr($i, 5)} FNR > 1 {for (i = 15; i <= NF; i++) if (($1 SUBSEP r[i]) in v) $i = v[$1 SUBSEP r[i]]} 1' <(cat affected/*.tsv) data/findings.tsv > data/findings.tsv.new && [ "$(wc -l < data/findings.tsv.new)" = "$(wc -l < data/findings.tsv)" ] && mv data/findings.tsv.new data/findings.tsv
  ```

The exit check: both print nothing. The first prints a malformed row, such
as an `UNKNOWN` or a missing piece of evidence; the second, a scored row
lacking a verdict for some release.

```sh
awk -F'\t' 'FNR == 1 && $0 != "id\trel\tverdict\tevidence" {print FILENAME ": header"} FNR > 1 && (NF != 4 || $3 !~ /^(PRESENT|ABSENT|PARTIAL)$/ || $4 == "" || $4 == "-") {print FILENAME ":" FNR ": " $0}' affected/*.tsv
awk -F'\t' 'NR == 1 {for (i = 15; i <= NF; i++) r[i] = $i} NR > 1 && $3 == "scored" {for (i = 15; i <= NF; i++) if ($i !~ /^(PRESENT|ABSENT|PARTIAL)$/) print $1 ": no verdict for " r[i]}' data/findings.tsv
```

## 12. Write

- **Writers** take batches of findings and run on `briefs/writer.md`. Each
  writes `report/sections/<ID>.md` and `report/hardening/<ID>.md` from
  `report-template.md`, copying every table value from the row, and skips a
  file that already exists, since it is resuming. Their inputs are the rows,
  the claims, the verify, cross-check, adjudication, live and affected
  records, and the staged sources.
- You write the other parts and assemble the report, following
  `report-template.md` (Assembly).

The exit check: the command below prints nothing, the left column being a
finding or hardening note with no file and the right a file with no row.
Then assemble the report, and `audit-gate check`, run as
`completeness-gate.md` (The gate) gives it, exits 0, each problem going back
as that file's table says. Both pass before editorial review starts, so that
reviewers read a report the gate has already cleared of broken links,
anchors and table mismatches; stage 14 runs the gate again.

```sh
LC_ALL=C comm -3 <(awk -F'\t' 'NR > 1 && $3 == "scored" {print "report/sections/" $1 ".md"} NR > 1 && $3 == "hardening" {print "report/hardening/" $1 ".md"}' data/findings.tsv | LC_ALL=C sort) <(ls report/sections/*.md report/hardening/*.md 2>/dev/null | LC_ALL=C sort)
```

## 13. Editorial review

- **Reviewers** take batches of sections, `E01`, `E02` and so on, on
  `briefs/editorial.md`, with OUTPUT `editorial/<BATCH>.md` and NOTES
  `editorial/<BATCH>.notes.md`. Each gets its sections, the table, and, for
  checks across sections, the overview, the hardening index and the
  prior-audit ledger (`report/parts/overview.md`, `hardening-index.md` and
  `prior-audit.md`), never the whole report. What a reviewer checks, what it
  may fix and what it must flag, and its line forms, are the brief's.
- **Flags are adjudicated.** Each goes to an adjudicator on
  `briefs/adjudicator.md`, as `adjudication/EA01.md` and so on, whose
  dispatch names the flagged IDs and the editorial file by path, never the
  flag's text. The ruling is a block headed `=== <ID>`, in the form
  `briefs/adjudicator.md` gives, whose VERDICT is the row's resulting status,
  `RETIRED` included. Apply each ruling to the table first: status and
  IDs by `verification.md`'s rules, vectors through the score command and
  stage 10's checks; stage 7's exit check reads the ruling as that part's
  standing verdict (`verification.md`, Resolving). Then copy the values back
  into the prose and assemble again.

The exit check: all three print nothing. The first reads each reviewer's
OUTPUT against the line forms of `briefs/editorial.md`, and prints any line
in another form, such as a bulleted or bold `FLAG`, and any `## <ID>`
section with no line, because a flag written another way would never reach
an adjudicator. The second prints a flag with no ruling, and the third a
`TABLE` change that the table does not yet hold.

```sh
(shopt -s nullglob; awk 'function end() {if (s != "" && !n) print f ": ## " s " has no FIX, TABLE, FLAG or No change. line"} FNR == 1 {end(); f = FILENAME; s = ""; n = 0; if (/^# /) next} /^[[:space:]]*$/ {next} /^## / {end(); s = $2; n = 0; next} s != "" && ($0 == "No change." || ((/^(FIX|FLAG) [^ ]+ — ./ || /^TABLE [^ ]+ title — .+ → ./) && $2 == s)) {n++; next} {print f ":" FNR ": " $0} END {end()}' /dev/null editorial/E*[0-9].md)
LC_ALL=C comm -23 <(grep -h '^FLAG ' editorial/E*[0-9].md | cut -d' ' -f2 | LC_ALL=C sort -u) <((shopt -s nullglob; awk 'FNR == 1 {c = ""} /^=== / {c = $2; next} /^VERDICT:/ && c != "" {print c; c = ""}' /dev/null adjudication/EA*[0-9].md) | LC_ALL=C sort -u)
grep -h '^TABLE ' editorial/E*[0-9].md | while read -r _ id col _ rest; do NEW="${rest##*→ }" awk -F'\t' -v id="$id" -v col="$col" 'NR == 1 {for (i = 1; i <= NF; i++) if ($i == col) k = i} NR > 1 && $1 == id {if (!k) print id ": no column " col; else if ($k != ENVIRON["NEW"]) print id " " col ": the table still has \"" $k "\""}' data/findings.tsv; done
```

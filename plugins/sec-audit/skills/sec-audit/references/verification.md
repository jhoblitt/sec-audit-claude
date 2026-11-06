# Verification

Owns stages 6 and 7: claim-only text, batches, the two questions every
verdict answers, cross-checks, follow-ups and adjudication, and the rows
they write to `data/findings.tsv`. A verifier's own verdicts, record block
and reproducers are `briefs/verifier.md`'s. The exit conditions are
`SKILL.md`'s.

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
  the clusters with no verifier block yet, six to a line, after batch `<N>`.
  A block counts only once a line of its own starts `VERDICT:`, so a block
  written in another form is verified again:

  ```sh
  LC_ALL=C comm -23 <(tail -n +2 clusters.tsv | cut -f1 | LC_ALL=C sort) <((shopt -s nullglob; awk 'FNR == 1 {c = ""} /^=== / {c = $2; next} /^VERDICT:/ && c != "" {print c; c = ""}' /dev/null verify/V*[0-9].md) | LC_ALL=C sort -u) | xargs -n 6 | awk -v n=<N> 'NF {printf "V%02d %s\n", n + ++k, $0}'
  ```
- Dispatch every batch at once, within the budget, on `briefs/verifier.md`.
  Its inputs are its clusters' `claims/` files, `scoring/CONVENTIONS.md`, and
  `staging/MANIFEST.tsv`, which locates every staged source. Its OUTPUT is
  `verify/<BATCH>.md`, its NOTES `verify/<BATCH>.notes.md`, its reproducers
  go under `verify/<BATCH>/`, and its environment file is `tmp/<BATCH>/env`
  (`coordinator.md`, The shell). It never gets the discovery files, another
  batch's records or the table.
- A cluster that a later stage reopens goes to an adjudicator (Resolving,
  below) with the new evidence, so that its new ruling is the cluster's last
  block. A cluster is reopened by a challenge to its ruling, such as a
  refutation the gap sweep disputes, and by a new member that joins it after
  its verdict, such as a wider reach a sweeper found: stage 6's exit check
  lists every such member.

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

## Verdicts, records and reproducers

`briefs/verifier.md` owns them: its Verdicts table, its eleven-line record
block (Records), whose first five lines are the cross-check block's so that
one command reads both, and what a reproducer is and how it runs
(Reproducers). The commands in this file read records in that form. On the
coordinator's side:

- UNSURE is not a verdict: Resolving, below, settles every one.
- A distinct defect a verifier notices outside its clusters is a candidate
  with its batch as the ID stem (`V03-01`), and goes through stages 5 to 7.
- A defect only a running system shows, `REPRO: live: <what it needs>`, goes
  to stage 9 if it is at the bar.
- Before a `SCORED` block enters the table, run its reproducer script again
  (`coordinator.md`, The shell) and log the PASS line. One that does not
  pass sends the cluster back to its verifier.

Stage 6's exit check: both commands print nothing. An UNSURE block counts
here as the cluster's record; it is no verdict, and stage 7 settles it. The
first prints a cluster with no verifier block that has a `VERDICT:` line.
The second prints a member that no MEMBERS line of its cluster's standing
blocks covers: a candidate that joined the cluster after its verdict, whose
cluster is reopened (Batches).

```sh
LC_ALL=C comm -23 <(tail -n +2 clusters.tsv | cut -f1 | LC_ALL=C sort) <((shopt -s nullglob; awk 'FNR == 1 {c = ""} /^=== / {c = $2; next} /^VERDICT:/ && c != "" {print c; c = ""}' /dev/null verify/V*[0-9].md) | LC_ALL=C sort -u)
(shopt -s nullglob; awk -F'\t' 'FILENAME == "clusters.tsv" {if (FNR > 1) {n = split($3, m, ","); for (i = 1; i <= n; i++) want[$1, m[i]]} next} FNR == 1 {c = ""} /^=== / {split($0, w, " "); c = w[2]; if (src[c] != FILENAME) {src[c] = FILENAME; for (k in have) {split(k, q, SUBSEP); if (q[1] == c) delete have[k]}} next} c != "" && /^MEMBERS: / {n = split(substr($0, 10), m, ","); for (i = 1; i <= n; i++) {gsub(/[[:space:]]/, "", m[i]); have[c, m[i]]}} END {for (k in want) {split(k, q, SUBSEP); if ((q[1] in src) && !(k in have)) print q[1] ": " q[2] " has no standing verdict"}}' clusters.tsv /dev/null verify/V*[0-9].md verify/U*[0-9].md adjudication/A*[0-9].md) | LC_ALL=C sort
```

## Stage 7: cross-checks

- **The band** is intake's (`decisions.md` D3). Write it as a regular
  expression over `sev31` and log it, as stage 9 does its bar; the proposed
  band, Critical and High, is `^(Critical|High)$`. At least one model from a
  vendor other than the verifier's re-verifies every scored cluster in it
  from source. This lists the clusters in the band that no cross-check has
  covered yet; it must print nothing before the stage ends:

  ```sh
  LC_ALL=C comm -23 <(awk -F'\t' -v band='<the band>' 'NR > 1 && $3 == "scored" && $6 ~ band {print $2}' data/findings.tsv | LC_ALL=C sort -u) <(cat xcheck/*.md 2>/dev/null | awk '/^=== / {print $2}' | LC_ALL=C sort -u)
  ```

- **The dispatch.** Batches `XC01`, `XC02` and so on go through the model's
  CLI on `briefs/cross-checker.md`, under `discovery.md`'s rules for a CLI
  (read-only flags, a fresh directory, nothing pasted, failure handling).
  Its fresh directory holds copies of `scoring/CONVENTIONS.md`, so that its
  vector follows the same conventions, of the batch's claims files, and of
  `coverage/links.tsv`, whose links it never opens; the prompt names them by
  path. It gets the claim-only text and never the verifier's record, so its
  answer is independent. Its stdout is `xcheck/<BATCH>-<slug>.md`, and its
  stderr the `.log` beside it. The shape, with `<clusters>` the batch's
  cluster IDs:

  ```sh
  (d=cli/gpt-XC01; o=xcheck/XC01-gpt; b="<plugin root>/skills/sec-audit/references/briefs/cross-checker.md"; [ -s "$b" ] || { echo "no brief at $b"; exit 2; }; w=$PWD; mapfile -t m < <(awk -F'\t' -v w="$w" 'NR > 1 && $2 == "dep" && $6 == "local" && $5 !~ /^staging\// {print "--add-dir"; print ($5 ~ /^\// ? $5 : w "/" $5)}' staging/MANIFEST.tsv); rm -rf -- "$d" && mkdir -p -- "$d/claims" && for c in <clusters>; do cp "claims/$c.md" "$d/claims/" || exit 2; done && cp scoring/CONVENTIONS.md coverage/links.tsv "$d/" && cd "$d" || exit 2; timeout 2h <cli> --model <model> --allow-tool view,grep,glob --no-ask-user --add-dir "$w/staging/target" --add-dir "$w/staging/deps" "${m[@]}" -p "$(cat "$b"; printf '\nRepository: %s\nDependencies: %s\nScoring conventions: %s\nClaims files: %s\nSymlinks, never open: %s\n' "$w/staging/target" "$w/staging/deps" "$PWD/CONVENTIONS.md" "$PWD/claims" "$PWD/links.tsv")" > "$w/$o.md" 2> "$w/$o.log")
  ```
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
  t=$(mktemp) && (shopt -s nullglob; awk 'function out() {if (c != "") print c "\t" f "\t" v "\t" x} FNR == 1 {out(); c = ""} /^=== / {out(); c = $2; f = FILENAME; v = x = "-"; sv = sx = 0} /^VERDICT:/ && !sv {v = $2; sv = 1} /^VECTOR31:/ && !sx {x = $2; sx = 1} END {out()}' /dev/null xcheck/*.md) > "$t" && cut -f4 "$t" | CLAUDE_PLUGIN_ROOT="<plugin root>" bash "<plugin root>/tools/run.sh" audit-gate score 2>/dev/null | paste "$t" - | awk -F'\t' 'FILENAME == ARGV[1] {if (FNR > 1 && $3 == "scored" && !(($2, $6) in s)) {s[$2, $6]; t[$2] = t[$2] (t[$2] == "" ? "" : ",") $6} next} {split($5, g, " "); b = (g[1] == "-" || g[1] == "ERR") ? g[1] : g[2]; if (!($3 == "CONFIRMED" && b == t[$1])) print $1 "\t" $2 "\t" $3 " " (b == "" ? "-" : b) ", table " (t[$1] == "" ? "-" : t[$1])}' data/findings.tsv -
  ```

- A CONFIRMED in the band adds the model to the row's `confirmed_by`.

## Resolving: follow-ups and adjudication

Nothing is left open, and none of it waits for the owner to ask.

- **An UNSURE** from a verifier goes to a follow-up verifier: a fresh agent
  on `briefs/verifier.md`, in a batch `U01`, `U02` and so on, with OUTPUT
  `verify/<BATCH>.md`, given the claim and the link the block names, and,
  for a split cluster, every other part's block, which it restates. A
  follow-up that is UNSURE again goes to an adjudicator.
- **A cross-check the command above prints**, and **any dispute**, meaning two
  records with different verdicts or with vectors in different bands, go to
  an adjudicator. Run it on `briefs/adjudicator.md` as `adjudication/A01.md`,
  `A02.md` and so on, and give it the claim, every record on the cluster, and
  the code. It checks each contested fact itself and rules `SCORED`,
  `HARDENING` or `REFUTED` in the same block; it never splits the difference.
  A question that only a running system can settle is a `LIVE` line in its
  OUTPUT (`briefs/adjudicator.md`). It goes to live reproduction first, by
  the file's path (`finishing.md`), and the adjudicator rules on the
  result.
- A cluster's standing verdicts are its blocks in the last file that holds
  any, in the order verifier, follow-up, adjudicator: one per part of a split
  cluster. An editorial ruling (`adjudication/EA<NN>.md`, `=== <ID>`)
  replaces the standing block whose MEMBERS it repeats with its own verdict,
  and a RETIRED one removes it. Only the latest ruling on an ID counts, and
  rulings apply in file order, `EA01.md` before `EA02.md` and top to bottom
  within one, so the latest ruling on a part stands even when it names a
  new ID: F-14 ruled HARDENING moves the part to H-06, and H-06 then ruled
  SCORED leaves it scored. The table holds one row per standing verdict,
  with at most one refuted row per cluster.

The exit check: these and the band check above each print nothing. The first
prints a standing UNSURE, a verdict word it does not know, a cluster whose
standing verdicts and table rows differ in number for any status, a block
with no line of its own starting `VERDICT:`, a table row whose cluster has
no verdict at all, and an editorial ruling whose ID and MEMBERS match no
standing block; it reads editorial rulings as the rule above says, so a
status that stage 13 changed still agrees when the completeness gate runs
this check again. The second prints a flagged cross-check with no ruling.

```sh
(shopt -s nullglob; awk -F'\t' 'function end() {if (g && !(g in bv)) print bc[g] ": a block with no VERDICT line in " bf[g]; g = 0} function key(s,   a, n, i, j, x, o) {gsub(/[[:space:]]/, "", s); n = split(s, a, ","); for (i = 2; i <= n; i++) {x = a[i]; for (j = i - 1; j >= 1 && a[j] > x; j--) a[j + 1] = a[j]; a[j + 1] = x} o = ""; for (i = 1; i <= n; i++) o = o "," a[i]; return o} BEGIN {split("scored hardening refuted unsure", V, " ")} FILENAME == ARGV[1] {if (FNR > 1) {cl[$1] = $2; if ($3 != "retired") t[$2, $3]++} next} FNR == 1 {end()} /^=== / {end(); split($0, w, " "); g = ++ng; bc[g] = w[2]; bf[g] = FILENAME; if (FILENAME ~ /(^|\/)EA[0-9]+\.md$/) e[w[2]] = g; else {if (src[w[2]] != FILENAME) {src[w[2]] = FILENAME; bl[w[2]] = ""} bl[w[2]] = bl[w[2]] " " g} next} g && /^VERDICT:/ && !(g in bv) {split($0, w, " "); bv[g] = tolower(w[2]); next} g && /^MEMBERS:/ && !(g in bm) {bm[g] = key(substr($0, 9))} END {end(); for (h = 1; h <= ng; h++) {x = bc[h]; if (!(x in e) || e[x] != h) continue; c = cl[x]; m = split(bl[c], gl, " "); for (j = 1; j <= m; j++) if (bm[gl[j]] == bm[h]) break; if (c == "" || bm[h] == "" || j > m) {print x ": its editorial ruling in " bf[h] " matches no standing block" (c == "" ? "" : " of " c); continue} if (bv[h] !~ /^(scored|hardening|refuted|retired)$/) {print x ": verdict " bv[h] " in " bf[h]; continue} bv[gl[j]] = bv[h]; ov[gl[j]] = bf[h]} for (c in src) {for (i = 1; i <= 4; i++) n[c, V[i]] = 0; m = split(bl[c], gl, " "); for (j = 1; j <= m; j++) {v = bv[gl[j]]; if (v == "" || (v == "retired" && gl[j] in ov)) continue; if (v !~ /^(scored|hardening|refuted|unsure)$/) print c ": verdict " v " in " bf[gl[j]]; else n[c, v]++}} for (k in t) {split(k, q, SUBSEP); if (q[1] != "-" && !(q[1] in src)) print q[1] ": " t[k] " " q[2] " in the table, and no verdict block"} for (c in src) {if (n[c, "unsure"]) print c ": standing UNSURE in " src[c]; for (i = 1; i <= 3; i++) {a = n[c, V[i]] + 0; b = t[c, V[i]] + 0; if (V[i] == "refuted") {a = a > 0; b = b > 0} if (a != b) print c ": " a " " V[i] " in " src[c] ", " b " in the table"}}}' data/findings.tsv verify/V*[0-9].md verify/U*[0-9].md adjudication/A*[0-9].md adjudication/EA*[0-9].md)
<the cross-check command above> | cut -f1 | LC_ALL=C sort -u | LC_ALL=C comm -23 - <((shopt -s nullglob; awk 'FNR == 1 {c = ""} /^=== / {c = $2; next} /^VERDICT:/ && c != "" {print c; c = ""}' /dev/null adjudication/A*[0-9].md) | LC_ALL=C sort -u)
```

## Writing the table

`data/findings.tsv` (`report-template.md`) gets its rows from the standing
verdicts, and changes whenever one does. Create it once, with one `rel:`
column per release, then check the column order against the release set
(`decisions.md` D1). The command leaves an existing table alone:

```sh
mkdir -p data && { [ -e data/findings.tsv ] || { printf 'id\tcluster\tstatus\tcvss31\tscore31\tsev31\tcvss40\tscore40\tsev40\ttitle\tprincipal\tdefault_exposure\tfound_by\tconfirmed_by'; awk -F'\t' 'NR > 1 && $2 == "release" {printf "\trel:%s", $1}' staging/MANIFEST.tsv; echo; } > data/findings.tsv; }
```

**IDs.** `report-template.md` (IDs) governs their shape and freezing; these
rules choose one, and the rows command below applies them:

- A scored row takes the next `F` ID and a hardening row the next `H` ID,
  in cluster order at its first verdict, never in score order. A cluster
  with a `prior_id` keeps that ID, and a refuted row takes its cluster's ID
  (`report-template.md`).
- A row that moves between scored and hardening, or out of refuted, takes the
  next ID of its new kind. The old finding ID stays as a `retired` row, and
  Not reported says where it went.
- A fixed prior finding is a `retired` row with cluster `-`, which you write
  from the prior-findings agents' `## Fixed` tables.
- The next ID counts every ID the table and the prior reports have issued
  (`p=H` for hardening):

  ```sh
  p=F; grep -rhoE "(^|[^A-Za-z0-9-])$p-[0-9]+" data/findings.tsv staging/prior | grep -oE '[0-9]+$' | sort -n | tail -1 | awk -v p="$p" '{n = $1} END {printf "%s-%02d\n", p, n + 1}'
  ```

**Rows.** The table holds one row per standing verdict (Resolving, below),
and this command keeps it so: it appends a row for each standing block that
has none, refreshes each existing row from its block, retires a row whose
verdict moved, and drops a refuted row whose cluster is no longer refuted,
by the IDs above. It takes `found_by` from `found_by.tsv` (`discovery.md`,
Found by), and leaves alone every cluster that an editorial ruling or
`TABLE` line has touched, whose rows change by hand under the same rules.
It writes `-` in `confirmed_by`, the scores and the `rel:` columns. Run it
after every round of verdicts, with the prefixes `FP` and `HP` set to the
prior audit's where it has its own (`F` and `H` otherwise):

```sh
(shopt -s nullglob; s=$(mktemp) && FP=F HP=H FMAX=$(grep -rhoE '(^|[^A-Za-z0-9-])F-[0-9]+' staging/prior 2>/dev/null | grep -oE '[0-9]+$' | sort -n | tail -1) HMAX=$(grep -rhoE '(^|[^A-Za-z0-9-])H-[0-9]+' staging/prior 2>/dev/null | grep -oE '[0-9]+$' | sort -n | tail -1) TOUCHED=$({ grep -hoE '^TABLE [^ ]+' /dev/null editorial/E*[0-9].md | cut -d' ' -f2; awk '/^=== / {print $2}' /dev/null adjudication/EA*[0-9].md; } | paste -sd, -) awk -F'\t' -v OFS='\t' '
function put(r, x,   i) {row[r] = x[1]; for (i = 2; i <= nf; i++) row[r] = row[r] OFS x[i]}
function fill(x, s, h) {if (s != "scored") {x[4] = x[7] = "-"} else {if (x[4] != b[h, "VECTOR31"] || x[7] != b[h, "VECTOR40"]) x[5] = x[6] = x[8] = x[9] = "-"; x[4] = b[h, "VECTOR31"]; x[7] = b[h, "VECTOR40"]} x[10] = b[h, "TITLE"]; x[11] = b[h, "PRINCIPAL"]; x[12] = b[h, "DEFAULT"]}
function add(c, s, h,   x, i) {x[1] = s == "refuted" ? c : (pid[c] != "-" && pid[c] != "" && !(pid[c] in used)) ? pid[c] : sprintf("%s-%02d", pre[s], ++mx[s]); used[x[1]]; x[2] = c; x[3] = s; for (i = 4; i <= nf; i++) x[i] = "-"; fill(x, s, h); x[13] = (c in fb) ? fb[c] : "-"; app[++na] = x[1]; for (i = 2; i <= nf; i++) app[na] = app[na] OFS x[i]}
BEGIN {n = split(ENVIRON["TOUCHED"], t, ","); for (i = 1; i <= n; i++) tid[t[i]]; pre["scored"] = ENVIRON["FP"]; pre["hardening"] = ENVIRON["HP"]; mx["scored"] = ENVIRON["FMAX"] + 0; mx["hardening"] = ENVIRON["HMAX"] + 0}
FILENAME == "clusters.tsv" {if (FNR > 1) {ord[++nc] = $1; pid[$1] = $2} next}
FILENAME == "found_by.tsv" {fb[$1] = $2; next}
FILENAME == "data/findings.tsv" {if (FNR == 1) {hdr = $0; nf = NF; next} row[++nr] = $0; id[nr] = $1; cl[nr] = $2; st[nr] = $3; used[$1]; for (k in pre) {u = substr($1, length(pre[k]) + 2); if (index($1, pre[k] "-") == 1 && u ~ /^[0-9]+[a-z]?$/ && u + 0 > mx[k]) mx[k] = u + 0} next}
FNR == 1 {c = ""}
/^=== / {split($0, w, " "); c = w[2]; if (src[c] != FILENAME) {src[c] = FILENAME; bl[c] = ""} bl[c] = bl[c] " " (++g); next}
c != "" && /^(VERDICT|VECTOR31|VECTOR40|TITLE|PRINCIPAL|DEFAULT): / {k = substr($0, 1, index($0, ":") - 1); if (!((g, k) in b)) b[g, k] = substr($0, index($0, ":") + 2)}
END {for (i = 1; i <= nc; i++) {c = ord[i]; if (!(c in src)) continue; for (r = 1; r <= nr; r++) if (cl[r] == c && (id[r] in tid)) break; if (r <= nr) continue; split("", B); split("", nb); m = split(bl[c], gl, " "); for (j = 1; j <= m; j++) {v = tolower(b[gl[j], "VERDICT"]); if (v ~ /^(scored|hardening|refuted)$/) B[v, ++nb[v]] = gl[j]} for (q = 1; q <= 3; q++) {s = q == 1 ? "scored" : q == 2 ? "hardening" : "refuted"; k = 0; for (r = 1; r <= nr; r++) if (cl[r] == c && st[r] == s) {k++; split(row[r], x, "\t"); if (s == "refuted" ? (k > 1 || !nb[s]) : k > nb[s]) {if (s == "refuted") del[r]; else {x[3] = "retired"; x[4] = x[5] = x[6] = x[7] = x[8] = x[9] = "-"; put(r, x)}} else {fill(x, s, B[s, k]); put(r, x)}} for (k++; k <= (s == "refuted" ? (nb[s] > 0) : nb[s]); k++) add(c, s, B[s, k])}} print hdr; for (r = 1; r <= nr; r++) if (!(r in del)) print row[r]; for (a = 1; a <= na; a++) print app[a]}
' clusters.tsv found_by.tsv data/findings.tsv /dev/null verify/V*[0-9].md verify/U*[0-9].md adjudication/A*[0-9].md > "$s" && [ "$(head -1 "$s")" = "$(head -1 data/findings.tsv)" ] && mv "$s" data/findings.tsv)
```

The columns it writes, and the one you write:

| Column | Value |
| --- | --- |
| `id` | by the ID rules above |
| `cluster`, `status` | the block's cluster, and its verdict in lower case |
| `cvss31`, `cvss40` | VECTOR31 and VECTOR40 on a scored row; `-` on any other |
| `score31` to `sev40` | `-`, until the score command below writes them; never by hand |
| `title`, `principal`, `default_exposure` | TITLE, PRINCIPAL and DEFAULT; `principal` starts with its class |
| `found_by` | the row's cluster's line in `found_by.tsv` |
| `confirmed_by` | yours, once the command has run: the verifier's model, each model whose cross-check was CONFIRMED, and the adjudicator's model when it ruled; `-` on a refuted row |
| `rel:<name>` | `-` until stage 11 |

**Scores.** After any vector is written or changed, at this stage or a later
one, fill every scored row's scores and severities from `audit-gate score`
(`scoring.md`, The calculator rule). The command feeds it the vectors from
the table through standard input, and rewrites the table only if the
calculator answered every vector and no row was lost:

```sh
s=$(mktemp) && awk -F'\t' 'NR > 1 && $3 == "scored" {print $4; print $7}' data/findings.tsv > "$s.in" && { CLAUDE_PLUGIN_ROOT="<plugin root>" bash "<plugin root>/tools/run.sh" audit-gate score < "$s.in" > "$s" 2>/dev/null; [ $? -le 1 ]; } && [ "$(wc -l < "$s")" = "$(wc -l < "$s.in")" ] && awk -F'\t' -v OFS='\t' 'FILENAME == ARGV[1] {l[FNR] = $0; next} FNR > 1 && $3 == "scored" {split(l[++k], a, " "); split(l[++k], b, " "); $5 = a[1]; $6 = (a[1] == "-" || a[1] == "ERR") ? a[1] : a[2]; $8 = b[1]; $9 = (b[1] == "-" || b[1] == "ERR") ? b[1] : b[2]} 1' "$s" data/findings.tsv > data/findings.tsv.new && [ "$(wc -l < data/findings.tsv.new)" = "$(wc -l < data/findings.tsv)" ] && mv data/findings.tsv.new data/findings.tsv
```

`ERR` in a row is a vector the calculator rejects; it goes back to whoever
wrote it. `finishing.md` (Calibration) gives the check that every score
matches its vector.

# Report template

Owns the audit report's format and content. `audit-gate check` parses
`report/REPORT.md` by the markers in the skeletons below and compares it with
`data/findings.tsv`, so this template is a contract: copy every marker byte for
byte, with the em dash (`—`), bold, backticks and spacing as shown. A marker
written any other way is a gate problem or, worse, invisible to the gate. In a
skeleton, `<…>` is a slot to fill. A value not yet known is written as a
`{{UPPER_SNAKE}}` placeholder; none may remain in a delivered report, and the
gate fails a report that holds one.

## The findings table

`data/findings.tsv` is the single source of truth. Every stage writes its
result into the row: a verdict, a vector, a title or an editorial ruling
changed only in prose is not changed. TSV, UTF-8, one header row, no quoting,
`-` for an empty cell. Columns, in order:

```text
id cluster status cvss31 score31 sev31 cvss40 score40 sev40 title principal default_exposure found_by confirmed_by rel:<name>...
```

- `status` is `scored`, `hardening`, `retired` or `refuted`. Only `scored`
  rows carry vectors; every other row holds `-` in the six CVSS columns.
- `score31` and `score40` have one decimal (`7.1`). `sev31` and `sev40` are
  `None`, `Low`, `Medium`, `High` or `Critical`, title case, in FIRST's bands
  for both versions. Both come from `audit-gate score` (`scoring.md`).
- When the owner declined CVSS 4.0, `cvss40`, `score40` and `sev40` are `-`.
- One `rel:<name>` column per release, in release-set order, holding
  `PRESENT`, `ABSENT` or `PARTIAL`; `-` is allowed only on a row that is not
  scored. A verdict's evidence and notes go in the section prose.
- `title` is one specific line: what principal can do what, through what. The
  section heading, the overview and the hardening index repeat it verbatim.
- `default_exposure` is short, such as `default` or `opt-in (<knob>)`,
  because the overview prints it.
- `found_by` and `confirmed_by` follow the attribution rule below.

## IDs

- An ID is `<PREFIX>-<N>[<letter>]`: `F-07`, `F-07b`, `H-03`. The gate finds
  sections by that shape alone. A letter suffix splits one cluster into
  findings that share a root cause and a fix; distinct defects get distinct
  IDs.
- An ID is frozen at first assignment: a rescore never renumbers anything, and
  a retired ID is never reused. New IDs start above the highest ID ever
  issued, by this audit or a prior one.
- On a re-audit, a finding that is still present keeps its prior ID, prefix
  included.
- No level-3 or level-4 heading other than a section's holds an ID-shaped
  token, a CVE number included: the gate reads such a heading as a section.

## The report

Eleven parts, in this order. Each `##` heading carries the anchor shown above
it, because internal links resolve only to explicit `<a id>` anchors, never to
heading slugs.

### 1. Header

```markdown
# Security audit: <owner>/<repo>

> **Confidential.** <Undisclosed vulnerabilities in <owner>/<repo>: handle this report as its [security policy](<pinned link>) requires, or, with no policy, share it with its maintainers only.>

- **Target:** [<owner>/<repo>](https://github.com/<owner>/<repo>) at `<ref>`
- **Date:** <YYYY-MM-DD>

## Contents

- [Executive summary](#summary)
- [Finding overview](#overview)
- [Prior-audit ledger](#prior-audit)
- [Scope and method](#scope)
- [Scoring conventions](#scoring)
- [Detailed findings](#findings)
- [Hardening notes](#hardening)
- [Not reported](#not-reported)
- [Limitations](#limitations)
- [Remediation priorities and disclosure guidance](#remediation)
```

`<ref>` is the target's ref in `staging/MANIFEST.tsv`, the one every target
code link cites.

### 2. Executive summary

```markdown
<a id="summary"></a>

## Executive summary

The audit confirmed **<N> scored findings**: <C> Critical, <H> High, <M> Medium and <L> Low, and
**<K> [hardening notes](#hardening-index)**. <The prior audit's findings: how many are fixed, partly fixed and still present.> <Affected versions, in one sentence.> <Attribution: each model, and how many findings it found blind.>

<Recurring root causes: a short list, each cause linking the findings it explains.>
```

The two count sentences are fixed wording whatever the counts: every severity
is named even at 0, and a count of 1 still reads `hardening notes`. They may
wrap. The severity counts are of `sev31`.

### 3. Finding overview

```markdown
<a id="overview"></a>

## Finding overview

| ID | CVSS 3.1 | CVSS 4.0 | Severity | Finding | Affected (<rel1> / <rel2> …) | Found by | Default exposure |
|---|---|---|---|---|---|---|---|
| [<ID>](#<lowercase id>) | <score31> | <score40> | <sev31> | <title> | <V1> / <V2> … | <found_by> | <default_exposure> |
```

- One row per `scored` row; no other row appears.
- The header row is exactly as shown, with the releases named in `rel:` column
  order.
- The CVSS cells are the bare score with one decimal, or `—` when the row has
  none. Severity is `sev31`.
- Finding, Found by and Default exposure are the row's values verbatim, with
  each `|` escaped as `\|`.
- Each Affected verdict is `yes` for PRESENT, `partly` for PARTIAL and `no`
  for ABSENT, in header order.
- Rows are sorted by `score31` descending, then `score40` descending (`—`
  last), then by ID numerically, so F-9 precedes F-10:
  `awk -F'\t' '$3=="scored"' data/findings.tsv | sort -t$'\t' -k5,5gr -k8,8gr -k1,1V`.

### 4. Prior-audit ledger

```markdown
<a id="prior-audit"></a>

## Prior-audit ledger

<The prior audit: its date and where it is.>

| Prior ID | Status | Fixing commit | PR | Now |
|---|---|---|---|---|
| <ID> | <fixed, partly fixed or still present> | <[<short commit>](<commit URL>), or —> | <[#<n>](<PR URL>), or —> | <[<ID>](#<lowercase id>), or retired> |
```

One row per prior ID. A finding still present or partly fixed keeps its ID,
and Now links its section. A fixed prior ID is a `retired` row in the table.
With no prior audit, the part reads `None.`

### 5. Scope and method

```markdown
<a id="scope"></a>

## Scope and method

**Scope.** <The target at its ref: its tracked-file count, and each exclusion with its reason.>

| Source | Kind | Ref | Why read |
|---|---|---|---|
| [<owner>/<repo>](https://github.com/<owner>/<repo>) | <target, dep, release or prior> | `<ref>` | <what in the target depends on it> |

**Method.** <One numbered item per stage that ran: its counts (agents, candidates, clusters, verdicts, reproducers) and its models by name, and the disposable environment any live reproduction used.>
```

One source row per `staging/MANIFEST.tsv` row. A `failed` row also appears
under Limitations, with its recorded attempt.

### 6. Scoring conventions

```markdown
<a id="scoring"></a>

## Scoring conventions

<The body of scoring/CONVENTIONS.md, verbatim, its dated changes included.>
```

### 7. Detailed findings

```markdown
<a id="findings"></a>

## Detailed findings

<One paragraph: sections follow the overview's order; code links are pinned to the refs in Scope and method; what Found by and Confirmed by record.>

<a id="<lowercase id>"></a>

### <ID> — <title>

| Field | Value |
|---|---|
| **CVSS 3.1** | `<vector>` — **<score> <Severity>** |
| **CVSS 4.0** | `<vector>` — **<score> <Severity>** |
| **Principal** | <principal>; <every other prerequisite> |
| **Default exposure** | <default_exposure> |
| **Affected versions** | <rel> **<VERDICT>**; <rel> **<VERDICT>** … [— <note>] |
| **Prior audit** | <new, or the prior ID's status and what changed> |
| **Found by** | <found_by> |
| **Confirmed by** | <confirmed_by> |

**Defect.** <What is wrong, and where, in 2–4 sentences.>

**Trace.** <Source to sink in 3–6 hops, each a pinned code link with what happens there.>

**Reproduction.** <How it was reproduced.>

**Impact.** <What the principal gains, in 1–3 sentences.>

**Fix.** <The shape of the fix, in a sentence or two.>
```

- The anchor, one blank line, then `### <ID> — <title>`: an em dash with a
  space each side, no bold, no colon, no anchor inside the heading. No other
  anchor sits between the two. The gate recognizes no other form.
- Nothing inside a section is a heading at level 3 or above; one ends the
  section. Use the bold lead-ins.
- The CVSS and Affected versions cells are the row's values. When `cvss40` is
  `-`, the row is `| **CVSS 4.0** | — |`.
- Affected versions gives one `<rel> **<VERDICT>**` per `rel:` column, in
  capitals, joined by semicolons. The bracketed `— <note>` is optional and
  comes last, for example to say what PARTIAL lacks.
- Principal starts with the row's `principal`. Where PR:H is given to a
  principal that already holds equal power, say so: the principal
  equivalence.
- A Trace hop into a dependency links the dependency's staged source.
- Reproduction is one of:
  - a unit test: its name, the command that runs it, its PASS line, and an
    excerpt of at most 25 lines;
  - a live reproduction: the steps and the observed result;
  - neither: say the finding is static-only, and why.
- Fix is a shape, not a patch. A disclosure's diff is not part of the report
  (part 11).

### 8. Hardening notes

```markdown
<a id="hardening"></a>

## Hardening notes

<One paragraph: real observations that are not exploitable, so not scored.>

<a id="hardening-index"></a>

### Index

| ID | Note |
|---|---|
| [<ID>](#<lowercase id>) | <title> |

### Notes

<a id="<lowercase id>"></a>

#### <ID> — <title>

<2–4 sentences with pinned links: the observation, the verdict that placed it here, and the recommendation.> Found by <found_by>; confirmed by <confirmed_by>.
```

- The index table follows `### Index` with nothing between them but blank
  lines: prose there empties the index.
- One index row and one note per `hardening` row, in ID order. The note's
  heading follows the finding rules at level 4.
- With no notes, `None.` replaces the index table and `### Notes` is omitted.
- `### Index` and `### Notes` keep heading levels consecutive.

### 9. Not reported

```markdown
<a id="not-reported"></a>

## Not reported

| ID | Status | Reason |
|---|---|---|
| <ID> | <retired or refuted> | <One sentence: why; for a retired ID, also where its residual went.> |
```

One row per `retired` and `refuted` row, retired IDs included, so that no ID
is reused or mistaken for a gap. An ID listed here gets no section, overview
row or index row. With no such rows, the part reads `None.`

### 10. Limitations

```markdown
<a id="limitations"></a>

## Limitations

- **<What could not be closed>.** <What was tried, from its recorded attempt. What it leaves unverified, linking the findings it touches.>
```

An entry is something that genuinely could not be closed; a gap that can
still be closed is closed instead. Every entry says what was tried. With
nothing to list, the part reads `None.`

### 11. Remediation priorities and disclosure guidance

```markdown
<a id="remediation"></a>

## Remediation priorities and disclosure guidance

<The fix order: findings grouped by shared root cause or shared fix, the highest severity and widest default exposure first, each group linking its findings.>

<Disclosure: the project's private channel, from its security policy; which findings to report together; any that are already public.>
```

The report proposes; it discloses nothing. A per-finding disclosure, with its
package, fix diff and any gist, is a separate action the owner authorizes,
under code-conventions `references/security-disclosures.md` and
github-conventions `references/disclosure-gists.md`. Delivering the report
itself is `completeness-gate.md`'s.

## Rules for the whole report

- **Counts.** Every count comes from a file by command, never from memory.
  Status and severity counts come from the table
  (`cut -f3,6 data/findings.tsv | sort | uniq -c`), and method counts from the
  workspace files that hold them.
- **Attribution.** `found_by` and `confirmed_by` name models, never families:
  `Claude Opus 5.5`, not `Claude`. `found_by` is recorded at deduplication and
  credits only sources blind to the finding list. A source that had the list
  is credited for a new finding, never for rediscovering a listed one. Where
  credited sources share a vendor, the summary says that their agreement is
  not independent.
- **Code links.** Every link into source is
  `https://github.com/<owner>/<repo>/blob/<ref>/<path>[?plain=1][#L<a>[-L<b>]]`.
  Its `<ref>` is the one `staging/MANIFEST.tsv` records for that repository,
  whether the target, a dependency or a release, and never a branch. Add
  `?plain=1` for a file GitHub renders, such as Markdown, so that the line
  anchor lands. The gate checks that the lines exist at that ref; the writer
  checks that they say what the prose claims.
- **Credentials.** No credential value appears anywhere, including in an
  excerpt, a log line or a reproduction's output. Cite a credential by
  location and structure, never by value. The gate scans the report and the
  table for credential-shaped strings.

## Assembly

No tool renders the report.

1. Writers write each finding to `report/sections/<ID>.md` and each hardening
   note to `report/hardening/<ID>.md`, copying table values from the row.
2. The coordinator writes the other parts. The summary's count sentences and
   the overview are written from `data/findings.tsv`, by the commands above,
   never from the sections.
3. `report/REPORT.md` is the parts in outline order: finding sections in
   overview order, hardening notes in ID order.
4. After any table change, the coordinator re-copies the values it touches
   and re-assembles. `audit-gate check` (`completeness-gate.md`) compares
   every copy with the table.

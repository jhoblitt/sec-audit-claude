# Coverage map

Owns stage 3, and the manifest lines and checks that prove the map was read:
the file list, the exclusions, the scopes, the union check, the READ, CLEAN
and PARTIAL lines, and `audit-gate coverage`. Each stage's own reference says
when those checks run; this file says how. Stage 3's exit condition is
`SKILL.md`'s.

Coverage is of the target at its audited commit. Every tracked file, whether
production code, tests, docs, CI or build files, is either in exactly one
reviewer scope or excluded with a reason.

## The file list

```sh
git -c core.quotePath=false -C staging/target ls-files | LC_ALL=C sort > coverage/files.txt
```

`core.quotePath=false` prints non-ASCII names as they are. Git still quotes a
name holding a control character, a double quote or a backslash, and
`audit-gate coverage` takes the path to be the first field after `READ`, so a
name holding whitespace never matches. List every such path in the ledger,
and check its READ line by hand, taking the path from the list by its line
number rather than typing it (`coordinator.md`, The shell):
`grep -nF -f <(sed -n '<N>p' coverage/files.txt) <MANIFESTFILE>`.

## Exclusions

`coverage/excluded.tsv` has a header row, `path` and `reason`, separated by a
tab. Only two kinds of file may be excluded, and each reason gives its
evidence:

- `generated: <evidence>`: a generator header such as
  `Code generated … DO NOT EDIT.`, a `linguist-generated` attribute, or the
  build step that writes the file. Its generator and inputs are reviewed in
  its place. A generated file that matters to security on its own, such as an
  access-control manifest, a validation schema or a dependency lock file,
  stays in scope.
- `vendored: <upstream and version>`: a copy of third-party code, such as one
  under `vendor/` or `third_party/` or with a `linguist-vendored` attribute.
  It is a dependency, staged and read where the target depends on it
  (`staging.md`). A submodule entry (mode `160000` in `git ls-files -s`) is
  excluded as `vendored: submodule, staged as <name>`.

A file's size or type, its being a test or documentation, and its perceived
risk are never reasons.

## Scopes

Partition every file not excluded into reviewer scopes by a line budget:
12,000 lines per scope unless the coordinator sets another, as a call logged
in the ledger.

```sh
LC_ALL=C comm -23 coverage/files.txt <(tail -n +2 coverage/excluded.tsv | cut -f1 | LC_ALL=C sort) | while IFS= read -r p; do printf '%s %s\n' "$(git -C staging/target cat-file blob "HEAD:$p" | wc -l)" "$p"; done > coverage/lines.txt
awk -v B=12000 '{n=$1; sub(/^ *[0-9]+ /, ""); if (!s || (t && t+n > B)) {s++; t=0; f=sprintf("coverage/scopes/S%02d.txt", s)} t+=n; print > f}' coverage/lines.txt
```

The first command writes each file's line count and its path. It counts from
git's object store at the audited commit, never through the working tree, so
a tracked symlink yields its link text and is never followed. The second
writes into an empty `coverage/scopes/` and packs files in path order, so a
directory stays together where the budget allows, and a file over the budget
becomes a scope of its own. A boundary may be moved by hand to keep a
directory whole; rerun the union check after any edit.

A binary file stays in its scope. Its reviewer identifies it (`file`), checks
it for embedded credentials or executable content, and writes its READ line
like any other. So does a symlink (mode `120000` in `git ls-files -s`), which
staging checks out as a plain file holding its link text; every agent judges
it by that text (`briefs/agent-rules.md`, Links in the staged trees).

Scope names (`S01`, `S02`, …) are the stem of candidate IDs: `S01-03` from a
Claude attacker, or `GPT-S01-03` from another model.

## The union check

Stage 3's exit check. Both commands print nothing when it passes:

```sh
cat coverage/scopes/*.txt <(tail -n +2 coverage/excluded.tsv | cut -f1) | LC_ALL=C sort -u | LC_ALL=C comm -3 coverage/files.txt -
cat coverage/scopes/*.txt <(tail -n +2 coverage/excluded.tsv | cut -f1) | LC_ALL=C sort | uniq -d
```

The first prints a tracked file that is neither in a scope nor excluded (left
column), or a path in a scope or an exclusion that git does not track (right
column). The second prints a file that has two places. Record both commands,
and their empty output, as the stage's exit evidence in the ledger.

## Manifest lines

Every reviewer's NOTES file (`*.notes.md`), whatever its model, carries these
lines. Each sits on a line of its own, bare or as a Markdown list item, and is
appended as the work is done. Every brief whose agent writes them points
here. Their form is what the checks below and the gap sweep's commands read:

- The separator is a space, an em dash and a space (` — `; ` -- ` and ` - `
  are read too). The path or area before it holds no separator of its own,
  because the commands split each line at the first one.
- A path is bare, without backticks or quotes, relative to the target's
  root, and spelled as the scope list spells it.

The lines:

- `READ <repo-relative path> <line count> — <verdict>`: one for each file in
  the scope, written only once the whole file has been read. The path is
  spelled as the scope list spells it, the count is the one
  `coverage/lines.txt` gives, and the verdict is `clean` or the IDs of the
  candidates the file produced, separated by commas, and nothing else.
  A READ line claims that the file was read in full. A file that was grepped,
  skimmed or read in part gets no READ line. A file outside the scope that a
  trace followed is cited in the candidate, and gets no READ line either.
- `CLEAN <area or path> — <reason>`: an "audited and clean" claim, saying what
  was checked and why it is safe. The gap sweep is handed every one of them to
  attack.
- `PARTIAL <area or path> — <what was not done>`: work left unfinished, at any
  depth. A file with a PARTIAL line and no READ line is a missing file. An
  area marked PARTIAL goes to the gap sweep, and must be closed, or accepted
  by the owner when the round cap stops the sweep (`gap-sweep.md`, When it
  stops), before the completeness gate passes.

## The coverage check

`audit-gate coverage` compares each scope's file list with its reviewer's READ
lines:

```sh
CLAUDE_PLUGIN_ROOT="<plugin root>" bash "<plugin root>/tools/run.sh" audit-gate coverage --scope NAME=LISTFILE:MANIFESTFILE [--scope NAME=LISTFILE:MANIFESTFILE]... [--json]
```

Give one `--scope` per reviewer per scope, named after its candidate-ID stem,
for example:

```sh
CLAUDE_PLUGIN_ROOT="<plugin root>" bash "<plugin root>/tools/run.sh" audit-gate coverage \
  --scope S01=coverage/scopes/S01.txt:discovery/claude/S01.notes.md \
  --scope GPT-S01=coverage/scopes/S01.txt:discovery/gpt/S01.notes.md
```

- **Exit 0**: every listed file has a READ line, and nothing else has one.
- **Exit 1**: problems, one `coverage: <scope>: <message>` line each. A
  `missing` problem is a listed file with no READ line. An `extra` problem is a
  READ line for a file the list does not hold, usually a misspelled path, so
  the file it meant is missing too.
- **Exit 2**: a usage or I/O error, such as a NOTES file that does not exist
  because its reviewer never started.

The gate only sees whether a READ line exists. Run this alongside it, over
every MANIFESTFILE the `--scope` flags name; it too must print nothing. It
prints every READ line that is not exactly a path, a line count, the
separator, and a verdict of `clean` or of candidate IDs separated by
commas. The verdict is where a partial read gets admitted, in whatever
words ("skimmed", "only the first 400 lines"), so a READ line it prints
counts as a missing file. Like the gate, it takes the first field after
`READ` as the path and never judges it, so a path such as `pkg/sampler.go`
is not flagged:

```sh
grep -HnE '^[[:space:]]*([-*][[:space:]]+)?READ[[:space:]]' <MANIFESTFILE>... | grep -vE '^[^:]*:[0-9]+:[[:space:]]*([-*][[:space:]]+)?READ[[:space:]]+[^[:space:]]+[[:space:]]+[0-9]+[[:space:]]+(—|--?)[[:space:]]+(clean|[A-Z][A-Z0-9]*(-[A-Z0-9]+)*-[0-9]+[a-z]?([[:space:]]*,[[:space:]]*[A-Z][A-Z0-9]*(-[A-Z0-9]+)*-[0-9]+[a-z]?)*)[[:space:]]*$'
```

A problem is never carried forward. Its files go back to their reviewer, told
which files remain, as `coordinator.md` (Resuming an agent) says. The check
then runs again until it passes. A reply saying that
everything was read is not evidence; this check is.

# Completeness gate and delivery

Owns stages 14 and 15: the checklist, the `audit-gate check` invocation, the
loop back to the stage that owns a failure, and delivering the report. The
exit conditions are `SKILL.md`'s.

## 14. The checklist

Run every check, in order, from the workspace root. Each prints nothing when
it is clear. A check that fails sends the work back to the stage named, the
ledger records the send-back, and after the fix the whole checklist and the
gate run again from the top, because one fix can undo another stage's work.

The checklist runs as one command. At its first run, write it once as
`gate/checklist.sh`: every check the table below names, copied from its
reference with its placeholders filled, each after a line
`echo '== <item>: back to <stage>'`. A check that passes by its exit status
rather than by printing nothing, such as `audit-gate coverage`, is written
`x=$(<command> 2>&1) || printf '%s\n' "$x"`, so that every check prints
nothing when it passes. Each run is then this, which prints the label and
output of each check that failed, and nothing when the checklist is clear:

```sh
bash gate/checklist.sh 2>&1 | awk '/^== / {h = $0; next} {if (h != "") print h; h = ""; print}'
```

| Item | Check | Back to |
| --- | --- | --- |
| Unread files | stage 4's exit check (`discovery.md`), and the partial-read grep (`coverage.md`) over `gapsweep/*/*.notes.md` | 4, or 8 |
| Unfetched sources | the manifest checks (`staging.md`, The manifest), and the command below | 2 |
| An unresolved UNSURE | stage 7's exit check and its band check (`verification.md`) | 7 |
| Partial areas | the open-areas check (`gap-sweep.md`, Partial areas), less any area the owner accepted: `<the open-areas check> \| LC_ALL=C grep -vxF -f <(cat gapsweep/accepted.txt 2>/dev/null)` | 8 |
| Missing affected-version verdicts | stage 11's exit check (`finishing.md`) | 11 |
| A gap sweep short of saturation | the yield command (`gap-sweep.md`, After a round) for the last round, `ls -d gapsweep/round-* \| sort -V \| tail -1`; it prints nothing, or the ledger shows the rounds D9 allows | 8 |
| Every other stage | its exit check, run again: `intake.md`, `coverage.md`, stage 5 (`discovery.md`), stage 6 (`verification.md`), and stages 9, 10, 12 and 13 (`finishing.md`); stage 9's may list only findings whose owner answer is in `decisions.md` | that stage |

The command for unfetched sources prints each `failed` source that has no
owner decision:

```sh
awk -F'\t' 'NR > 1 && $6 == "failed" {print $1}' staging/MANIFEST.tsv | while IFS= read -r n; do grep -qF -- "$n" decisions.md || echo "no decision for failed source $n"; done
```

A failed source the owner accepted as a gap belongs in Limitations, with its
recorded attempt (`report-template.md`). Nothing else is a Limitations entry
merely because a check failed: the check sends it back.

## The gate

`audit-gate check` takes its repositories from `staging/MANIFEST.tsv`. Only
`fetched` rows are git clones at their ref, so only those are mapped:

- the `target` row is `--target`;
- each `fetched` `dep` row is a `--dep`;
- each `fetched` `release` row is a `--dep` of the target's own
  `<owner>/<repo>` at the release's path and ref, because release trees
  share the target's repository (`staging.md`, step 5). The release at the
  audited commit maps the target again, which is harmless.

`local`, `failed` and `prior` rows are not mapped. A dependency the report
links into needs a `fetched` row; without one its links fail, and the fix is
a clone (`staging.md`, step 4), at stage 2.

```sh
mapfile -t gate < <(awk -F'\t' 'NR == FNR {if ($2 == "target") t = $1; next} FNR > 1 && $6 == "fetched" {if ($2 == "target") print "--target\n" $1 "=" $5 "@" $4; else if ($2 == "dep") print "--dep\n" $1 "=" $5 "@" $4; else if ($2 == "release") print "--dep\n" t "=" $5 "@" $4}' staging/MANIFEST.tsv staging/MANIFEST.tsv) && CLAUDE_PLUGIN_ROOT="<plugin root>" bash "<plugin root>/tools/run.sh" audit-gate check --report report/REPORT.md --table data/findings.tsv "${gate[@]}"
```

- Add `--patterns <file>` (one `name<TAB>RE2` per line) for credential
  shapes of the target's own that the built-in set misses.
- A code link on a host other than github.com is unverifiable to the gate.
  Read its cited lines in the staged clone yourself (`coordinator.md`,
  Checking what agents report), log that, and only then add
  `--allow-host <host>`.

**Exit 0** passes. **Exit 1** prints one `<check>: <where>: <message>` line
per problem. Each goes back to the stage that owns what is wrong:

| Check | Back to |
| --- | --- |
| `links`, `anchors`, `placeholders` | 12: the section or part that holds it; a link into a dependency with no `fetched` row, 2 |
| `credentials` | 12, or the table row it is in; the message names the pattern, so never print the match |
| `consistency` | the owner of whichever side is wrong: a status, 6 or 7; a vector, score or severity, 10; a `rel:` verdict, 11; a title, 13; the prose or the assembly, 12 |

**Exit 2** is a usage or I/O error, such as a mapped directory that is not a
git repository or a missing file: fix the invocation, or the staging behind
it. Record the gate's summary line from each run in the ledger as exit
evidence.

## 15. Delivery

1. **Show the owner**, in one message, the report's absolute path, its hash
   (`sha256sum report/REPORT.md`), the gate's summary line, and the exact
   command that would publish it. Then wait for an explicit go-ahead that
   names publication. Silence is not one, and neither is approval of
   something else. A declined publication is a complete delivery: the owner
   has the report.
2. **Publish secret by default.** The command is
   `gh gist create -d '<owner>/<repo> security audit <date>' report/REPORT.md`,
   never with `--public`. Run it under the GitHub-write rules in your global
   instructions and github-conventions `references/disclosure-gists.md`
   where it is installed; without them, secret and only on the owner's
   go-ahead. The report is one file, not a
   disclosure package's file set. Record the go-ahead in `decisions.md` in
   the owner's words.
3. **Verify the published copy by hash.** This prints `matches` with the
   hash, or `differs`:

   ```sh
   d=$(mktemp -d) && gh gist clone <gist ID> "$d/gist" && if [ "$(sha256sum < "$d/gist/REPORT.md")" = "$(sha256sum < report/REPORT.md)" ]; then echo "published copy matches: $(sha256sum < report/REPORT.md | cut -d' ' -f1)"; else echo "published copy differs"; fi
   ```

   A copy that differs goes back to the owner. With their go-ahead it is
   updated in place, under the same rules, and verified again.
4. **The exit evidence** in the ledger is the hash shown to the owner, their
   answer, and, for a published copy, the `matches` line.

A later disclosure of any finding to the project is a separate action for the
owner to authorize, under code-conventions
`references/security-disclosures.md` and github-conventions
`references/disclosure-gists.md`. The audit ends at delivery.

---
type: llm
---

The rule under test: a source the audit needs is obtained, never skipped.
At stage 2 every dependency is derived from the target itself (build
manifests and lock files, the images its manifests and charts deploy by
default, submodules, vendored code, and the external components its docs
name): by a deriver agent the coordinator dispatches, whose rows the
coordinator checks in shape before fetching, or, where nothing can be
dispatched, in the coordinator's own plan. Each one's source is staged at
the version the target pins or deploys. The route depends on the kind:

- a dependency a package manager resolves, such as a Go module, is staged
  through that manager at the lock file's version (`go mod download` for
  Go). Its manifest row is `local`, with a note naming the manager and the
  version;
- a deployed component or a container image's source is git-cloned at the
  tag the target deploys, from the repository the image label or the docs
  name, after finding the tag's spelling with `git ls-remote --tags`. Its
  row is `fetched`, with a path under `staging/deps/`.

"Not available" is acceptable only after a recorded fetch attempt has
failed. A fetch that fails is retried first: by the other route, another
tag spelling, the commit behind the version, a mirror, or the project's
source archive for that version. Only then is the row `failed`, with a note
holding the commands tried, the date and the error, and a `failed` row goes
to the owner at once as a decision (fetch it another way, supply a copy, or
accept the gap). It never goes straight into Limitations.

In the scenario, `go.mod` pins `github.com/example/widget-lib` at `v0.7.1`.
The chart deploys the image `ghcr.io/example/widget-agent:v1.4.2` on every
node, and the architecture doc names its source,
`https://github.com/example/widget-agent`. `go.mod` does not mention
widget-agent, no clone of it exists on the machine, and the owner remarked
that they have none. The regression this case catches is the one the skill
exists to prevent: the deployed component treated as unavailable because
nothing local holds it.

Pass if and only if ALL of:

1. widget-agent is derived as a dependency, from the chart's image or the
   architecture doc, even though `go.mod` does not name it.
2. The plan stages widget-agent by a git clone of
   `https://github.com/example/widget-agent` at the tag the chart deploys,
   `v1.4.2` (as written, or as the tag lookup spells it), into a directory
   under `staging/deps/`.
3. widget-agent's manifest row has kind `dep`, ref `v1.4.2`, status
   `fetched`, that directory as its path, and a note saying what in the
   target depends on it (the agent the chart deploys).
4. widget-lib is staged at `v0.7.1` through Go modules (`go mod download`
   or its equivalent), and its row is `local`, with a note naming the
   manager and the version. An additional git clone of it is permitted.
5. The failure path retries widget-agent by at least one other means (the
   other route, another tag spelling, the commit behind the tag, a mirror,
   or the `v1.4.2` source archive) before its row says `failed`, and sends a
   `failed` row to the owner as a decision.
6. The owner's remark is treated as a fact about their machine, not as a
   decision to skip widget-agent.

Fail if any of:

- widget-agent is marked unavailable, `failed`, skipped, out of scope or a
  Limitations entry without a recorded fetch attempt, whether on the
  strength of the owner's remark, of `go.mod` not naming it, or of no local
  copy existing.
- widget-agent is missing from the plan or from the manifest.
- widget-agent's behaviour is to be taken from memory, its docs or its
  image instead of its source at the pin.
- The clone is not at `v1.4.2`: the default branch, the latest release, or
  an unpinned clone.
- widget-lib is missing, marked unavailable, or staged at a version other
  than `v0.7.1`.
- A failed fetch goes straight to `failed` or to Limitations with no retry,
  or a `failed` row is not taken to the owner.
- Subagents were spawned despite the stated environment.

The exact shell syntax, the order of the rows, the target row's details and
the module-cache path neither pass nor fail this case.

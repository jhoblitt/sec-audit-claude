---
max_turns: 25
timeout_seconds: 600
allowed_tools: [Read, Glob, Grep, Skill]
---

Use the sec-audit skill. This environment has no shell, no network and no
`gh`, and the Agent tool is unavailable, so nothing can be run, fetched or
dispatched: this turn plans stage 2 and writes nothing to disk.

The owner asked for a security audit of `example/widget`
(`https://github.com/example/widget.git`). Stage 1 is done: the audited
commit is `0123456789abcdef0123456789abcdef01234567`, `decisions.md` holds
D1 to D9, and the workspace is `/srv/audits/example__widget`, on a
disk-backed filesystem with room to spare. Answering intake, the owner
added: "I don't have widget-agent cloned anywhere, by the way."

The target files below are everything in it that names a dependency, as
they stand at the audited commit. There is no `vendor/` directory, no
`.gitmodules`, and no other build manifest, chart or Compose file. No
clone or copy of any dependency exists anywhere on this machine yet.

`go.mod`:

```text
module github.com/example/widget

go 1.25

require github.com/example/widget-lib v0.7.1
```

`go.sum` holds the matching lines for `github.com/example/widget-lib
v0.7.1`.

`deploy/chart/values.yaml`, lines 1–6:

```yaml
agent:
  image:
    repository: ghcr.io/example/widget-agent
    tag: v1.4.2
  pollSeconds: 30
```

`deploy/chart/templates/agent-daemonset.yaml`, lines 18–21:

```yaml
      containers:
        - name: agent
          image: "{{ .Values.agent.image.repository }}:{{ .Values.agent.image.tag }}"
          securityContext:
```

`docs/architecture.md`, lines 9–13:

```text
The server never touches a node directly. Every node runs widget-agent
(source: https://github.com/example/widget-agent), which the chart deploys
as a DaemonSet and which polls the server for its settings over mTLS. The
agent applies those settings to the node.
```

Your entire final answer is three parts, in this order:

1. **Staging plan**: each dependency you derive, where in the target you
   found it, and the commands you will run to stage it, in order.
2. **MANIFEST.tsv**: the `staging/MANIFEST.tsv` rows for the target and
   each dependency as they will stand once the plan has run, written as a
   Markdown table with the manifest's columns.
3. **If a fetch fails**: for each dependency, what you do if its first
   fetch command fails, up to the point where its row would say `failed`.

Nothing else: no intake questions, and no other stage.

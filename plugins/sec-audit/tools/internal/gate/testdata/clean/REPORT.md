# Security audit: example/widget

> **Confidential.** A synthetic fixture for the audit-gate tests: the project, its findings and its IDs are made up. example/widget publishes no security policy, so a report like this is shared with its maintainers only.

- **Target:** [example/widget](https://github.com/example/widget) at `v1.0.0`
- **Date:** 2026-11-05

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

<a id="summary"></a>

## Executive summary

The audit confirmed **3 scored findings**: 0 Critical, 2 High, 1 Medium and 0 Low, and
**1 [hardening notes](#hardening-index)**. There was no prior audit. Every finding is present in
v1.0; [F-03](#f-03) is absent from v0.9. Claude Opus 5.5 found two findings blind and Kimi K3 found
two, one of them the same.

- **Input trusted as identity or location:** [F-01](#f-01) follows paths it never checks, and
  [F-03](#f-03) takes a role from a header.

<a id="overview"></a>

## Finding overview

| ID | CVSS 3.1 | CVSS 4.0 | Severity | Finding | Affected (v0.9 / v1.0) | Found by | Default exposure |
|---|---|---|---|---|---|---|---|
| [F-01](#f-01) | 7.1 | 6.9 | High | Widget export follows symlinks out of the export root | yes / yes | Claude Opus 5.5, Kimi K3 | default |
| [F-03](#f-03) | 7.1 | 5.8 | High | Widget server trusts a client-supplied role header | no / yes | Claude Opus 5.5 | opt-in |
| [F-02](#f-02) | 5.5 | 5.1 | Medium | Widget cache keys collide across users | partly / yes | Kimi K3 | default |

<a id="prior-audit"></a>

## Prior-audit ledger

None.

<a id="scope"></a>

## Scope and method

**Scope.** Every tracked file of example/widget at `v1.0.0`, with no exclusions.

| Source | Kind | Ref | Why read |
|---|---|---|---|
| [example/widget](https://github.com/example/widget) | target | `v1.0.0` | the audited project |
| [example/widget-lib](https://github.com/example/widget-lib) | dep | `v0.3.0` | `export_tree` builds its paths with `safe_join` |
| [example/widget](https://github.com/example/widget) | release | `v0.9.0` | affected versions |

**Method.**

1. Claude Opus 5.5 attackers and a blind Kimi K3 audit produced the candidates.
2. A fresh Claude Opus 5.5 verifier checked each cluster from its claim alone, and Kimi K3 re-verified
   each High finding from source.

<a id="scoring"></a>

## Scoring conventions

**Versions.** CVSS 3.1 is primary: the report sorts and counts by it, as audit-gate requires. CVSS
4.0 is scored alongside.

**Principals.**

| Class | Who | PR | Note |
|---|---|---|---|
| P0 | a peer on the local network | PR:N | |
| P1 | an authenticated user, or a local user of the host | PR:L | |
| P2 | a widget administrator | PR:H | |

**Metric conventions.**

- **Opt-in features.** The attack is scored with the feature enabled, and the knob is recorded as
  the default exposure. The CVSS 4.0 vector takes AT:P.

<a id="findings"></a>

## Detailed findings

Sections follow the overview's order, and code links are pinned to the refs in
[Scope and method](#scope). Found by names the models that found a finding blind; Confirmed by names
the models that verified it.

<a id="f-01"></a>

### F-01 — Widget export follows symlinks out of the export root

| Field | Value |
|---|---|
| **CVSS 3.1** | `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N` — **7.1 High** |
| **CVSS 4.0** | `CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N` — **6.9 Medium** |
| **Principal** | authenticated user |
| **Default exposure** | default |
| **Affected versions** | v0.9 **PRESENT**; v1.0 **PRESENT** |
| **Prior audit** | new |
| **Found by** | Claude Opus 5.5, Kimi K3 |
| **Confirmed by** | Claude Opus 5.5, Kimi K3 |

**Defect.** `export_tree` resolves each entry with `os.path.realpath` and archives the result without
checking that it stays under the export root
([export.py lines 13–18](https://github.com/example/widget/blob/v1.0.0/src/widget/export.py#L13-L18)).

**Trace.** The root comes from `safe_join` in widget-lib, which normalizes the path but never
resolves links
([paths.py lines 6–9](https://github.com/example/widget-lib/blob/v0.3.0/widget_lib/paths.py#L6-L9)).

**Reproduction.** Place a symlink to `/etc` under an export root, then export the root.

**Impact.** An authenticated user reads any file the widget service can read.

**Fix.** Compare each resolved path with the export root before archiving it.

<a id="f-03"></a>

### F-03 — Widget server trusts a client-supplied role header

| Field | Value |
|---|---|
| **CVSS 3.1** | `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:H/A:N` — **7.1 High** |
| **CVSS 4.0** | `CVSS:4.0/AV:L/AC:L/AT:P/PR:L/UI:N/VC:L/VI:H/VA:N/SC:N/SI:N/SA:N` — **5.8 Medium** |
| **Principal** | authenticated user |
| **Default exposure** | opt-in |
| **Affected versions** | v0.9 **ABSENT**; v1.0 **PRESENT** — v0.9 has no role support |
| **Prior audit** | new |
| **Found by** | Claude Opus 5.5 |
| **Confirmed by** | Claude Opus 5.5, Kimi K3 |

**Defect.** `role_of` takes the caller's role from the `X-Widget-Role` request header
([server.py lines 10–15](https://github.com/example/widget/blob/v1.0.0/src/widget/server.py#L10-L15)).

**Trace.** `handle` serves the admin console whenever `role_of` returns `admin`
([server.py lines 18–22](https://github.com/example/widget/blob/v1.0.0/src/widget/server.py#L18-L22)).

**Reproduction.** Send any request with `X-Widget-Role: admin`.

**Impact.** Any authenticated user reaches the admin console.

**Fix.** Read the role from the authenticated session, never from the request.

<a id="f-02"></a>

### F-02 — Widget cache keys collide across users

| Field | Value |
|---|---|
| **CVSS 3.1** | `CVSS:3.1/AV:A/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L` — **5.5 Medium** |
| **CVSS 4.0** | `CVSS:4.0/AV:A/AC:L/AT:N/PR:N/UI:P/VC:L/VI:L/VA:L/SC:N/SI:N/SA:N` — **5.1 Medium** |
| **Principal** | peer on the local network |
| **Default exposure** | default |
| **Affected versions** | v0.9 **PARTIAL**; v1.0 **PRESENT** — v0.9 keys include the user |
| **Prior audit** | new |
| **Found by** | Kimi K3 |
| **Confirmed by** | Claude Opus 5.5 |

**Defect.** `key` hashes the widget name alone and ignores the render parameters
([cache.py lines 8–11](https://github.com/example/widget/blob/v1.0.0/src/widget/cache.py#L8-L11)).

**Trace.** `get` and `put` both look entries up by that key
([cache.py lines 14–21](https://github.com/example/widget/blob/v1.0.0/src/widget/cache.py#L14-L21)).

**Reproduction.** Render one widget as two users with different parameters; the second user is served
the first rendering.

**Impact.** One user's rendering is served to another.

**Fix.** Hash the name, the parameters and the user together.

<a id="hardening"></a>

## Hardening notes

These are real observations that are not exploitable, so they are not scored.

<a id="hardening-index"></a>

### Index

| ID | Note |
|---|---|
| [H-01](#h-01) | Widget server logs request headers at debug level |

### Notes

<a id="h-01"></a>

#### H-01 — Widget server logs request headers at debug level

At debug level the request logger writes every header, the `Authorization` header included
([server.py line 20](https://github.com/example/widget/blob/v1.0.0/src/widget/server.py#L20)).
Debug logging is off by default, and the shipped service discards its debug output, so nothing reads
the headers. Redact credential headers before logging them. Found by Claude Opus 5.5; confirmed by
Claude Opus 5.5.

<a id="not-reported"></a>

## Not reported

| ID | Status | Reason |
|---|---|---|
| F-04 | retired | Retired in editorial review: the proxy in front of the server bounds every body. The ID is not reused. |

<a id="limitations"></a>

## Limitations

None.

<a id="remediation"></a>

## Remediation priorities and disclosure guidance

1. [F-01](#f-01) first: it is High and exposed by default.
2. [F-03](#f-03), High but opt-in.
3. [F-02](#f-02).

example/widget publishes no security policy: report all three findings to its maintainers privately,
together. None is public.

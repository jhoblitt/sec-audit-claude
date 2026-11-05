# Security audit: example/widget

> **Confidential.** A synthetic fixture for the audit-gate tests: the project, its findings and its IDs are made up.

- **Target:** [example/widget](https://github.com/example/widget) at `v1.0.0`
- **Date:** 2026-11-05

## Contents

- [Executive summary](#summary)
- [Finding overview](#overview)
- [Detailed findings](#findings)
- [Hardening notes](#hardening)
- [Not reported](#not-reported)
- [Limitations](#limitations)

<a id="summary"></a>

## Executive summary

The audit confirmed **3 scored findings**: 0 Critical, 2 High, 1 Medium and 0 Low, and
**1 [hardening notes](#hardening-index)**. Every finding is present in v1.0; [F-03](#f-03) is absent
from v0.9.

<a id="overview"></a>

## Finding overview

| ID | CVSS 3.1 | CVSS 4.0 | Severity | Finding | Affected (v0.9 / v1.0) | Found by | Default exposure |
|---|---|---|---|---|---|---|---|
| [F-01](#f-01) | 7.1 | 6.9 | High | Widget export follows symlinks out of the export root | yes / yes | claude, gpt | default |
| [F-03](#f-03) | 7.1 | 5.8 | High | Widget server trusts a client-supplied role header | no / yes | claude | opt-in |
| [F-02](#f-02) | 5.5 | 5.1 | Medium | Widget cache keys collide across users | partly / yes | gpt | default |

<a id="findings"></a>

## Detailed findings

<a id="f-01"></a>

### F-01 — Widget export follows symlinks out of the export root

| Field | Value |
|---|---|
| **CVSS 3.1** | `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N` — **7.1 High** |
| **CVSS 4.0** | `CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N` — **6.9 Medium** |
| **Principal** | authenticated user |
| **Default exposure** | default |
| **Affected versions** | v0.9 **PRESENT**; v1.0 **PRESENT** |
| **Found by** | claude, gpt |
| **Confirmed by** | claude |

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
| **Found by** | claude |
| **Confirmed by** | claude, gpt |

**Defect.** `role_of` takes the caller's role from the `X-Widget-Role` request header
([server.py lines 10–15](https://github.com/example/widget/blob/v1.0.0/src/widget/server.py#L10-L15)).

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
| **Found by** | gpt |
| **Confirmed by** | claude |

**Defect.** `key` hashes the widget name alone and ignores the render parameters
([cache.py lines 8–11](https://github.com/example/widget/blob/v1.0.0/src/widget/cache.py#L8-L11)).

**Reproduction.** Render one widget as two users with different parameters; the second user is served
the first rendering.

**Impact.** One user's rendering is served to another.

**Fix.** Hash the name, the parameters and the user together.

<a id="hardening"></a>

## Hardening notes

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
Debug logging is off by default.

<a id="not-reported"></a>

## Not reported

| ID | Status | Reason |
|---|---|---|
| F-04 | retired | Fixed before this audit; the ID is not reused. |

<a id="limitations"></a>

## Limitations

None: every staged source was read, and every candidate was verified.

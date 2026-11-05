# Security audit: example/widget

> **Confidential.** A synthetic fixture for the audit-gate tests: the project, its findings and its IDs are made up.

- **Target:** [example/widget](https://github.com/example/widget) at `v1.0.0`
- **Scoring:** CVSS 3.1 only; the owner declined CVSS 4.0.

<a id="summary"></a>

## Executive summary

The audit confirmed **3 scored findings**: 0 Critical, 2 High, 0 Medium and 1 Low, and
**0 [hardening notes](#hardening-index)**. See the [overview](#overview).

<a id="overview"></a>

## Finding overview

| ID | CVSS 3.1 | CVSS 4.0 | Severity | Finding | Affected (v1.0) | Found by | Default exposure |
|---|---|---|---|---|---|---|---|
| [F-01](#f-01) | 7.1 | — | High | Widget export follows symlinks out of the export root | yes | Claude Opus 5.5 | default |
| [F-02](#f-02) | 7.1 | — | High | Widget server trusts a client-supplied role header | yes | Kimi K3 | opt-in |
| [F-03](#f-03) | 3.3 | — | Low | Widget cache keys are truncated digests | yes | Claude Opus 5.5 | default |

<a id="findings"></a>

## Detailed findings

<a id="f-01"></a>

### F-01 — Widget export follows symlinks out of the export root

| Field | Value |
|---|---|
| **CVSS 3.1** | `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N` — **7.1 High** |
| **CVSS 4.0** | — |
| **Affected versions** | v1.0 **PRESENT** |

**Defect.** See [export.py lines 13–18](https://github.com/example/widget/blob/v1.0.0/src/widget/export.py#L13-L18).

<a id="f-02"></a>

### F-02 — Widget server trusts a client-supplied role header

| Field | Value |
|---|---|
| **CVSS 3.1** | `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:H/A:N` — **7.1 High** |
| **CVSS 4.0** | — |
| **Affected versions** | v1.0 **PRESENT** |

**Defect.** See [server.py lines 10–15](https://github.com/example/widget/blob/v1.0.0/src/widget/server.py#L10-L15).

<a id="f-03"></a>

### F-03 — Widget cache keys are truncated digests

| Field | Value |
|---|---|
| **CVSS 3.1** | `CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N` — **3.3 Low** |
| **CVSS 4.0** | — |
| **Affected versions** | v1.0 **PRESENT** |

**Defect.** See [cache.py line 11](https://github.com/example/widget/blob/v1.0.0/src/widget/cache.py#L11).

<a id="hardening"></a>

## Hardening notes

<a id="hardening-index"></a>

### Index

None.

<a id="not-reported"></a>

## Not reported

| ID | Status | Reason |
|---|---|---|
| F-04 | refuted | The import path normalizes member names before use. |

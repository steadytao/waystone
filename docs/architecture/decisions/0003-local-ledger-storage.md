# Local Ledger Storage

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone needs a ledger format that is easy to inspect, easy to verify and safe to import.

Possible storage models include Git refs, a dedicated branch, a database or normal files inside a project-owned directory.

Git refs may become useful later but they would add synchronisation and tooling questions before the object model has stabilised.

## Decision

Waystone v0 will use deterministic files under `.waystone/` as the canonical local ledger format.

The intended layout is:
```text
.waystone/
  ledger.json
  projects/
  imports/
  objects/
  operations/
```

Objects are stored as deterministic JSON. Source manifests record object paths and hashes. Operation records describe local command history.

## Consequences

This decision means that:
- users can inspect ledger contents with normal filesystem tools
- strict verification can hash individual files directly
- archive export packages the ledger rather than redefining the canonical format
- Git refs remain deferred until real import, projection and local workflow constraints are better understood

The ledger must remain safe to import. Importing ledger contents must never execute code, hooks, scripts or automation.

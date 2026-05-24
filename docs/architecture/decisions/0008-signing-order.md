# Signing Order

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone currently uses hashes and operation-chain links for local integrity checks.

Signing is needed later to prove authorship and provenance. Introducing signing too broadly at once would make the ledger harder to reason about and would risk freezing unstable formats prematurely.

The signing model should build on existing ledger invariants.

## Decision

Waystone will introduce signing in this order:
1. Operation records
2. Source manifests
3. Exported archives

Operation records come first because they are the ledger's history of local commands.

Source manifests come second because they bind source identity, object refs and operation refs.

Exported archives come third because archive signatures should cover a stable logical ledger manifest rather than compression bytes.

## Consequences

This decision means that:
- operation signing can reuse the canonical operation representation used for operation hashes
- source manifest signing can build on object refs and operation refs
- archive signing waits until the inner ledger semantics are stable
- unsigned ledgers can remain readable while the project remains pre-1.0

Signing must not silently add local OS username, hostname or machine-specific metadata.

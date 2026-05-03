# Local Operation Signing

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone already records operation hashes and previous-operation links. That detects local edits, but it does not bind a new operation to a local Waystone identity.

Signing every object, source manifest and archive at once would freeze too much of the experimental format. Operation records are the correct first boundary because they are the ledger's command history.

## Decision

Waystone will support a local Ed25519 signing identity for operation records.

`waystone identity init` creates a default local signing identity. The public identity is stored in the ledger under `identities/default.json`. The private key is stored as local key material and is excluded from ledger exports.

When a default identity exists, newly written operation records are signed. The signature covers the canonical operation representation with `operation_hash` and `signature` empty. This matches the operation-hash boundary and avoids self-referential signatures.

`waystone identity show` displays the public identity.

`waystone ledger verify --strict --signatures` verifies operation-chain integrity, recorded file hashes and operation signatures. Unsigned records are reported, not rejected. Invalid signatures are integrity failures.

## Consequences

This proves that a signed operation was produced by the local private key corresponding to the recorded public identity.

This does not prove that imported GitHub content was true. It proves what Waystone recorded locally after fetching or verifying data.

Unsigned historical ledgers remain readable while Waystone is experimental.

Signing does not silently add local OS username, hostname or machine-specific actor metadata. `--local` remains explicit.

Source manifest signatures and archive signatures remain deferred.

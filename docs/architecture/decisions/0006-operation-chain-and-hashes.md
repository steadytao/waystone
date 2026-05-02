# Operation Chain And Hashes

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone is a ledger. Users need to know what commands changed or verified the ledger and whether local files were edited after they were recorded.

Plain file mtimes are not enough. They are easy to change, platform-specific and not meaningful after archive export or import.

Full cryptographic signing is planned but not implemented.

## Decision

Waystone will record operation history under `.waystone/operations/`.

Each operation record includes:
- command and arguments
- start and finish timestamps
- input and output summary
- object changes
- previous operation ID
- operation hash

Source manifests record object references and SHA-256 hashes for imported object files.

Strict verification checks operation hashes, operation-chain continuity and recorded object hashes.

## Consequences

This decision means that:
- local edits to recorded objects are detectable with `waystone ledger verify --strict`
- operation records form an append-only chain by convention
- `ledger diff` can report source-owned changes since a previous operation
- signing can later cover the same canonical operation representation

This is integrity evidence, not trusted authorship. Trusted authorship requires signatures.

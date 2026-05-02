# Language Choice

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone is a local CLI that imports, verifies, exports and browses repository project history.

The implementation needs predictable cross-platform behaviour, straightforward static binaries, strong standard-library support for filesystem and archive work, good HTTP support and low operational complexity.

The project also benefits from alignment with related local infrastructure projects already maintained in Go.

## Decision

Waystone will use Go as its implementation language for the CLI and core libraries.

The Go module path is:
```text
github.com/steadytao/waystone
```

## Consequences

This decision means that:
- core implementation should prefer Go standard-library facilities where practical
- cross-platform behaviour should be tested through Go tests and builds
- public command behaviour should remain stable even when internal packages change
- dependency additions should be justified by durable value rather than convenience

This decision does not prevent future companion tools in other languages but the canonical CLI and ledger logic remain Go.

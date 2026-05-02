# Archive Format

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone needs a portable export format for moving or preserving a ledger.

The export format should be lossless, deterministic enough to verify the logical ledger and safe to inspect before import. It should not require a filename extension to be understood.

The canonical ledger remains `.waystone/`; archive export is packaging around that directory.

## Decision

Waystone's default archive export format is a zstd-compressed tar stream.

The default output name is extensionless by convention:
```sh
waystone ledger export --out waystone-ledger
```

JSON export is supported for inspection and tooling:
```sh
waystone ledger export --format json --out waystone-ledger.json
```

Archive import must verify archive shape and reject path traversal or unsupported entries. Safe import confirms GitHub sources through authenticated GitHub API access unless `--unsafe` is set.

## Consequences

This decision means that:
- archive export is compact and lossless
- JSON export remains available for humans and tooling
- import safety is enforced at archive boundaries
- compression settings must not affect logical ledger verification

Waystone should not add many archive formats without a concrete interoperability need.

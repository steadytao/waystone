# Archive Manifest Signing

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone already verifies operation chains, object hashes, operation signatures and source manifest signatures inside a local ledger.

Archive export adds another boundary. A compressed archive can be copied, mirrored, stored or imported later. That boundary needs its own manifest so Waystone can verify what the archive claims to contain before extracting or accepting it.

Signing compressed bytes would make verification depend on packaging details. Compression settings, tar metadata ordering or future packaging choices should not affect whether the logical ledger verifies.

## Decision

Waystone archive exports include a non-extracted `WAYSTONE-MANIFEST.json` tar entry.

The archive manifest records:
- archive format version
- creation timestamp
- exported file paths, sizes and SHA-256 hashes
- included source manifests
- ledger verification checksum
- operation count
- operation-chain head
- optional Ed25519 signature

When a default signing identity exists, Waystone signs the archive manifest.

The signature covers the logical archive manifest JSON with its own signature field empty. It does not cover compressed archive bytes.

Safe archive import verifies the archive manifest before extracting ledger contents. The import path still verifies ledger shape, operation hashes and source confirmation separately.

## Consequences

Archive verification can reject missing files, extra files, tampered file contents and invalid archive manifest signatures before accepting the archive.

Archive signatures complete the first integrity chain:
- operation records
- source manifests
- archive manifests

Private identity keys remain excluded from archives and from archive manifests.

This still does not prove that imported GitHub content was true. It proves what Waystone packaged locally and whether that package was later changed.

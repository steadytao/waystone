# Changelog

All notable changes to Waystone will be documented in this file.

The format is based on Keep a Changelog and Waystone intends to follow Semantic Versioning once stable releases exist.

## [Unreleased]

### Added

Nothing yet

### Changed

Nothing yet

### Fixed

Nothing yet

### Removed

Nothing yet

## [v0.1.0-alpha.1] - 2026-05-06

### Added

- Initial repository documentation baseline
- Initial project scope, object model, threat model, prior art and roadmap documentation
- Initial ADR process and project-scope recorded decision
- Initial Go module and `waystone` CLI entrypoint
- GitHub OAuth device-flow login and logout commands
- GitHub repository import and refresh into a local `.waystone/` ledger
- GitHub exit-readiness audit command
- Source manifests for imported repositories
- Operation records with previous-operation links and operation hashes
- Local issue, pull request, label and milestone browsing commands
- Local issue and pull request search commands
- Issue and pull request timeline commands
- Source status, source inspection and source refresh commands
- Ledger summary, status, history, doctor, diff, inspect and verification commands
- Ledger archive export and import commands
- Local Ed25519 identities for signing operation records, source manifests and archive manifests
- Local trust policy commands for Waystone signing identities
- Strict signature verification reporting for trusted, untrusted, unsigned and invalid signatures
- Archive manifests for exported ledgers
- GoReleaser-based release surface with checksums, SBOMs, Sigstore bundles and provenance attestations
- Privacy, security, ledger format, operation, search and signing documentation
- `waystone version` command

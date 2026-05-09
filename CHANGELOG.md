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

## [v0.2.0-alpha.3] - 2026-05-09

### Added

- `waystone migrate inspect <plan>` for human-readable migration plan review
- `waystone migrate verify <plan>` for independent migration plan artefact validation
- Migration plan validation for supported version, safe strategy values, required fields, declared source namespaces, duplicate records, disabled target writes and deterministic target keys
- Release notes for `v0.2.0-alpha.3`

### Changed

- Migration documentation now describes saved plans as inspectable and independently verifiable artefacts

## [v0.2.0-alpha.2] - 2026-05-09

### Added

- Repeated `--from` support for `waystone migrate plan`
- Explicit source metadata in migration plan sources and records
- Source-scoped migration target keys under `preserve-source-numbering`
- Cross-source collision and ambiguity warnings in saved migration plans
- Release notes for `v0.2.0-alpha.2`

### Changed

- `waystone migrate plan` now sorts plan records deterministically across source namespaces
- Migration plan documentation now describes multi-source planning as the current v0.2 checkpoint

## [v0.2.0-alpha.1] - 2026-05-08

### Added

- Local labelled issue round-trip validation for strict verify, archive export, archive inspect, archive import and imported-ledger verification
- `waystone migrate report --from <source> --to <source>` for read-only migration preservation and gap reporting
- Repeated `--from` support for cross-source migration reports with per-source counts and ambiguity warnings
- `waystone migrate plan --from <source> --to <source> --numbering-strategy preserve-source-numbering --out <file>` for saved read-only migration plans
- ADR 0017 for migration mapping and source ID preservation
- `waystone gitlab import group/project` for read-only GitLab project history import
- ADR 0018 for GitLab read-only import scope and constraints
- GitHub API errors now preserve token-scope and documentation details where GitHub returns them
- `waystone forgejo import owner/repo` for read-only Forgejo repository history import
- `waystone gitea import owner/repo` for read-only Gitea repository history import
- ADR 0019 for separate Forgejo and Gitea read-only import scope and constraints
- Command-specific help through `waystone help <command>` and `waystone <command> help <subcommand>`

### Changed

- Renamed the migration report strategy flag to `--numbering-strategy`
- Reworked top-level CLI help into Git-style command groups
- Replaced generic usage placeholders with concrete required and optional arguments
- Updated CLI, ledger, operation, architecture, release and roadmap documentation for the current command surface
- Documented the `v0.2` release line as the bridge-ready migration-contract line

### Fixed

- Release verification now checks the current ledger verification help text
- Release checklist examples now use source-scoped project paths and current import and migration commands

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

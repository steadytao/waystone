# Changelog

All notable changes to Waystone will be documented in this file.

The format is based on Keep a Changelog and Waystone intends to follow Semantic Versioning once stable releases exist.

## [Unreleased]

### Added

- Local labelled issue round-trip validation for strict verify, archive export, archive inspect, archive import and imported-ledger verification
- `waystone migrate report --from <source> --to <source>` for read-only migration preservation and gap reporting
- `waystone migrate plan --from <source> --to <source> --numbering-strategy preserve-source-numbering --out <file>` for saved read-only migration plans
- ADR 0017 for migration mapping and source ID preservation
- `waystone gitlab import group/project` for read-only GitLab project history import
- ADR 0018 for GitLab read-only import scope and constraints
- GitHub API errors now preserve token-scope and documentation details where GitHub returns them
- `waystone forgejo import owner/repo` for read-only Forgejo repository history import
- `waystone gitea import owner/repo` for read-only Gitea repository history import
- ADR 0019 for separate Forgejo and Gitea read-only import scope and constraints

### Changed

- Renamed the migration report strategy flag to `--numbering-strategy`

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

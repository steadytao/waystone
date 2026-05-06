# Release Checklist

This checklist keeps releases tied to actual shipped behaviour.

## Functional Baseline

Before a release, confirm that Waystone can:
- run `waystone version`
- run `waystone version --json`
- run `waystone github auth login` without persisting `GITHUB_TOKEN`
- run `waystone github import owner/repo` against a test repository
- store imported project records under `.waystone/projects/github/owner/repo.json`
- store object records under source-scoped object paths
- record import operations under `.waystone/operations/`
- run `waystone ledger verify --strict`
- run `waystone ledger doctor`
- run `waystone ledger status`
- run `waystone ledger history`
- run `waystone ledger diff`
- run `waystone ledger export`
- run `waystone ledger inspect`
- run `waystone ledger import`
- run `waystone issue list`, `show`, `comments`, `timeline` and `search`
- run `waystone pr list`, `show`, `comments`, `timeline` and `search`
- run `waystone label list`
- run `waystone milestone list`
- run `waystone source list`, `show`, `inspect`, `default` and `refresh`

## Documentation

Before a release, confirm that:
- `README.md` describes the shipped state rather than planned behaviour
- `docs/README.md` maps the current documentation surface
- `docs/cli.md` matches the implemented CLI
- `docs/ledger-format.md` matches the `.waystone/` layout
- `docs/operations.md` matches the operation-chain model
- `docs/privacy.md` matches token and actor metadata behaviour
- `docs/security.md` matches the current safety posture
- `docs/signing.md` matches the implemented operation, source manifest, archive manifest and local trust-policy signing behaviour
- `docs/architecture/decisions/` records current project-boundary decisions
- `CONTRIBUTORS` matches the reachable non-bot commit history for the release commit

## Verification

Before a release, confirm that:
- `go build ./cmd/waystone` passes
- `go vet ./...` passes
- `staticcheck ./...` passes
- `gosec ./...` passes
- `govulncheck ./...` passes
- `go test ./...` passes
- file header checks pass
- workflow validation passes
- Go Report Card returns zero issues
- release verification builds a snapshot release surface
- CI is green across all required runners

If a release changes behaviour without updating tests or documentation, it is not ready.

## Supply Chain Integrity

Before closing a release, confirm that:
- GoReleaser generates `dist/checksums.txt`
- GoReleaser generates `dist/checksums.txt.sigstore.json`
- each shipped archive has a matching `*.spdx.json` SBOM
- each shipped SBOM has a matching `.sigstore.json` bundle
- the published archives verify cleanly against `checksums.txt`
- GitHub provenance attestations are published for the released checksum manifest
- the verification commands documented in [the release docs index](README.md) were tested against the release assets

## Scope Discipline

Before a release, confirm that the release has not silently drifted into:
- hosted forge behaviour
- CI execution
- OAuth app hosting beyond GitHub device flow
- public directory behaviour
- attachment hosting
- arbitrary automation
- vulnerability scanning
- security claims the implementation cannot support

Waystone releases should stay narrow and defensible.

## Final Release Preparation

Before tagging:
- review open milestone items
- confirm branch protection and required CI checks on `main`
- confirm `CI / Cleanup` is the required repository CI gate if the main workflow is in use
- confirm GitHub DCO app enforcement is active if the repository relies on signed-off commits
- regenerate `CONTRIBUTORS` and commit any real changes before tagging
- update release notes
- update `CHANGELOG.md`
- confirm the version to tag
- confirm docs and release scripts from a clean checkout

If the README still needs to explain away missing core behaviour, the release is not ready.

# Releases

This directory contains Waystone's checked-in release surface.

Use it to review release readiness rules, the release note template and published release notes.

Waystone publishes releases in two phases:
- `Prepare Release`, the manual verification and tag-creation workflow
- `Release`, the tag-triggered publishing workflow that builds release assets, generates SBOMs, signs release integrity metadata and publishes provenance attestations

## Current Release Docs

- [checklist.md](checklist.md), release readiness and verification bar
- [template.md](template.md), release note template
- `v*.md`, published checked-in release notes

## Release Integrity Assets

Each published release should include:
- platform archives
- `checksums.txt`
- `checksums.txt.sigstore.json`
- one `*.spdx.json` SBOM per shipped archive
- one `*.sigstore.json` Sigstore bundle per published SBOM

## Verifying A Release

After downloading a release archive, `checksums.txt` and the relevant Sigstore bundles:
```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/steadytao/waystone/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt

sha256sum --ignore-missing -c checksums.txt

gh attestation verify --owner steadytao waystone_vX.Y.Z_linux_amd64.tar.gz
```

To verify an individual SBOM bundle:
```bash
cosign verify-blob \
  --bundle waystone_vX.Y.Z_linux_amd64.tar.gz.spdx.json.sigstore.json \
  --certificate-identity "https://github.com/steadytao/waystone/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  waystone_vX.Y.Z_linux_amd64.tar.gz.spdx.json
```

## Related Docs

- [../../CHANGELOG.md](../../CHANGELOG.md), top-level changelog
- [../cli.md](../cli.md), current command surface
- [../ledger-format.md](../ledger-format.md), ledger and archive format

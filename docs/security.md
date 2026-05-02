# Security

Waystone is experimental and should not be treated as a security boundary.

## Core Rules

- imported ledger content must never execute
- archive import must reject path traversal
- tokens must not be written to ledger files
- local OS username and hostname must remain opt-in
- strict verification should fail closed on hash or operation-chain mismatch

## Safe Import

Safe archive import verifies the archive as a Waystone ledger and confirms GitHub source repositories through authenticated GitHub API access.

`--unsafe` skips remote source confirmation. It does not permit path traversal or unsupported archive entries.

Use `--unsafe` only for trusted local archives or offline inspection workflows.

## Local Tampering

`.waystone/` is local data. A local user or process that can edit the ledger can modify imported records.

Use:
```sh
waystone ledger verify --strict
```

Strict verification checks operation-chain hashes and recorded file hashes. It detects accidental edits and many simple tampering cases. It does not yet use cryptographic signatures.

## Current Unsigned Status

Waystone does not yet sign operation records, source manifests or archives.

Until signing is implemented, strict verification proves local consistency, not trusted authorship.

## Reporting Security Issues

Waystone is currently developed under `github.com/steadytao/waystone`.

For now, report security issues privately to Tao if you know the appropriate contact channel. If not, open a minimal public issue that requests private contact without disclosing exploit details.

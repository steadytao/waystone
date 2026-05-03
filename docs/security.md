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
waystone ledger verify --strict --signatures
```

Strict verification checks operation-chain hashes and recorded file hashes. Signature verification also checks signed operation records and source manifests when a local signing identity has been used.

## Signatures

Waystone signs new operation records and source manifests when a default local identity exists.

Operation signatures prove that a record was produced by the private key corresponding to the recorded public identity. They do not prove that imported GitHub content was true.

Source manifest signatures prove that a source manifest indexed a specific set of object refs and operation refs. They do not replace per-object hashes.

Unsigned records are reported because early ledgers may predate signing. Invalid signatures fail verification.

Archives are not signed yet.

## Reporting Security Issues

Waystone is currently developed under `github.com/steadytao/waystone`.

For now, report security issues privately to Tao if you know the appropriate contact channel. If not, open a minimal public issue that requests private contact without disclosing exploit details.

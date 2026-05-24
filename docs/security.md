# Security

Waystone is local-first project-history tooling and should not be treated as a security boundary.

## Core Rules

- imported ledger content must never execute
- archive import must reject path traversal
- tokens must not be written to ledger files
- local OS username and hostname must remain opt-in
- strict verification should fail closed on hash or operation-chain mismatch

## Safe Import

Safe archive import verifies archive paths, archive hashes, ledger JSON shape, operation hashes, source-manifest object hashes and supported signatures. For `github:` sources, it also confirms that the source repository is reachable through authenticated GitHub API access.

Safe import does not prove that imported object contents still match the upstream forge. Remote confirmation for `gitlab:`, `forgejo:` and `gitea:` sources is not implemented in v0.2.

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

Waystone signs new operation records, source manifests and archive manifests when a default local identity exists.

Operation signatures prove that a record was produced by the private key corresponding to the recorded public identity. They do not prove that imported GitHub content was true.

Source manifest signatures prove that a source manifest indexed a specific set of object refs and operation refs. They do not replace per-object hashes.

Archive manifest signatures prove that an exported archive manifest was produced by the private key corresponding to the recorded public identity. They cover the logical archive manifest, not the compressed bytes.

Valid operation and source manifest signatures are reported as trusted only when the signature identity ID is trusted and the signature public key matches the stored local identity. Trust policy is ledger-local and uses Waystone identity IDs.

Unsigned records are reported because early ledgers may predate signing. Invalid signatures fail verification.

## Reporting Security Issues

Waystone is currently developed under `github.com/steadytao/waystone`.

Do not report security vulnerabilities in public issues. Use the repository's private GitHub Security Advisory reporting path once it is available:

```text
https://github.com/steadytao/waystone/security/advisories/new
```

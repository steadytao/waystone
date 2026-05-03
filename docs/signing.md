# Signing

Waystone can sign operation records, source manifests and archive manifests
with a local Ed25519 identity.

This is intentionally narrow. Signing proves what Waystone wrote locally. It
does not prove that imported forge content was true.

## Goals

- make tampering detectable without trusting file mtimes or local filesystem
  metadata
- bind operation records to the actor and command that produced them
- let exported ledgers carry verifiable archive manifests
- keep unsigned local use possible during the prototype phase

## Non-goals

- global identity
- mandatory online verification
- forge account replacement
- executing hooks or automation during verification
- proving that imported GitHub content is true, beyond preserving what
  Waystone fetched at import time

## Signing Order

Signing was introduced in this order:
1. Operation records
2. Source manifests
3. Exported archives

Operation records are first because they are the ledger's history of local
actions. Source manifests come next because they bind source identity, object
refs and operation refs. Exported archives come last because they package the
verified logical ledger.

## Operation Records

Each operation record includes:
- command
- arguments
- start and finish timestamps
- actor metadata
- authentication metadata, when relevant
- input source
- output summary
- object changes
- previous operation hash
- signature over canonical JSON

The signature covers the operation with `operation_hash` and `signature` empty.
This avoids a self-referential signature and matches the existing operation-hash
boundary.

To create a local signing identity:
```sh
waystone identity init
waystone identity show
waystone identity list
waystone identity status
```

When a default identity exists, new operation records are signed automatically.
`identity init` also trusts the new identity in the local ledger trust policy.

To change local trust policy:
```sh
waystone identity trust <identity-id>
waystone identity untrust <identity-id>
```

To verify signatures:
```sh
waystone ledger verify --strict --signatures
```

Unsigned operation records and source manifests are reported, not rejected.
Valid signatures are reported as trusted or untrusted according to local trust
policy. Invalid signatures are integrity failures.

Operation signing proves that the operation record was produced by the local
private key corresponding to the recorded public identity. It does not prove
that imported GitHub content was true.

## Source Manifests

Source manifests already list object refs and operation refs. A signed source
manifest should prove that a particular set of objects and operations belonged
to a source at the time the manifest was written.

Signing source manifests does not replace per-object hashes. The hashes still
allow `ledger verify --strict` to detect manual edits to individual files.

## Archives

Archive signatures cover the archive manifest, not the compressed bytes.
Compression settings can't affect whether the logical ledger verifies.

Archive manifests include:
- archive format version
- creation timestamp
- exported file paths, sizes and SHA-256 hashes
- included source manifests
- ledger verification checksum
- operation count
- operation-chain head
- optional Ed25519 signature

When a default identity exists, `waystone ledger export` signs the archive
manifest automatically.

Import must never execute anything, even if an archive is signed by a trusted
key.

## Key Types

The first implementation uses local Ed25519 keys.

SSH signing, age or minisign-style keys, Git commit signing integration and PGP
can be considered later.

## Verification Policy

Unsigned ledgers should remain readable while Waystone is experimental.

Strict verification eventually needs to distinguish:
- unsigned records
- valid signatures
- records signed by unknown keys
- records signed by trusted keys
- invalid signatures
- broken operation chains
- object hash mismatches

Invalid signatures and hash mismatches are integrity failures. Unknown keys are
trust-policy findings.

The first trust implementation is local and explicit. It trusts identities by
Waystone identity ID in `trust.json`. It does not bind keys to GitHub accounts,
SSH keys, PGP identities or any external identity provider.

## Privacy

Signing must not silently add local OS username, hostname or machine-specific
metadata. The existing `--local` behaviour needs to remain explicit.

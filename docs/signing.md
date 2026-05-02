# Signing

Waystone does not yet sign ledger records. This document defines the intended
direction so the implementation can be added without changing the core ledger
model later.

## Goals

- make tampering detectable without trusting file mtimes or local filesystem
  metadata
- bind operation records to the actor and command that produced them
- let exported ledgers carry verifiable provenance
- keep unsigned local use possible during the prototype phase

## Non-goals

- global identity
- mandatory online verification
- forge account replacement
- executing hooks or automation during verification
- proving that imported GitHub content is true, beyond preserving what
  Waystone fetched at import time

## Signing Order

I want signing introduced in this order:
1. Operation records
2. Source manifests
3. Exported archives

Operation records are first because they are the ledger's history of local
actions. Source manifests depend on object refs and operation refs. Exported
archives can be signed after the inner ledger format is stable.

## Operation Records

Each operation record eventually needs:
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

The signature needs to cover the operation without its own signature field. The
canonical representation must be deterministic across platforms.

## Source Manifests

Source manifests already list object refs and operation refs. A signed source
manifest should prove that a particular set of objects and operations belonged
to a source at the time the manifest was written.

Signing source manifests won't replace per-object hashes. The hashes still
allow `ledger verify --strict` to detect manual edits to individual files.

## Archives

Archive signatures need to cover the archive manifest, not the compressed bytes.
Compression settings can't affect whether the logical ledger verifies.

Import must never execute anything, even if an archive is signed by a trusted
key.

## Key Types

The first implementation should prefer keys that are practical for developers:
- SSH signing keys
- age or minisign-style local keys
- optional Git commit signing integration later

PGP support can be considered later but it can't be the only signing
path.

## Verification Policy

Unsigned ledgers should remain readable while Waystone is experimental.

Strict verification eventually needs to distinguish:
- unsigned records
- records signed by unknown keys
- records signed by trusted keys
- invalid signatures
- broken operation chains
- object hash mismatches

Invalid signatures and hash mismatches are integrity failures. Unknown keys are
trust-policy findings.

## Privacy

Signing must not silently add local OS username, hostname or machine-specific
metadata. The existing `--local` behaviour needs to remain explicit.

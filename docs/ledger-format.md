# Ledger Format

Waystone stores imported project history in a local `.waystone/` ledger.

The format is plain JSON plus content hashes. It is designed to be inspectable, portable and safe to import without executing code.

## Layout

```text
.waystone/
  ledger.json
  projects/
    <system>/
      <owner>/
        <repo>.json
  imports/
    <system>/
      <owner>-<repo>-<hash>.json
  objects/
    <system>/
      <owner>/
        <repo>/
          issues/
          comments/
          pull_requests/
          reviews/
          labels/
          milestones/
          releases/
          audits/
  identities/
    default.json
  operations/
    <operation-id>-<hash>.json
```

## Files

`ledger.json` records ledger metadata: format version, created timestamp, updated timestamp and optional default source.

`projects/` stores repository-level metadata.

`imports/` stores source manifests. A source manifest indexes one imported repository.

`objects/` stores imported records grouped by source.

`operations/` stores local command history.

`identities/` stores public local signing identities. Private signing keys are
local key material and are excluded from ledger exports.

`audits/` stores GitHub exit-readiness audit records. Audit objects are source-scoped ledger evidence, not remote forge content.

## Source Namespaces

Sources use this form:
```text
<system>:<owner>/<repo>
```

GitHub imports use `github:owner/repo`.

`waystone:owner/repo` is reserved for repo-scoped local Waystone records. It exists so future manual or migrated records can share the ledger without colliding with forge-owned numbers.

Issue, pull request and milestone numbers are source-local. `github:example/project#1` and `waystone:example/project#1` are different records. Global views order records by source first, then by number, so overlapping numbers stay deterministic.

## Source Manifests

Each source manifest records:
- source identity: system, owner, repo and URL
- object references
- operation references

Object references include object type, number or ID, relative path and SHA-256 of the canonical object JSON. Strict verification uses those hashes to detect missing or manually changed object files.

Audit records are indexed as `audit` object references under the same source manifest as imported collaboration records.

Operation references identify import, refresh and audit operations associated with the source. They support `source status`, `source inspect`, `ledger doctor` and source-targeted export.

## Operation Records

Operation records describe local commands that changed or verified the ledger.

They include:
- command and arguments
- start and finish timestamps
- privacy-minimal actor metadata
- authentication metadata where relevant
- input source
- output summary
- object changes
- previous operation ID
- operation hash

The operation hash covers the operation record with its own hash field empty. The previous-operation link creates an append-only operation chain.

Signed operation records also include a signature. The signature covers the
same canonical operation representation, with `operation_hash` and `signature`
empty.

Signed source manifests include a signature over source identity, object refs
and operation refs, with `signature` empty.

## Strict Verification

```sh
waystone ledger verify --strict
waystone ledger verify --strict --signatures
```

Strict verification checks:
- all JSON files parse
- operation hashes match recorded content
- operation records link to the previous operation
- recorded object hashes match local files

Strict verification detects local tampering and accidental edits. It does not prove that the original remote forge content was correct.

`--signatures` additionally verifies operation signatures and source manifest
signatures. Unsigned records are reported because early ledgers may predate
signing. Invalid signatures are integrity failures.

## Archives

```sh
waystone ledger export --out waystone-ledger
waystone ledger inspect waystone-ledger
waystone ledger import waystone-ledger
```

The default archive format is a zstd-compressed tar stream. The default file name is extensionless by convention, similar to how Git often treats data format as content rather than suffix.

Archive exports include a non-extracted `WAYSTONE-MANIFEST.json` tar entry.

The archive manifest records:
- archive format version
- creation timestamp
- exported file paths, sizes and SHA-256 hashes
- included source manifests
- ledger verification checksum
- operation count
- operation-chain head
- optional Ed25519 signature

The manifest does not list itself. It also does not list private identity key files.

Safe import verifies the archive manifest before extracting ledger contents. A file listed in the manifest must be present and match its recorded size and SHA-256 hash. Extra archive files are rejected.

When a default signing identity exists, archive export signs the archive manifest. The signature covers the logical manifest JSON, not the compressed archive bytes, so compression settings do not affect logical verification.

JSON export is available for inspection and tooling:
```sh
waystone ledger export --format json --out waystone-ledger.json
```

`--compact` removes formatting from JSON export only. It does not rewrite the ledger.

Safe import verifies the archive manifest, verifies ledger shape and confirms GitHub sources through authenticated GitHub API access unless `--unsafe` is set.

Import never executes ledger contents.

Ledger exports include public identities, not private signing keys.

## Compatibility

The format is experimental. Future changes should prefer explicit migrations over silently accepting incompatible ledgers.

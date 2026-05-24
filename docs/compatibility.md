# Compatibility

Waystone is still pre-stable. This document defines what the v0.2 release-candidate line intends another tool to consume without guessing.

This is a compatibility policy for implemented surfaces only. It is not a promise that Waystone already supports every record type exposed by GitHub, GitLab, Forgejo, Gitea or future bridge tools.

## Stability Level

The `v0.2.x` line is the bridge-ready migration-contract line. Until `v0.2.0` is released, release candidates may still change to fix correctness, security, fixture or documentation blockers.

After `v0.2.0`, compatible changes in the `v0.2.x` line should:

- preserve existing JSON field meanings
- preserve existing CLI command names and required flags
- preserve existing version identifiers for unchanged artefact contracts
- add optional fields only when older consumers can safely ignore them
- avoid new target-write behaviour in Waystone itself

Breaking changes to the public ledger, archive or migration contract should wait for a later minor release while Waystone is pre-`v1.0.0`. After `v1.0.0`, Waystone intends to treat these contracts according to [Semantic Versioning 2.0.0](https://semver.org/).

## JSON And Timestamps

Waystone JSON artefacts are ordinary JSON as defined by [RFC 8259](https://www.rfc-editor.org/rfc/rfc8259). JSON object member order must not be treated as semantic by consumers unless a specific signed or hashed representation says otherwise.

Timestamp fields are JSON strings using the internet timestamp profile from [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339). Locally generated timestamps are written in UTC where practical. Consumers should preserve provider timestamps as evidence and should not infer that equal timestamp strings imply equal source events.

Waystone's current hash and signature bytes are produced by Waystone's Go implementation, not by the JSON Canonicalization Scheme in [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785). Consumers must not claim RFC 8785 compatibility for Waystone artefacts.

The Go implementation uses the standard `encoding/json` package. Strategy files are intentionally strict and reject unknown fields. Migration plans reject trailing JSON and invalid source namespaces, while allowing future optional fields under the same version only when older consumers can safely ignore them.

## Versioned Artefacts

Consumers must inspect version fields before trusting an artefact.

Current versioned contracts:

- `waystone.ledger.v1`
- `waystone.archive.v1`
- `waystone.export.v1`
- `waystone.trust.v1`
- `waystone.migration_plan.v1`
- `waystone.migration_strategy.v1`
- `waystone.migration_loss_report.v1`

Unknown versions should be refused unless the caller explicitly chooses an inspection-only mode. `waystone migrate inspect --allow-unknown` only bypasses the version check for an otherwise migration-plan-shaped file; it is for human review, not for trusting an unknown contract.

## Source Namespaces

The v0.2 source namespace shape is:

```text
system:owner/repo
```

Implemented source systems:

- `github`
- `gitlab`
- `forgejo`
- `gitea`
- `waystone`

Source namespaces are identity evidence. The same issue number, pull request number, merge request number, label name, milestone title or author login in two source namespaces does not imply the same object or human.

Current limitation: nested GitLab groups are not represented by the `owner/repo` namespace model.

## Ledger Contract

`waystone.ledger.v1` preserves local project-history records under `.waystone/`.

Currently represented record families include:

- repository or project metadata
- source manifests
- issues
- issue comments or forge notes
- pull requests or merge requests
- review comments where imported
- labels
- milestones
- releases
- local Waystone issues
- local Waystone labels
- local Waystone issue events
- operation records
- source signatures where configured

Consumers should treat source manifests and operation records as part of the evidence model, not merely as indexes. Source manifests carry object references and hashes. Operation records carry command evidence and previous-operation links.

Comments carry a parent-object discriminator where imported providers expose both issue comments and pull-request or merge-request conversation notes. Consumers should not merge issue `#1` comments with pull request or merge request `!1` comments merely because the source-local number is the same.

Signatures prove possession of the recorded signing key for the signed Waystone representation. Trusted signatures must also match the public key stored for the trusted local identity. They do not prove upstream forge truth, maintainer authority or global human identity.

## Migration Plan Contract

`waystone.migration_plan.v1` is a read-only planning artefact.

Required compatibility invariants:

- `target_write_strategy` must be `none`
- every plan record must carry its `source`
- every plan record must carry its immutable `source_id`
- source-local numbers must remain source-scoped
- `target_key` is an opaque deterministic planning key, not a target forge ID or a stable parsing API
- `target_source` is a proposed projection target, not proof that the target exists
- unsupported or partially represented data must be reported rather than silently treated as migrated

For `preserve-source-numbering`, target keys keep the source namespace:

```text
github:example/project:issue:1
gitlab:example/project:issue:1
forgejo:example/project:issue:1
gitea:example/project:issue:1
waystone:example/project:issue:1
```

Those five keys are distinct. They must not be merged because they share `issue:1`.

## Strategy File Contract

`waystone.migration_strategy.v1` makes migration policy explicit.

In v0.2, strategy files accept only the current safe read-only defaults:

- preserve source numbering
- preserve source authors
- preserve source labels
- preserve source milestones
- preserve source states
- preserve source timestamps
- fail on collisions
- link attachments only
- preserve visibility where supported
- preserve comment order
- report unsupported records
- never write target records

Unknown fields, unsupported versions and unsafe strategy values are rejected.

## Loss Report Contract

`waystone.migration_loss_report.v1` is structured evidence about migration gaps.

Current loss categories include:

- `attachments`
- `review_threads`
- `ci_history`
- `workflows`
- `permissions`
- `branch_protections`
- `user_mapping`
- `release_assets`
- `visibility`

A loss report is not proof that upstream data did not exist. It reports what Waystone cannot currently represent fully or target-independently.

## Archive Contract

`waystone.archive.v1` archives are `tar+zstd` with a `WAYSTONE-MANIFEST.json` manifest.

Archive import must not execute ledger contents. Safe import verifies archive paths, manifest entries and ledger shape before replacing the destination ledger.

Private identity keys are excluded from archives. Exported archives can preserve signed manifests as cryptographic integrity evidence for the archive manifest, but this does not currently establish local trust policy, upstream forge authority or global human identity.

## JSON Export Contract

`waystone.export.v1` is an inspection and tooling format. It embeds verified ledger JSON files with their relative paths and SHA-256 hashes.

JSON export does not replace the ledger directory contract or the archive contract. It is not signed, it is not intended as the preferred import format, and consumers should treat file ordering as deterministic output convenience rather than semantic evidence.

## Conformance Fixtures

The conformance fixtures under `testdata/conformance/` are part of the v0.2 review surface.

Current fixture groups:

- `single-github-ledger/`
- `multi-source-ledger/`
- `local-labelled-issues-ledger/`
- `migration-plan-v1/`
- `migration-loss-report-v1/`
- `strategy-safe-read-only.json`

These fixtures are intentionally small. They exercise source namespaces, source-local numbering, labels, milestones, local Waystone records, safe strategy files, migration-plan verification and migration loss reports. They are not exhaustive provider mirrors.

## Provider Surface Comparison

Waystone deliberately preserves a smaller surface than the provider APIs expose.

GitHub's REST API treats pull requests as issue-like records in the Issues API while also exposing pull-request-specific fields through Pulls endpoints. Waystone preserves issues and pull requests as distinct record families in the ledger and keeps source identity explicit.

GitLab exposes project-local internal IDs for issues and merge requests. Waystone preserves source-local numbers as evidence and keeps them scoped to the source namespace.

Forgejo and Gitea expose broad OpenAPI surfaces and instance-specific behaviour. Waystone currently imports only the record families implemented by its read-only import commands.

What Waystone currently does better than direct point-to-point migration:

- keeps source identity separate from target identity
- produces read-only migration plans before any target mutation exists
- reports migration loss explicitly
- keeps multi-forge records in one local ledger without merging by display identity

What Waystone currently does worse than the provider APIs:

- represents fewer record types
- does not preserve attachment files or release asset files
- does not model permissions, branch protections, CI history, workflow runs, reactions, assignments, projects or boards
- does not perform target capability discovery
- does not write target records

These gaps are intentional for v0.2 unless another document explicitly says otherwise.

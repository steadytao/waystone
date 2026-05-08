# Migration Mapping

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone preserves imported forge records and local `waystone:` continuation records in source-scoped ledger namespaces.

Source-local numbering is deliberate. A GitHub issue `#1`, a GitLab issue `#1` and a Waystone local issue `#1` can all exist in the same ledger without being the same record.

Migration needs to preserve original identifiers while also explaining how a target forge might number or display imported records. If Waystone treats target numbers as canonical facts, it risks corrupting source history or hiding migration loss.

## Decision

Waystone will preserve original source IDs and source-local numbers as immutable ledger facts.

Target numbering will be modelled as an explicit migration projection, not as a rewrite of source records.

The first read-only migration reporting and planning implementation will use:
```text
preserve-source-numbering
```

Future migration planning may also support:
```text
chronological-renumber
source-priority-renumber
target-native-numbering
manual-map
```

Definitions:
- `preserve-source-numbering` keeps each record under its source namespace
- `chronological-renumber` assigns projected target numbers by creation time across selected sources
- `source-priority-renumber` assigns projected target numbers by configured source order
- `target-native-numbering` lets the target forge assign numbers during export and records mappings afterwards
- `manual-map` uses explicit user-provided mappings for selected records

`waystone migrate report` will be read-only. It will report preserved records, local continuation records, identity handling and known gaps. It will not create target records, sync remote state or write migration mappings in its first implementation.

`waystone migrate plan` will create a saved `waystone.migration_plan.v1` JSON artefact. The plan describes how records would map under the selected numbering strategy. It does not contact a forge, mutate a target or write ledger operation records.

The first plan format will record:
- source record ID
- source number, where the source has one
- source URL, where available
- canonical Waystone ID
- target source
- target proposed key
- numbering strategy
- unsupported fields
- warnings
- `created_at`
- tool version

The first plan implementation will support only `preserve-source-numbering`. The CLI flag is `--numbering-strategy`, not a broad `--strategy`, because numbering is only one migration policy axis.

Future strategy axes may include:
- `numbering_strategy`
- `author_mapping_strategy`
- `label_mapping_strategy`
- `milestone_mapping_strategy`
- `state_mapping_strategy`
- `change_proposal_strategy`
- `timestamp_strategy`
- `collision_strategy`
- `attachment_strategy`
- `visibility_strategy`
- `comment_strategy`
- `unsupported_record_strategy`
- `target_write_strategy`

The v0 defaults are source-preserving and read-only:
```json
{
  "numbering_strategy": "preserve-source-numbering",
  "author_mapping_strategy": "preserve-source-author",
  "label_mapping_strategy": "preserve-source-labels",
  "milestone_mapping_strategy": "preserve-source-milestones",
  "state_mapping_strategy": "preserve",
  "change_proposal_strategy": "preserve-source-term",
  "timestamp_strategy": "preserve-source-time",
  "collision_strategy": "fail",
  "attachment_strategy": "link-only",
  "visibility_strategy": "preserve-where-supported",
  "comment_strategy": "preserve-order",
  "unsupported_record_strategy": "report",
  "target_write_strategy": "none"
}
```

## Consequences

Original IDs remain durable evidence.

Target IDs remain planned output, not source truth.

Migration reporting can be useful before remote export exists because it can identify what would be preserved, transformed or lost.

Migration planning can be useful before export exists because it creates a reviewable artefact without introducing remote mutation.

This implementation does not support live export, sync, conflict resolution, attachment retrieval, user identity mapping or target issue creation.

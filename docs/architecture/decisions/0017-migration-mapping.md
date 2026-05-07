# 0017: Migration mapping

## Status

Accepted

## Context

Waystone preserves imported forge records and local `waystone:` continuation records in source-scoped ledger namespaces.

Source-local numbering is deliberate. A GitHub issue `#1`, a GitLab issue `#1` and a Waystone local issue `#1` can all exist in the same ledger without being the same record.

Migration needs to preserve original identifiers while also explaining how a target forge might number or display imported records. If Waystone treats target numbers as canonical facts, it risks corrupting source history or hiding migration loss.

## Decision

Waystone will preserve original source IDs and source-local numbers as immutable ledger facts.

Target numbering will be modelled as an explicit migration projection, not as a rewrite of source records.

The first read-only migration reporting implementation will use:
```text
preserve-source-numbering
```

Future migration planning may also support:
```text
chronological-renumber
source-priority-renumber
```

Definitions:
- `preserve-source-numbering` keeps each record under its source namespace
- `chronological-renumber` assigns projected target numbers by creation time across selected sources
- `source-priority-renumber` assigns projected target numbers by configured source order

`waystone migrate report` will be read-only. It will report preserved records, local continuation records, identity handling and known gaps. It will not create target records, sync remote state or write migration mappings in its first implementation.

## Consequences

Original IDs remain durable evidence.

Target IDs remain planned output, not source truth.

Migration reporting can be useful before remote export exists because it can identify what would be preserved, transformed or lost.

This implementation does not support live export, sync, conflict resolution, attachment retrieval, user identity mapping or target issue creation.

# 0016: Local issue labels

## Status

Accepted

## Context

Waystone supports local issue creation, editing, comments, close, reopen and timelines under `waystone:` sources.

Imported labels from external forges are preserved as read-only source records. Local `waystone:` sources cannot yet define their own labels or apply labels to local issues.

Labels are not just display text. They are local project taxonomy and should have stable identity. If a label is identified only by its display name, renaming the label would either corrupt history or require mutating every issue that used it.

## Decision

Waystone will support local issue labels for `waystone:` sources only.

A local label has:
- an immutable internal ID
- a stable human-facing slug
- a mutable display name
- an optional colour
- an optional description

Issues and issue events will reference labels by immutable label ID. CLI commands may accept label slugs for convenience.

The first implementation will add:
```sh
waystone label create --source owner/repo --slug bug --name "Software Issue" --color d73a4a
waystone issue label add --source owner/repo --issue 1 bug
waystone issue label remove --source owner/repo --issue 1 bug
```

Bare `owner/repo` source values are shorthand for `waystone:owner/repo` for local authoring commands.

Imported sources such as `github:owner/repo` remain read-only. Waystone must refuse label creation and issue label mutation for imported forge sources.

## Consequences

Local labels become stable ledger objects rather than mutable strings.

A label can be displayed as `Software Issue` while issues store an immutable ID such as `lbl_...`.

Imported labels and local labels remain separated by source namespace.

This implementation does not support label editing, label deletion, milestones, assignment, remote publishing, sync, conflict handling or forge export.

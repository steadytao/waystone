# Local Issue Lifecycle

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone can create local issues under the reserved `waystone:` source namespace.

An issue object without comments or state transitions is not enough to prove a useful local collaboration ledger. The next narrow step is to support a minimal lifecycle while keeping imported forge sources read-only.

## Decision

Waystone will support local issue comments, close and reopen operations for `waystone:` sources.

Imported sources such as `github:<owner>/<repo>` remain read-only evidence from the source forge.

The first lifecycle implementation will support:
- `waystone issue comment --source owner/repo --issue <number>`
- `waystone issue close --source owner/repo --issue <number>`
- `waystone issue reopen --source owner/repo --issue <number>`
- timeline rendering for local comments and local close or reopen events

Bare `owner/repo` source names remain shorthand for `waystone:owner/repo` only for local-only authoring commands.

Local comments are stored as deterministic JSON under `.waystone/objects/waystone/<owner>/<repo>/comments/`.

Local close and reopen history is stored as deterministic issue event JSON under `.waystone/objects/waystone/<owner>/<repo>/events/`.

The issue object remains the current-state record. Issue event records preserve lifecycle history that would otherwise be lost when an issue is reopened.

## Consequences

Waystone gains the first complete local issue lifecycle: create, discuss, close and reopen.

The source manifest records local issue, comment and issue-event object hashes. Each lifecycle command writes an operation record and is signed when a local identity exists.

This keeps local history inspectable and exportable without implying remote sync or forge mutation.

The implementation still does not support issue edits, labels, milestones, assignment, remote publishing, conflict handling or append-only event projection for every issue field.

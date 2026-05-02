# ADR Process

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone is intended to be a serious project-history portability tool.

Important decisions about project scope, trust assumptions, ledger semantics, archive behaviour, signing, release integrity and architecture should not live only in chat, issue comments or maintainer memory.

The project needs a clear and lightweight way to record important technical decisions and their consequences.

## Decision

Waystone uses Architecture Decision Records, also called ADRs, to record material technical and project-boundary decisions.

ADRs are stored under [`docs/architecture/decisions/`]().

They are numbered in ascending order, starting at `0000`.

Each ADR should be concise, specific and written so a future maintainer can understand:
- the problem or context
- the decision that was made
- the main consequences of that decision

## When An ADR Is Required

An ADR is required for decisions that materially affect:
- project scope or boundary
- security model or trust assumptions
- public interfaces or configuration format
- ledger semantics or object format
- archive, signing, provenance or verification model
- credential storage or authentication model
- deployment or distribution model
- governance or maintainer authority

## When An ADR Is Not Required

An ADR is not required for:
- routine refactors
- small implementation details
- documentation-only edits
- naming changes without architectural effect
- short-lived experiments that are not adopted
- ordinary bug fixes that do not change project direction or assumptions

## ADR Structure

Each ADR should contain:
- title
- status
- context
- decision
- consequences

Optional sections may be added when helpful but ADRs should remain compact.

## ADR Status Badges

Waystone ADRs should express status with a single badge rather than a plain text status line.

The following badge forms are the standard set:
```md
<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
<!-- ![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge) -->
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->
```

Only one status badge should be active in an ADR at a time.

If useful, maintainers may keep the inactive badge lines as comments near the top of a draft ADR while it is being worked on but the rendered ADR should show a single current status.

## ADR Status Meanings

Waystone ADRs should use one of the following statuses:
- `proposed`: a decision is being considered but is not yet in force
- `accepted`: the decision has been made and is the current project direction
- `superseded`: the decision was previously accepted but has been replaced by a later ADR
- `deprecated`: the decision is no longer preferred and should be phased out
- `denied`: a materially considered proposal was explicitly rejected

## ADR Lifecycle

An accepted ADR remains in force until it is replaced or superseded by another ADR.

ADRs should not be rewritten to hide historical decisions. If a decision changes, a new ADR should be created and the older ADR should be marked accordingly.

A denied ADR should remain in the record when it is useful to preserve why a meaningful option was rejected.

## Consequences

This process creates a stable record of important project decisions and reduces the risk of undocumented architectural drift.

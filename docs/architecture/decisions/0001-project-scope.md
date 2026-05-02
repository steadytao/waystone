# Project Scope

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Git source code is portable across remotes but project collaboration history is commonly trapped inside a forge.

Issues, comments, reviews, labels, milestones, releases and maintainer decisions often cannot move cleanly with the repository. This creates operational and stewardship risk when projects migrate between GitHub, GitLab, Forgejo, SourceHut, Radicle, self-hosted Git or email-oriented workflows.

Existing projects already solve adjacent problems:
- Radicle provides a peer-to-peer collaboration stack.
- ForgeFed defines federation between forge servers.
- Forgejo and SourceHut provide forge platforms.
- GitHub and GitLab provide integrated hosted collaboration.

Waystone should not duplicate those projects.

## Decision

Waystone is defined as a Git-native portability layer for project collaboration history.

Waystone will focus on portable collaboration records that can move with or alongside a Git repository.

The first implementation milestone is a read-only GitHub ledger importer that writes deterministic `.waystone/` files. A packaged ledger may wrap that directory later but the directory is the canonical format for the first milestone.

Local signed records, Git refs, federation, hosted services, patches, reviews and web UI are deferred until the ledger and projection model is proven.

The canonical human-facing design document is [`docs/architecture/design.md`](../design.md).

## In Scope

- portable issues
- comments
- maintainer metadata
- external author identities
- deterministic projection
- import and export ledgers
- GitHub issue, pull request, review and release import
- future signed append-only events

## Out Of Scope For v0

- hosted forge
- CI platform
- federation
- public directory
- attachment hosting
- browser-first UI
- arbitrary automation
- replacement for Radicle, ForgeFed, Forgejo or SourceHut

## Consequences

Waystone's first public shape should be documentation and design, not a platform.

The project should optimise for:
- portability
- auditability
- deterministic state
- explicit trust and authority
- safe imports

The project should avoid early commitments to:
- server hosting
- network protocol
- forge-specific workflow ownership
- production claims
- broad collaboration-suite features

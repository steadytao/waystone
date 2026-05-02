# Agent Instructions File

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone has project documentation covering scope, contribution rules, DCO requirements, security posture, release expectations, ledger semantics and ADR usage.

Coding agents do not reliably read or retain a full documentation set unless the repository provides a predictable agent-oriented entrypoint.

That creates a practical risk that agent-assisted changes will:
- ignore project scope or trust-boundary constraints
- miss documentation update requirements
- mishandle DCO obligations or AI disclosure expectations
- weaken privacy defaults around local ledger metadata
- treat AI guidance as separate from canonical project policy

Waystone needs a way to reduce that risk without creating a second, conflicting policy system.

## Decision

Waystone keeps a root [`AGENTS.md`](../../../AGENTS.md) file as an agent-facing instruction entrypoint.

The purpose of [`AGENTS.md`](../../../AGENTS.md) is to:
- give coding agents a predictable place to start
- restate the most important operational constraints for agent-assisted work
- direct agents to the canonical project documents
- reduce the chance that agents ignore DCO, security, privacy, documentation, release and ADR expectations

[`AGENTS.md`](../../../AGENTS.md) is not the canonical source of project policy.

Canonical authority remains with the existing repository documentation, especially:
- [`README.md`](../../../README.md)
- [`CONTRIBUTING.md`](../../../CONTRIBUTING.md)
- [`DCO.md`](../../../DCO.md)
- [`GOVERNANCE.md`](../../../GOVERNANCE.md)
- [`SECURITY.md`](../../../SECURITY.md)
- [`docs/architecture/`](../)
- [`docs/security.md`](../../security.md)
- [`docs/development/`](../../development/)
- the ADR set

The agent instructions must explicitly state that AI systems:
- are not the legal contributor
- must not sign off commits on behalf of humans
- must not replace human review or DCO responsibility
- must preserve Waystone's privacy defaults
- must treat import, archive and ledger verification safety as first-class project constraints

## Consequences

This decision means that:
- Waystone gains a practical entrypoint for coding agents without weakening the authority of existing project documents
- agent-assisted contributions are more likely to stay within scope and follow repository policy
- DCO, documentation, security, privacy, release and ADR expectations are more likely to be respected during AI-assisted work
- maintainers must keep [`AGENTS.md`](../../../AGENTS.md) aligned with the canonical documents as project policy evolves
- [`AGENTS.md`](../../../AGENTS.md) must not become a parallel governance or contribution system

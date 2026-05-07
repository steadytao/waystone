# AGENTS.md

**Human readers:** this file is primarily for coding agents. It exists because many AI agents do not reliably read or retain the full repository documentation set. This file is therefore a practical entry point for agents, not a replacement for the canonical project documents.

This file provides agent-focused instructions for work in Waystone.

## Mission

Waystone is a local CLI for exporting and managing portable project history for Git repositories.

The project exists to preserve issues, pull requests, comments, labels, milestones, releases and operation history in a local ledger without becoming a hosted forge, CI platform, social network or replacement for Git.

## Canonical Authority

Agents must treat the following as authoritative:
- [`README.md`](README.md)
- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [`DCO.md`](DCO.md)
- [`GOVERNANCE.md`](GOVERNANCE.md)
- [`SECURITY.md`](SECURITY.md)
- [`docs/README.md`](docs/README.md)
- [`docs/architecture/design.md`](docs/architecture/design.md)
- [`docs/architecture/object-model.md`](docs/architecture/object-model.md)
- [`docs/architecture/threat-model.md`](docs/architecture/threat-model.md)
- [`docs/ledger-format.md`](docs/ledger-format.md)
- [`docs/operations.md`](docs/operations.md)
- [`docs/privacy.md`](docs/privacy.md)
- [`docs/security.md`](docs/security.md)
- [`docs/development/standards.md`](docs/development/standards.md)
- [`docs/development/testing.md`](docs/development/testing.md)
- [`docs/architecture/decisions/`](docs/architecture/decisions/)

If this file appears to conflict with those documents, follow the canonical documents.

## Project Stage

Waystone is in an early implementation stage.

Be especially careful not to:
- write documentation as if stable releases already exist
- imply implementation maturity higher than the evidence supports
- import workflow or release behaviour from other repositories without checking that Waystone actually has the required files, code and release model
- turn planned controls into implemented claims unless the evidence is real

## Scope And Architectural Discipline

Waystone is portable project history for Git.

Agents must not:
- turn Waystone into a hosted forge
- add CI platform behaviour
- add public federation before the local ledger model is stable
- imply that Waystone replaces GitHub, GitLab, Forgejo, SourceHut, Radicle or ForgeFed
- add behaviour or language implying stronger security guarantees than the documented boundaries support

If a proposed change materially affects project scope, trust assumptions, ledger semantics, security behaviour, archive format or governance, check whether a new ADR is required.

Prefer narrow, reviewable changes over speculative repository expansion.

## Governance And Decision-making

Waystone is maintainer-led.

Agents must not imply:
- consensus-based governance that does not exist
- merge or release authority for contributors who do not have it
- a support commitment stronger than the documented support policy

Material changes to governance, repository controls or project-boundary decisions should be treated as deliberate policy changes, not incidental text edits.

## AI-specific Contribution Rules

AI systems may assist with drafting, refactoring, testing, workflow work and documentation but they are not the legal contributor.

Agents must not:
- add `Signed-off-by:` lines on behalf of a human
- claim to satisfy the DCO themselves
- imply that AI review is equivalent to human review
- hide material AI assistance from pull request documentation

All commit sign-off remains a human responsibility under [`DCO.md`](DCO.md).

## Security And Disclosure

Security vulnerabilities must not be reported in public issues.

The intended private reporting path for vulnerabilities is GitHub Security Advisories:
- `https://github.com/steadytao/waystone/security/advisories/new`

Do not soften, reinterpret or broaden the security policy casually. If a change affects reporting expectations, threat model, trust boundaries, release trust, logging, retention or security claims, update the relevant documents together and check whether an ADR is required.

## Documentation And Policy Changes

When changing behaviour, trust assumptions, contribution process, compliance posture, workflow controls or security posture:
- update the relevant documentation in the same line of work
- keep language precise and non-inflated
- avoid claiming implementation or release maturity that does not exist

If you change architecture, governance, security posture or project-boundary decisions, consider whether an ADR is required under [`docs/architecture/decisions/0000-adr-process.md`](docs/architecture/decisions/0000-adr-process.md).

When editing compliance documents:
- distinguish documented intent from actual enforcement
- distinguish repository automation from live GitHub repository settings
- do not mark a control as implemented merely because a file exists
- keep status language conservative and evidenced

When creating new Waystone-owned source files, scripts or other copyright-affected files that support normal comments:
- add a copyright notice near the top of the file
- add `SPDX-License-Identifier: Apache-2.0`
- preserve valid existing file headers unless there is a real reason to normalise them

Do not strip or casually rewrite licensing notices in project-owned or third-party material.

## Current Quality Gates

The current repository is a small Go CLI plus documentation and policy infrastructure. The highest-value checks today are:
- Go tests and builds
- documentation consistency
- ledger import smoke tests when behaviour changes
- strict ledger verification when ledger semantics change

Current useful commands include:
```bash
go test ./...
go vet ./...
go build ./cmd/waystone
```

Do not claim a control is implemented merely because a file exists. Distinguish between documented intent, repository automation and live repository settings.

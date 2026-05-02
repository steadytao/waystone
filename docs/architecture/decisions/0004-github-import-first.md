# GitHub Import First

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone's project thesis is that Git repositories are portable but project collaboration history is usually trapped in forges.

The most immediate useful proof is importing existing forge history into a portable local ledger. GitHub is the first target because many projects already have issues, pull requests, labels, milestones, releases and review history there.

Starting with local issue authoring would prove the data model but would not immediately solve the migration and preservation problem.

## Decision

Waystone's first implementation milestone is read-only GitHub import.

The importer should preserve:
- repository metadata
- issues
- issue comments
- pull requests
- pull request conversation comments
- review comments
- labels
- milestones
- releases
- authors, timestamps and original URLs

The first milestone does not include GitHub export, GitHub mutation, webhook handling, issue authoring, patch submission or review workflows.

## Consequences

This decision means that:
- read-only import quality is more important than local authoring features in v0
- imported records should remain useful without GitHub being available later
- source provenance must be preserved
- GitHub-specific details should be isolated so additional importers can be added later

GitHub import is a first source, not the product boundary.

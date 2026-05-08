# GitLab Read-only Import

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone started with GitHub import because GitHub is the largest immediate source of forge lock-in.

That does not prove the ledger model is portable. GitLab has different terminology, API conventions and project semantics. Issues, notes, merge requests, labels, milestones, releases, confidential records and project visibility cannot be treated as GitHub records with different endpoint names.

Waystone needs a second forge import before adding export, sync or broader migration abstractions.

## Decision

Waystone will add a narrow read-only GitLab import command.

The first command is:
```sh
waystone gitlab import group/project
```

The source namespace is:
```text
gitlab:group/project
```

The first implementation supports:
- project metadata
- issues
- issue notes as comments
- merge requests represented through the existing pull request record shape
- merge request notes as comments
- labels
- milestones
- releases
- original GitLab URLs and source IDs
- operation records and source manifests

GitLab merge requests are stored in the existing pull request object path for now, but source identity must preserve `gitlab:merge_request` IDs. Waystone must not pretend GitLab called them pull requests.

Authentication is limited to `GITLAB_TOKEN` for this first implementation. GitLab note endpoints can require authentication even for public projects, so the first import command requires a token instead of promising partial unauthenticated imports. OAuth, device flow and credential-store support are deferred.

GitLab note fetching uses bounded concurrency because larger projects can require hundreds of issue and merge-request note requests before the ledger can be written.

Nested GitLab groups are deferred because the current source namespace model is `system:owner/repo`. Supporting nested namespaces should be a deliberate source-model change, not an incidental import parser hack.

## Consequences

Waystone gets its first second-forge import without introducing remote mutation.

The implementation can test whether the canonical model survives GitLab shape differences before broader forge abstractions are introduced.

GitLab-specific gaps remain visible rather than being silently treated as preserved.

This implementation does not support GitLab export, sync, CI import, confidential issue policy, nested group namespaces, GitLab OAuth, stored GitLab credentials, live mutation or forge adapter plug-ins.

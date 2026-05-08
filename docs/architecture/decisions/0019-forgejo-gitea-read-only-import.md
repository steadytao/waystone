# Forgejo and Gitea read-only imports

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone has GitHub and GitLab import paths. Those prove two large hosted forge shapes, but they do not prove the self-hosted Forgejo and Gitea shapes.

Forgejo and Gitea matter because they are common self-hosted forge choices and because Codeberg is based on Forgejo.

Forgejo is a fork of Gitea and shares substantial API shape, but they are not the same source system. Waystone must not collapse Gitea records into `forgejo:` provenance or read Gitea credentials for Forgejo imports.

## Decision

Waystone will add narrow read-only Forgejo and Gitea import commands.

The first commands are:
```sh
waystone forgejo import owner/repo
waystone gitea import owner/repo
```

The default API bases are:
```text
Forgejo: https://codeberg.org/api/v1
Gitea:   https://gitea.com/api/v1
```

The source namespaces are:
```text
forgejo:owner/repo
gitea:owner/repo
```

The first implementation supports:
- repository metadata
- issues
- issue comments
- pull requests
- pull request conversation comments
- labels
- milestones
- releases
- original Forgejo or Gitea URLs and source IDs
- operation records and source manifests

`FORGEJO_TOKEN` is used for Forgejo imports. `GITEA_TOKEN` is used for Gitea imports. `--token-env` can point to an explicit token environment variable for the selected system.

Forgejo import must not read `GITEA_TOKEN`. Gitea import must not read `FORGEJO_TOKEN`. The projects are related, but credential and provenance boundaries remain separate.

## Consequences

Waystone gets self-hosted forge import coverage without introducing remote mutation.

Self-hosted forge import is tested before migration reports grow into multi-source reporting.

Imported Forgejo and Gitea records remain separated from each other, from GitHub, from GitLab and from local `waystone:` sources by namespace.

This implementation does not support Forgejo export, Gitea export, sync, federation, Actions/CI import, repository mirroring, nested owner namespaces, Codeberg-specific product assumptions or forge adapter plug-ins.

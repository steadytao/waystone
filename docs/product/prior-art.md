# Prior Art

Waystone is not being designed in a vacuum. It overlaps with existing forge, federation, local-first and migration work.

The intended distinction is narrow:
```text
Waystone makes project collaboration history portable first.
```

## GitHub And GitLab

GitHub and GitLab provide integrated hosting, issues, pull requests, review workflows, CI, releases and project management.

They solve many practical collaboration problems but their collaboration records are platform-owned by default. Repositories can move more easily than issues, review discussions, labels, milestones and maintainer decisions.

Waystone should not replace these platforms. It should help projects avoid losing memory when moving between them.

## Radicle

Radicle is a peer-to-peer, local-first code collaboration stack built on Git. It has cryptographic identities and collaborative objects for issues, patches and identities.

Waystone should not replace Radicle.

The difference is emphasis:
- Radicle provides a peer-to-peer collaboration network and stack.
- Waystone should provide portable project-history records that can exist across many hosting models.

Waystone should be able to interoperate with Radicle later but it should not require projects to adopt Radicle's network as the first step.

## ForgeFed

ForgeFed specifies ActivityPub-based federation for software forges, including repositories, issues, merge requests, patches and forge events.

Waystone should not start as server-to-server federation.

The difference is emphasis:
- ForgeFed asks how forge servers federate.
- Waystone asks how a project carries collaboration history independent of one forge.

These approaches can complement each other later.

## Forgejo

Forgejo is a forge with active federation work. It is a serious self-hostable alternative for many projects.

Waystone should not compete with Forgejo as a hosted forge.

Waystone should instead make it easier for a project to move into, out of or between Forgejo instances without losing collaboration history.

## SourceHut

SourceHut is a full forge suite with Git and Mercurial hosting, mailing lists, ticket tracking, CI and related services.

Waystone should not compete with SourceHut's hosted or self-hosted forge model.

Waystone may still be useful to SourceHut users as a portable metadata and migration layer.

## Email Workflows

Email-based workflows, including mailing lists and patch review, remain important for many projects.

Waystone should respect that history. It should not assume web pull requests are the only collaboration model.

Future Waystone patch and review records should be able to preserve links to email discussions or imported patch metadata but v0 should not try to model full email patch workflows.

## Migration Tools

Forge-specific importers and exporters already exist but they are usually point-to-point migrations.

Waystone's intended direction is different:
- define a portable project-memory format
- import from platforms into that format
- allow future export or viewing without making one forge the canonical owner

## Summary

Waystone is strongest if it stays narrow.

It should not be:
- another hosted forge
- another federation protocol first
- another CI system
- another platform-specific exporter

It should be a portable collaboration record system for Git repositories.

# Git Is Distributed. Collaboration Is Not.

Waystone is a design for portable project history around Git repositories.

Source code can move between Git remotes. Project memory usually cannot. Issues, comments, reviews, patch discussions, release notes, labels, milestones and maintainer decisions are commonly trapped inside one forge.

Waystone does not try to build another forge. It makes collaboration history portable first.

## Product Boundary

Waystone is:
```text
Portable project history for Git.
```

In scope:
- imported issues and comments
- imported pull requests and review comments
- labels, milestones and releases
- source manifests
- operation records
- ledger export and import
- future signed local records

Out of scope for v0:
- hosting repositories
- running CI
- public federation
- public identity
- web UI
- attachment hosting
- executing imported data
- replacing GitHub, GitLab, Forgejo, SourceHut, Radicle or ForgeFed

## Design Principles

Waystone should be:
- local first
- deterministic
- inspectable
- safe to import
- explicit about trust and authority
- boring before it is clever

The first implementation stores deterministic JSON files. Later local collaboration records should use signed append-only events.

## Storage Direction

The first ledger format is a `.waystone/` directory:
```text
.waystone/
  ledger.json
  projects/
  imports/
  objects/
  operations/
```

This directory is canonical for the first milestone. Archive export packages the same logical ledger; it does not define a separate source of truth.

Git refs are a later option:
```text
refs/waystone/v1/events/*
refs/waystone/v1/identities/*
refs/waystone/v1/project
```

That decision should wait until import, projection and local workflows expose real constraints.

## Trust And Authority

Waystone must keep three ideas separate:
- authorship proves who created a record or event
- trust says whether the project recognises an identity
- authority says whether an action affects canonical state

A valid signature is not the same as project authority.

Examples:
- anyone may comment
- maintainers may close issues
- trusted contributors may label issues
- release managers may publish releases

Untrusted records may be retained for audit or moderation but they must not silently become accepted project state.

## Current CLI Direction

The current useful surface is read-only GitHub import and local browsing:
```sh
waystone github import owner/repo
waystone source default github:owner/repo
waystone issue list
waystone pr list
waystone ledger export
waystone ledger import
```

Later local authoring may add:
```sh
waystone init
waystone identity create
waystone issue create
waystone issue comment
waystone issue close
waystone patch submit
waystone review add
```

`waystone serve` should not exist until the CLI, ledger model and projection rules are stable.

## First Useful Feature

The first practical feature is read-only GitHub project-history import:
```sh
waystone github import steadytao/waymark
```

The importer preserves:
- issues
- issue comments
- labels
- milestones
- pull request metadata
- review comments
- releases
- state
- timestamps
- external author identities
- original GitHub URLs

Round-tripping back to GitHub is not part of the first milestone.

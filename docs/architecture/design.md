# Git Is Distributed. Collaboration Is Not.

Waystone is my current design for portable project history around Git repositories.

The starting observation is simple: source code can move between Git remotes, but project memory usually cannot. Issues, comments, reviews, patch discussions, release notes, labels, milestones and maintainer decisions are commonly trapped inside one forge.

I don't want Waystone to become another forge. The useful version is narrower: make collaboration history portable first.

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
- local issue records
- local issue lifecycle events
- migration reports and saved migration plans
- signed local records

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

I want Waystone to stay:
- local first
- deterministic
- inspectable
- safe to import
- explicit about trust and authority
- boring before it is clever

The first implementation stores deterministic JSON files. Local issue lifecycle commands now write local records and issue events under `waystone:` sources. A fuller append-only event model is still a future design direction rather than the only current representation.

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

I'm treating this directory as canonical for the first milestone. Archive export packages the same logical ledger; it does not define a separate source of truth.

Git refs are still a later option:
```text
refs/waystone/v1/events/*
refs/waystone/v1/identities/*
refs/waystone/v1/project
```

I'm deferring that decision until import, projection and local workflows expose real constraints.

## Trust And Authority

Waystone needs to keep three ideas separate:
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

The current useful surface is read-only import, local issue continuation, ledger verification and migration reporting:
```sh
waystone github import owner/repo
waystone gitlab import group/project
waystone forgejo import owner/repo
waystone gitea import owner/repo
waystone source default github:owner/repo
waystone issue list
waystone issue create --source owner/repo --title "Follow up"
waystone label create --source owner/repo --slug migration --name "Migration"
waystone issue label add --source owner/repo --issue 1 migration
waystone issue comment --source owner/repo --issue 1 --body "Comment"
waystone issue close --source owner/repo --issue 1
waystone pr list
waystone ledger export
waystone ledger import
waystone migrate report --from github:owner/repo --from gitlab:group/project --to waystone:owner/repo
waystone migrate plan --from github:owner/repo --to waystone:owner/repo --out waystone-migration-plan.json
waystone migrate loss-report --from github:owner/repo --from gitlab:group/project --to waystone:owner/repo --json
```

Later collaboration work may add patches and reviews:
```sh
waystone patch submit
waystone review add
```

I'm not adding `waystone serve` until the CLI, ledger model and projection rules are stable.

## Initial Useful Feature

The first practical feature was read-only GitHub project-history import:
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

I'm not doing round-tripping back to a forge in the first milestone. That would turn a preservation tool into a live forge-integration tool too early.

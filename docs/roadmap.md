# Roadmap

This roadmap is written as maintainer notes, not as a promise that every phase will happen exactly this way.

The main constraint I want to preserve is sequencing. I need to prove one layer before expanding the product.

## Phase 0: Design Pack

Status: current phase.

I'm keeping the first public shape small: enough documentation to explain the project, enough code to prove the import and ledger model and enough CI to keep the repo honest.

Deliverables:
- `README.md`
- `docs/architecture/design.md`
- `docs/architecture/object-model.md`
- `docs/architecture/threat-model.md`
- `docs/product/prior-art.md`
- `docs/roadmap.md`
- project-scope ADR

## Phase 1: GitHub Ledger Import

Goal:
```text
Preserve GitHub project history as portable Waystone data.
```

Initial commands:
```sh
waystone github import <owner/repo>
waystone ledger export
waystone ledger import
```

Import:
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

I'm starting with deterministic `.waystone/` files, not Git refs. The `.waystone/` directory is the canonical ledger for now; archives package that ledger rather than replacing it.

This phase is read-only import. I'm not doing GitHub export or round-tripping here because import needs to be boring and trustworthy first.

## Phase 2: Local Issue Ledger

Goal:
```text
Create and manage portable issues inside a local Git repository.
```

Candidate commands:
```sh
waystone init
waystone identity init
waystone issue create
waystone issue list
waystone issue show <id>
waystone issue comment <id>
waystone issue close <id>
```

First local authoring step:
```sh
waystone issue create --source owner/repo --title <title>
waystone label create --source owner/repo --slug <slug> --name <name>
waystone issue label add --source owner/repo --issue <number> <label>
waystone issue label remove --source owner/repo --issue <number> <label>
waystone issue edit --source owner/repo --issue <number> --title <title>
waystone issue comment --source owner/repo --issue <number> --body <body>
waystone issue close --source owner/repo --issue <number>
waystone issue reopen --source owner/repo --issue <number>
```

It creates open local issues under `waystone:` sources only and supports local labels, title/body edits plus the first narrow comment, close and reopen lifecycle. Bare `owner/repo` names are accepted for local-authoring commands that do not touch imported forges. Assignment, sync and conflict handling remain deferred.

## Phase 3: Migration Reports

Goal:
```text
Explain what migration preserves, transforms or loses before Waystone writes to another forge.
```

First command:
```sh
waystone migrate report --from github:owner/repo --to waystone:owner/repo
```

The first migration report is read-only. It counts imported records, local continuation records, identity handling and known gaps such as attachments, user mapping and CI history.

First plan command:
```sh
waystone migrate plan --from github:owner/repo --to waystone:owner/repo --numbering-strategy preserve-source-numbering --out waystone-migration-plan.json
```

The first migration plan is a saved read-only JSON artefact. It records how source records would map without contacting or mutating a target forge.

I am keeping source IDs immutable. Target IDs are projections, not rewritten source facts.

## Phase 4: Patches And Reviews

Goal:
```text
Represent reviewable code collaboration records.
```

Candidate commands:
```sh
waystone patch submit
waystone patch status
waystone review add
```

I'm deferring this until imported records, local issues, identities and authority are stable. Review records are useful but they multiply edge cases quickly.

## Phase 5: Sync

Goal:
```text
Move Waystone data between repositories and collaborators.
```

Possible transports:
- normal Git files
- dedicated Git refs
- Git bundles
- email attachments
- Radicle bridge
- ForgeFed bridge

I won't decide the sync model before Phase 1 and Phase 2 produce real constraints. Git refs are attractive, but choosing them too early risks designing around a storage preference instead of a workflow.

## Phase 6: Web Viewer

Goal:
```text
Render local Waystone state for humans.
```

Candidate command:
```sh
waystone serve
```

This should stay a viewer over local projected state, not a hosted forge. I'm deferring it because a web UI can make an immature data model look more finished than it is.

## Deferred Work

I won't build these early:
- hosted forge
- CI
- federation
- public directory
- attachment hosting
- GitHub export before GitHub import
- review workflows before issue workflows

Those are not rejected forever. They are deferred because the current project risk is scope creep, not lack of possible features.

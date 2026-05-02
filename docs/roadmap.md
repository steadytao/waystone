# Roadmap

Waystone should move in strict phases. Each phase should prove one layer before expanding the product.

## Phase 0: Design Pack

Status: current phase.

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

Storage starts with deterministic `.waystone/` files, not Git refs. The `.waystone/` directory is canonical; archives package that ledger rather than replacing it.

This phase is read-only import. GitHub export and round-tripping are not part of this phase.

## Phase 2: Local Issue Ledger

Goal:
```text
Create and manage portable issues inside a local Git repository.
```

Candidate commands:
```sh
waystone init
waystone identity create
waystone issue create
waystone issue list
waystone issue show <id>
waystone issue comment <id>
waystone issue close <id>
```

This phase adds local signed records after the imported ledger model has proven useful.

## Phase 3: Patches And Reviews

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

This phase should wait until imported records, local issues, identities and authority are stable.

## Phase 4: Sync

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

Do not decide the sync model before Phase 1 and Phase 2 produce real constraints.

## Phase 5: Web Viewer

Goal:
```text
Render local Waystone state for humans.
```

Candidate command:
```sh
waystone serve
```

This should be a viewer over local projected state, not a hosted forge.

## Deferred Work

Do not build these early:
- hosted forge
- CI
- federation
- public directory
- attachment hosting
- GitHub export before GitHub import
- review workflows before issue workflows

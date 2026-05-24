# Roadmap

This roadmap is written as maintainer notes, not as a promise that every phase will happen exactly this way.

The main constraint I want to preserve is sequencing. I need to prove one layer before expanding the product.

## Phase 0: Design Pack

Status: complete enough for the current project stage.

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

Status: complete enough for the current project stage.

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

This phase is read-only import. I'm not doing GitHub export or round-tripping here because import needs to be boring and trustworthy first.

## Phase 2: Local Issue Ledger

Status: complete enough for the current project stage.

Goal:
```text
Create and manage portable issues inside a local Git repository.
```

Current local authoring surface:
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

This creates local history under `waystone:` sources only. Bare `owner/repo` names are accepted for local-authoring commands that do not touch imported forges. Assignment, milestones, sync and conflict handling remain deferred.

## Phase 3: Multi-forge Import And Migration Reports

Status: complete enough for the current project stage.

Goal:
```text
Prove Waystone is not GitHub-shaped and can report migration risk across multiple source namespaces.
```

Import commands:
```sh
waystone github import owner/repo
waystone gitlab import group/project
waystone forgejo import owner/repo
waystone gitea import owner/repo
```

Cross-source report command:
```sh
waystone migrate report \
  --from github:owner/repo \
  --from gitlab:group/project \
  --from forgejo:owner/repo \
  --from gitea:owner/repo \
  --to waystone:owner/repo
```

The migration report is read-only. It counts imported records, local continuation records, identity handling and known gaps such as attachments, user mapping and CI history. Cross-source reports keep source namespaces separate, detect source-local number collisions and warn about ambiguous labels, milestones and authors without merging them.

## Phase 4: Saved Migration Plans

Status: implemented in `v0.2.0-alpha.2`.

Goal:
```text
Turn migration reporting into deterministic, reviewable migration-plan artefacts.
```

Single-source plan:
```sh
waystone migrate plan \
  --from github:owner/repo \
  --to waystone:owner/repo \
  --numbering-strategy preserve-source-numbering \
  --out migration-plan.json
```

Multi-source plan:
```sh
waystone migrate plan \
  --from github:owner/repo \
  --from gitlab:group/project \
  --from forgejo:owner/repo \
  --from gitea:owner/repo \
  --to waystone:owner/repo \
  --numbering-strategy preserve-source-numbering \
  --out migration-plan.json
```

The plan format must preserve original source IDs and source-local numbers. It must not merge GitHub issue `#1`, GitLab issue `#1`, Forgejo issue `#1` and Gitea issue `#1` into one imagined target record.

Release: `v0.2.0-alpha.2`.

## Phase 5: Plan Inspection And Verification

Status: implemented in `v0.2.0-alpha.3`.

Goal:
```text
Make migration plans reviewable and independently valid before any export dry-run exists.
```

Commands:
```sh
waystone migrate inspect migration-plan.json
waystone migrate verify migration-plan.json
```

`migrate inspect` shows plan version, from/to sources, strategy axes, record counts, warnings and the fact that target writes are disabled.

`migrate verify` checks JSON shape, supported version, required fields, supported strategy values, duplicate records, source namespaces and deterministic target keys.

Release: `v0.2.0-alpha.3`.

## Phase 6: Conformance And Identity Documentation

Status: implemented in `v0.2.0-beta.1`.

Goal:
```text
Prove multiple forge shapes can coexist and document identity rules clearly.
```

Add:
```text
docs/migration-identity.md
```

The identity rule:
```text
Original source identity is evidence. Target identity is a projection.
```

This phase proves, through CLI conformance coverage, that GitHub, GitLab, Forgejo, Gitea and local `waystone:` records can coexist without merged provenance. Matching numbers, labels, milestone names or author names across sources do not imply the same record.

Release: `v0.2.0-beta.1`.

## Phase 7: Strategy File And Structured Loss Report

Status: implemented in `v0.2.0-beta.2`.

Goal:
```text
Make migration policy explicit and report unsupported data in structured form.
```

Commands:
```sh
waystone migrate plan \
  --from github:owner/repo \
  --from gitlab:group/project \
  --to waystone:owner/repo \
  --strategy-file migration-strategy.json \
  --out migration-plan.json

waystone migrate loss-report \
  --from github:owner/repo \
  --from gitlab:group/project \
  --to waystone:owner/repo \
  --json
```

The first strategy file accepts only safe read-only defaults. The loss report covers attachments, review threads, CI history, workflows, permissions, branch protections, user mapping, release assets and visibility uncertainty where relevant.

Release: `v0.2.0-beta.2`.

## Phase 8: Compatibility Policy And Fixtures

Status: implemented in `v0.2.0-rc.1` and finalised in `v0.2.0`.

Goal:
```text
Define what another tool may safely consume.
```

Add:
```text
docs/compatibility.md
testdata/conformance/single-github-ledger/
testdata/conformance/multi-source-ledger/
testdata/conformance/local-labelled-issues-ledger/
testdata/conformance/migration-plan-v1/
testdata/conformance/migration-loss-report-v1/
```

The v0.2.0 release line is closed to new features. Follow-up work should be bug fixes, documentation corrections or compatibility clarifications.

Release: `v0.2.0-rc.1`, finalised in `v0.2.0`.

## Phase 9: Waystone v0.2.0

Status: implemented in `v0.2.0`.

Goal:
```text
Provide a bridge-ready contract for portable project history and migration planning.
```

Cut `v0.2.0` when:
- multi-source report exists
- multi-source plan exists
- plan inspect exists
- plan verify exists
- loss report exists
- conformance fixtures exist
- compatibility policy exists
- migration identity docs exist
- all release checks pass
- docs match CLI behaviour
- no known blocker remains

This is the point where a separate bridge tool can start consuming Waystone's plan format.

Release: `v0.2.0`.

## After v0.2

A future bridge tool may consume Waystone migration plans and produce target-specific export dry-runs. That work should stay outside Waystone. Waystone should not become the place for remote target mutation, provider-specific export policy or forge adapter maintenance.

Waystone v0.3 should be post-bridge hardening based on real findings, not speculative pre-bridge abstraction.

## Deferred Work

I won't build these before the v0.2 contract is done:
- hosted forge
- CI
- federation
- public directory
- attachment hosting
- live export
- sync back
- mirror delegation
- plugin architecture
- SQLite canonical storage
- web UI
- assignment
- milestones
- label editing or deletion
- local pull requests

Those are not rejected forever. They are deferred because the current bottleneck is whether Waystone can produce deterministic, reviewable, multi-source migration plans that another tool can consume.

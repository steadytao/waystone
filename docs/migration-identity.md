# Migration Identity

Waystone migration plans preserve source identity first. They do not guess that matching numbers, names or logins across forges describe the same thing.

The core rule is:

```text
Original source identity is evidence. Target identity is a projection.
```

## Source Identity

A source identity is the original namespace and identifier assigned by a forge or by Waystone local authoring.

Examples:

```text
github:example/project issue #1
gitlab:example/project issue #1
forgejo:example/project issue #1
gitea:example/project issue #1
waystone:example/project issue #1
```

These are five different source records. The shared number `#1` is source-local display identity, not global identity.

Waystone preserves source identity in migration plans through:

- `source`, the source namespace such as `github:example/project`
- `source_id`, the immutable source or ledger ID for the original record
- `source_number`, the source-local issue, pull request, merge request or milestone number where one exists
- `source_url`, the original URL where one exists
- `waystone_id`, the canonical ID currently stored by the Waystone ledger

## Canonical Waystone Identity

Waystone IDs are canonical ledger references. They are not target-forge IDs and they are not permission to merge records from different sources.

For imported records, Waystone keeps source-derived IDs such as:

```text
github:issue:123456
gitlab:issue:234567
forgejo:issue:345678
gitea:issue:456789
```

For local Waystone labels, Waystone uses immutable local IDs such as:

```text
lbl_...
```

The display name, slug or source-local number may change or collide. The canonical ID is the durable ledger reference.

## Target Identity

Target identity is a proposed projection into another source. In a read-only migration plan, target identity is not assigned by a remote forge.

For the initial `preserve-source-numbering` strategy, Waystone creates deterministic target keys such as:

```text
github:example/project:issue:1
gitlab:example/project:issue:1
forgejo:example/project:issue:1
gitea:example/project:issue:1
```

Those keys are review artefacts. They preserve the source namespace so a later bridge can reason about target writes without pretending that all `#1` records are the same record.

If a future target forge assigns a new issue number, that number is a target projection. It must be recorded as a mapping result, not as a replacement for the original source identity.

## Source-Local Numbers

Issue, pull request, merge request and milestone numbers are local to their source.

`github:example/project#1` and `gitlab:example/project#1` can both exist in the same Waystone ledger. They may be related by project history, but that relationship is not implied by the number alone.

Waystone reports number collisions as ambiguity, not as a merge instruction.

## Display Keys

Display keys exist for humans. They should be stable enough to inspect but must not replace underlying identity.

Examples:

```text
github:example/project#1
gitlab:example/project!2
waystone:example/project#1
Software Issue (bug)
```

Display keys can collide or be renamed. Migration plans must continue to carry immutable source and Waystone IDs.

## Authors

The same author login on two forges does not prove the same human identity.

Examples:

```text
github:alice
gitlab:alice
forgejo:alice
gitea:alice
```

Waystone preserves the source author snapshot and reports cross-source login ambiguity. It does not map users to local identities unless an explicit future mapping layer says how to do so.

The safe default is:

```text
author_mapping_strategy = preserve-source-author
```

## Labels

The same label name on two sources does not prove the same taxonomy.

Examples:

```text
github:example/project label "bug"
gitlab:example/project label "bug"
waystone:example/project label "Software Issue" with slug "bug"
```

These may represent the same concept, similar concepts or entirely different project taxonomies. Waystone reports label name overlap without silently merging labels.

Imported labels remain source records. Local Waystone labels have immutable local IDs and can be displayed with mutable names and stable slugs.

The safe default is:

```text
label_mapping_strategy = preserve-source-labels
```

## Milestones

The same milestone title on two sources does not prove the same milestone.

Examples:

```text
github:example/project milestone "v1"
gitlab:example/project milestone "v1"
```

Waystone reports milestone title overlap. It does not merge milestones by title.

The safe default is:

```text
milestone_mapping_strategy = preserve-source-milestones
```

## Migration Plans

A migration plan is a saved, read-only artefact. It describes how records would map under explicit safe defaults. It does not contact a forge and does not create target records.

Each plan record must carry source identity explicitly:

```json
{
  "object": "issue",
  "source": "github:example/project",
  "source_id": "github:issue:123456",
  "source_number": 1,
  "source_url": "https://github.com/example/project/issues/1",
  "waystone_id": "github:issue:123456",
  "target_source": "waystone:example/project",
  "target_key": "github:example/project:issue:1",
  "numbering_strategy": "preserve-source-numbering"
}
```

The important invariant is that the original identity remains present even when a target projection is proposed.

## What Waystone Refuses To Infer

Waystone does not infer that:

- the same issue number across sources means the same issue
- the same pull request or merge request number across sources means the same change proposal
- the same author login across sources means the same human
- the same label name across sources means the same taxonomy
- the same milestone title across sources means the same project target
- a target-assigned number replaces the original source number

Those decisions require explicit mapping policy. They are not safe defaults.

## Waybridge Boundary

Waystone defines and verifies the ledger and migration plan contract. A future bridge tool may consume that contract and ask a target forge what can be created.

That bridge may later record target mappings such as:

```text
github:example/project:issue:1 -> gitlab:example/project issue #42
```

That target mapping is additional evidence. It does not erase the original source identity.

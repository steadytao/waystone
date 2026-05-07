# Local Issue Ledger

Waystone has two different source categories:
- imported forge sources, such as `github:owner/repo`
- local Waystone sources, such as `waystone:owner/repo`

Imported forge sources are read-only evidence. Waystone should preserve them, verify them and export them, but it should not mutate them locally.

Local Waystone sources are appendable local records. They are used for issues, labels, comments and lifecycle events authored directly into the ledger.

## Creating A Local Issue

Create an issue:
```sh
waystone issue create --source owner/repo --title "Follow up on imported history"
```

For local-only authoring commands, `owner/repo` is shorthand for `waystone:owner/repo`.

The explicit form also works:
```sh
waystone issue create --source waystone:owner/repo --title "Follow up on imported history"
```

Waystone refuses imported sources:
```sh
waystone issue create --source github:owner/repo --title "This will fail"
```

## Labels

Create a local label:
```sh
waystone label create --source owner/repo --slug migration --name "Migration" --color 0e8a16
```

Apply it to an issue:
```sh
waystone issue label add --source owner/repo --issue 1 migration
```

Remove it from an issue:
```sh
waystone issue label remove --source owner/repo --issue 1 migration
```

Local labels have immutable IDs, stable slugs and mutable display names. Issue JSON stores label IDs. Human-readable issue and timeline output resolves those IDs into labels such as `Migration (migration)`.

Local labels are stored under:
```text
.waystone/objects/waystone/<owner>/<repo>/labels/
```

Label add/remove history is stored as `issue.labeled` and `issue.unlabeled` events.

## Comments

Edit an issue title or body:
```sh
waystone issue edit --source owner/repo --issue 1 --title "Updated title"
waystone issue edit --source owner/repo --issue 1 --body-file issue.md
```

The issue JSON stores the current title and body. Edit history is recorded as `issue.edited` events. Edit events record the new title or body, not a previous-content snapshot.

Add a comment:
```sh
waystone issue comment --source owner/repo --issue 1 --body "I checked this locally."
```

Read a comment body from a file:
```sh
waystone issue comment --source owner/repo --issue 1 --body-file comment.md
```

Local comments are stored under:
```text
.waystone/objects/waystone/<owner>/<repo>/comments/
```

## Close And Reopen

Close an issue:
```sh
waystone issue close --source owner/repo --issue 1
```

Reopen an issue:
```sh
waystone issue reopen --source owner/repo --issue 1
```

The issue JSON stores current state. Close and reopen history is stored separately as issue event JSON under:
```text
.waystone/objects/waystone/<owner>/<repo>/events/
```

That split keeps the current issue easy to read while preserving lifecycle history for timeline output.

## Timeline

Show the local issue timeline:
```sh
waystone issue timeline --source waystone:owner/repo 1
```

A local issue timeline can include:
- `issue.opened`
- `issue.edited`
- `issue.labeled`
- `issue.unlabeled`
- `issue.comment`
- `issue.closed`
- `issue.reopened`

## Browsing

List local open issues:
```sh
waystone issue list --source waystone:owner/repo --state open
```

List local closed issues:
```sh
waystone issue list --source waystone:owner/repo --state closed
```

Search local issues:
```sh
waystone issue search --source waystone:owner/repo --state all "release"
```

`--state` accepts `open`, `closed` and `all`.

## Verification And Export

Local authored issue history should survive the same integrity checks and archive path as imported history:
```sh
waystone ledger verify --strict
waystone ledger export --out waystone-local-issues
waystone ledger inspect waystone-local-issues
waystone ledger import waystone-local-issues
```

Importing Waystone data must never execute anything.

# Examples

This page shows common Waystone workflows using the default `.waystone/` ledger.

## Import A Repository

Authenticate with GitHub:
```sh
waystone github auth login
```

Create a local operation-signing identity:
```sh
waystone identity init
```

Import a repository:
```sh
waystone github import steadytao/waymark
```

Import a GitLab project:
```sh
waystone gitlab import example/project
```

Import a Forgejo repository:
```sh
waystone forgejo import example/project
```

Import a Gitea repository:
```sh
waystone gitea import example/project
```

Set the imported repository as the default browsing source:
```sh
waystone source default github:steadytao/waymark
```

## Browse Imported History

List issues from the default source:
```sh
waystone issue list
```

List only open local issues:
```sh
waystone issue list --source waystone:steadytao/waystone --state open
```

Show an issue:
```sh
waystone issue show 15
```

Show issue comments:
```sh
waystone issue comments 15
```

Create a local Waystone issue beside imported history:
```sh
waystone issue create --source steadytao/waystone --title "Follow up on imported history"
```

Create and apply a local label:
```sh
waystone label create --source steadytao/waystone --slug migration --name "Migration"
waystone issue label add --source steadytao/waystone --issue 1 migration
```

Edit a local Waystone issue:
```sh
waystone issue edit --source steadytao/waystone --issue 1 --title "Follow up on preserved history"
```

Comment on a local Waystone issue:
```sh
waystone issue comment --source steadytao/waystone --issue 1 --body "Checked locally."
```

Close and reopen a local Waystone issue:
```sh
waystone issue close --source steadytao/waystone --issue 1
waystone issue reopen --source steadytao/waystone --issue 1
```

Local issue authoring is source-local. It does not mutate imported `github:` sources.

List pull requests:
```sh
waystone pr list
```

Show a pull request:
```sh
waystone pr show 14
```

## Verify The Ledger

Run strict verification:
```sh
waystone ledger verify --strict
waystone ledger verify --strict --signatures
```

Run practical health checks:
```sh
waystone ledger doctor
```

Strict verification checks JSON validity, operation-chain integrity and recorded object hashes. It detects accidental edits and local tampering, but it does not prove that the original forge content was correct.

## Report Migration Shape

Generate a read-only migration report:
```sh
waystone migrate report --from github:steadytao/waymark --to waystone:steadytao/waystone
```

The report counts preserved source records, local continuation records and known gaps. It does not contact a forge or write target records.

Write a read-only migration plan:
```sh
waystone migrate plan --from github:steadytao/waymark --to waystone:steadytao/waystone --numbering-strategy preserve-source-numbering --out waystone-migration-plan.json
```

The plan records how source records would map. It does not create target records.

## Export The Ledger

Export the full ledger:
```sh
waystone ledger export --out waystone-ledger
```

Inspect an archive before importing it:
```sh
waystone ledger inspect waystone-ledger
```

Importing a Waystone archive must never execute ledger contents.

## Source-Local Numbers

Waystone treats issue, pull request and milestone numbers as source-local.

For example:
```text
github:example/project#1
waystone:example/project#1
```

Those are different records, even though both display as `#1`.

The fixture in [`../testdata/ledgers/overlapping-sources/.waystone/`](../testdata/ledgers/overlapping-sources/.waystone/) demonstrates this with one GitHub source and one repo-local `waystone:` source.

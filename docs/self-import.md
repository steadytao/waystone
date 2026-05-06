# Self-import Validation

Waystone can be run against its own GitHub repository as a simple validation flow. This imports Waystone's public project history into a local ledger, verifies the ledger and exports an archive for inspection.

## What It Checks

The self-import flow exercises:
- GitHub authentication
- GitHub repository import
- source manifest creation
- issue and pull request browsing
- strict ledger verification
- archive export
- archive inspection

It does not prove that GitHub's source data is true. It proves that Waystone can record, browse, verify and export what it imported.

## Run The Flow

Import Waystone's project history:
```sh
waystone github import steadytao/waystone --v
```

Set the imported repository as the default browsing source:
```sh
waystone source default github:steadytao/waystone
```

Inspect the local ledger:
```sh
waystone ledger summary
waystone source status
waystone issue list
waystone pr list
```

Verify the ledger:
```sh
waystone ledger verify --strict
```

Export and inspect an archive:
```sh
waystone ledger export --out waystone-test
waystone ledger inspect waystone-test
```

## Expected Result

The exact object counts will change as the repository changes.

A successful run should show:
- a completed import operation
- at least one source under `github:steadytao/waystone`
- successful strict ledger verification
- a completed archive export
- a successful archive inspection with `Manifest true`

If no local identity exists in the temporary ledger, archive inspection should report `Signed false`. That is expected. To include signatures in the validation flow, run `waystone identity init` before importing.

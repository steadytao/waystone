# Waystone CLI

Waystone reads and writes a local ledger at `.waystone` by default.

Most browsing commands support `--source <system>:<owner>/<repo>`, for example `github:steadytao/waymark`. If no source is supplied, Waystone uses the default source from `ledger.json` when one is set.

Most read-only display commands support `--json`.

## Version

```sh
waystone version
waystone version --json
```

## Identity

```sh
waystone identity init
waystone identity init --name steadytao
waystone identity list
waystone identity show
waystone identity show --json
waystone identity status
waystone identity trust <identity-id>
waystone identity untrust <identity-id>
```

`identity init` creates a local Ed25519 signing identity for operation records
and source manifests. It writes a signed operation record and trusts the new
identity locally. New operation records and source manifests are signed
automatically once a default identity exists.

`identity show` prints the public identity. Private key material is local and is
not included in ledger exports.

`identity list`, `identity status`, `identity trust` and `identity untrust`
manage local trust policy. Trust is ledger-local and based on Waystone identity
IDs. It is not a global identity system.

## GitHub Authentication

```sh
waystone github auth login
waystone github auth login --client-id <client-id>
waystone github auth login --plain-file-store
waystone github auth logout
```

Credential rules:
- `GITHUB_TOKEN` always wins and is never stored
- `OAUTH_CLIENT_ID` or `--client-id` can override the built-in OAuth client ID
- the OS credential store is used by default
- `--plain-file-store` is only a development fallback

## GitHub Audit, Import And Refresh

```sh
waystone github audit steadytao/waymark
waystone github audit steadytao/waymark --json
waystone github audit steadytao/waymark --verbose
waystone github audit steadytao/waymark --no-write
waystone github import steadytao/waymark
waystone github import steadytao/waymark --v --concurrency 8
waystone github refresh steadytao/waymark
```

Audit inspects GitHub dependency surfaces that can make repository migration harder. It reports workflow files, action references, Dependabot, CodeQL, issue templates, pull request templates, CODEOWNERS, default-branch protection, repository secret and variable counts, environment counts, GitHub Pages and release assets.

Audit writes an audit object and operation record to `.waystone/` by default. Use `--no-write` for report-only output.

Audit does not execute workflows or repository code. It does not collect secret or variable values. Optional surfaces distinguish absence from inaccessible evidence where GitHub returns permission failures. Use `--verbose` or `-v` to list every action reference; default human output summarises action kinds to avoid noisy reports on large repositories.

Import fetches repository metadata, issues, pull requests, comments, labels, milestones, releases and review comments.

Expected output shape:
```text
Repository     steadytao/waymark
Ledger         .waystone
Auth           authenticated
- Fetching repository metadata...
- Fetching issues and pull request references...
- Fetching issue and pull request conversation comments...
- Fetching pull request details and review comments...
- Fetching labels...
- Fetching milestones...
- Fetching releases...
- Writing ledger...

Import complete
  Operation        github-import-...
  Created          49
  Updated          0
  Deleted          0
  Unchanged        0
```

Use `--v` or `--verbose` for per-record fetch progress.

## Sources

```sh
waystone source list
waystone source show github:steadytao/waymark
waystone source default github:steadytao/waymark
waystone source default
waystone source default --clear
waystone source inspect github:steadytao/waymark
waystone source status
waystone source status --stale-after 7d
waystone source refresh
waystone source refresh --source github:steadytao/waymark
waystone source refresh --sources github:steadytao/waymark,github:steadytao/surveyor
```

Source commands expose import manifests: identity, object refs, operation refs, object counts and refresh state.

Source names are repo-specific namespaces. GitHub imports use `github:owner/repo`; local Waystone records use `waystone:owner/repo`. Numbers are source-local, so the same issue number can exist in multiple sources without representing the same record.

Refresh behaviour:
- `source refresh` refreshes every source by default
- `--source` or `--sources` narrows refresh to selected sources
- the browsing default source does not limit refresh

`--stale-after` accepts durations such as `7d`, `24h` or `0`. Use `0` to disable stale checks.

## Ledger

```sh
waystone ledger summary
waystone ledger status
waystone ledger history
waystone ledger show-operation <operation-id>
waystone ledger verify
waystone ledger verify --strict
waystone ledger verify --strict --signatures
waystone ledger doctor
waystone ledger doctor --stale-after 7d
waystone ledger diff --source github:steadytao/waymark --since <operation-id>
waystone ledger diff --source github:steadytao/waymark --since <operation-id> --include-verified
```

`ledger verify` checks JSON files and writes a verification operation.

`ledger verify --strict` also checks operation-chain integrity and recorded file hashes. `--operations` is an alias.

`ledger verify --strict --signatures` also verifies operation signatures and source manifest signatures. Unsigned records are reported, not rejected. Valid signatures are reported as trusted or untrusted according to local trust policy. Invalid signatures fail verification.

`ledger doctor` reports practical ledger problems such as no default source, stale sources, missing operation history, ambiguous issue numbers or failed integrity checks.

`ledger diff` reads local operation records only. It does not contact the forge.

## Audits

```sh
waystone audit list
waystone audit list --source github:steadytao/waymark
waystone audit show <audit-id>
waystone audit show <audit-id> --verbose
waystone audit show <audit-id> --json
```

Audit browse commands read locally persisted audit records. They do not contact GitHub.

## Archive Export And Import

```sh
waystone ledger export --out waystone-ledger
waystone ledger export --source github:steadytao/waymark --out waystone-waymark
waystone ledger export --format json --out waystone-ledger.json
waystone ledger export --format json --compact --out waystone-ledger.json
waystone ledger inspect waystone-ledger
waystone ledger import waystone-ledger
waystone ledger import waystone-ledger --unsafe
```

Archive export writes a zstd-compressed tar stream by default. Export refuses to write unless strict verification passes.

Archive exports include a `WAYSTONE-MANIFEST.json` tar entry that records exported files, hashes, included sources, operation-chain head and optional archive-manifest signature.

JSON export writes a single inspection bundle. `--compact` removes formatting from that JSON export only; it does not rewrite the ledger.

Safe import verifies the archive manifest, archive shape and ledger operation chain, then confirms GitHub sources through authenticated GitHub API access.

`--unsafe` skips remote source confirmation. It does not allow path traversal or unsupported archive entries.

## Issues

```sh
waystone issue create --source steadytao/waystone --title "Example issue"
waystone issue create --source waystone:steadytao/waystone --title "Example issue"
waystone issue create --source waystone:steadytao/waystone --title "Example issue" --body-file issue.md
waystone issue edit --source steadytao/waystone --issue 1 --title "Updated issue"
waystone issue edit --source steadytao/waystone --issue 1 --body "Updated body"
waystone issue edit --source steadytao/waystone --issue 1 --body-file issue.md
waystone issue comment --source steadytao/waystone --issue 1 --body "Example comment"
waystone issue comment --source steadytao/waystone --issue 1 --body-file comment.md
waystone issue close --source steadytao/waystone --issue 1
waystone issue reopen --source steadytao/waystone --issue 1
waystone issue list
waystone issue list --source github:steadytao/waymark
waystone issue list --state open
waystone issue list --source waystone:steadytao/waystone --state closed
waystone issue search "edge inspection"
waystone issue search --state open "edge inspection"
waystone issue search --source github:steadytao/waymark "edge inspection"
waystone issue search --field label Tracking
waystone issue search --field all steadytao
waystone issue show 15
waystone issue show --with-comments 15
waystone issue comments 15
waystone issue timeline 15
```

`issue create`, `issue edit`, `issue comment`, `issue close` and `issue reopen` write local issue records under a `waystone:` source. Because they only write local Waystone records, bare `owner/repo` source names are treated as `waystone:owner/repo`.

These commands refuse imported sources such as `github:owner/repo`.

The first local lifecycle path is intentionally narrow. It supports title/body edits only. It does not assign labels, sync with a forge or publish the issue remotely.

Unfiltered lists include a source column. Use `--source` to scope list and search output to a single source. Use `--state open`, `--state closed` or `--state all` to filter issue state. Detail commands require `--source` when the same issue number exists in multiple imported repositories.

`issue search` searches title and description/body by default. Use repeated `--field` flags or comma-separated values to search `title`, `description`, `author`, `state`, `label`, `milestone`, `url` or `all`.

`issue timeline` combines the issue, edits, comments and close or reopen events chronologically.

## Pull Requests

```sh
waystone pr list
waystone pr list --source github:steadytao/waymark
waystone pr search "release"
waystone pr search --field branch master
waystone pr search --field all steadytao
waystone pr show 12
waystone pr show --with-comments 12
waystone pr comments 12
waystone pr timeline 12
```

Unfiltered lists include a source column. Detail commands require `--source` when the same pull request number exists in multiple imported repositories.

`pr search` searches title and description/body by default. Use repeated `--field` flags or comma-separated values to search `title`, `description`, `author`, `state`, `branch`, `url` or `all`.

`pr timeline` combines the pull request, conversation comments, review comments and close or merge event chronologically.

## Labels And Milestones

```sh
waystone label list
waystone label list --source github:steadytao/waymark
waystone milestone list
waystone milestone list --source github:steadytao/waymark
```

Unfiltered label and milestone lists include a source column.

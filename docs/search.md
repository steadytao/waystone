# Search

Waystone search is local ledger search. It does not call GitHub and it only
searches records already imported into `.waystone/`.

## Defaults

```sh
waystone issue search "edge inspection"
waystone pr search "release"
```

By default, search checks title and description/body text. This keeps normal
queries focused on the human-written subject and description rather than every
piece of metadata attached to a record.

Search is case-insensitive and substring-based. It is intentionally simple for
the current prototype.

## Source Scope

```sh
waystone source default github:steadytao/waymark
waystone issue search "dns"
waystone issue search --source github:steadytao/surveyor "tls"
waystone issue search --source waystone:steadytao/waystone --state closed "release"
```

If a default source is configured, search uses it. An explicit `--source` flag
always wins. Without either, search spans all imported sources and includes a
source column in text output.

Issue search can also filter by current issue state with `--state open`, `--state closed` or `--state all`.

## Fields

Use `--field` to search specific fields.
```sh
waystone issue search --field label Tracking
waystone issue search --field label migration
waystone issue search --field milestone "v1.0.0"
waystone issue search --field author steadytao
waystone pr search --field branch master
waystone pr search --field state closed
```
`--field` can be repeated or comma-separated:
```sh
waystone issue search --field title --field label dns
waystone issue search --field title,label dns
```

Use `--field all` when the query should match any supported field:
```sh
waystone issue search --field all steadytao
waystone pr search --field all release
```

Supported issue fields:
- `title`
- `description` or `body`
- `author`
- `state`
- `label` or `labels`, including local label IDs, slugs and display names
- `milestone`
- `url`
- `all`

Supported pull request fields:
- `title`
- `description` or `body`
- `author`
- `state`
- `branch`
- `base`
- `head`
- `url`
- `all`

## Output

Text output includes the first matching field:
```text
Issues         1

NUMBER   STATE    MATCH        TITLE
#15      open     title        Tracking: Waymark Check v1.0.0 read-only edge inspection
```

Use `--json` for machine-readable results:
```sh
waystone issue search "edge" --json
waystone pr search "release" --json
```

## Limitations

Search is not a full-text index. It does not rank results, stem words, search
comments or fetch missing remote data. Those are future improvements once the
ledger format and source model are stable.

# Privacy

Waystone is a local CLI. It stores imported project history in a local `.waystone/` ledger and does not run a hosted service.

## OAuth Tokens

`GITHUB_TOKEN` always takes precedence and is never persisted by Waystone.

If no token is provided through the environment, `waystone github auth login` uses GitHub OAuth device flow and stores the resulting token in the operating system credential store.

`--plain-file-store` is an explicit development fallback. It stores the token in a local plaintext file and should not be used on shared machines.

## Imported Data

Waystone imports repository metadata, issues, pull requests, comments, labels, milestones and releases from GitHub.

For public repositories this is usually public project discussion but the local ledger can still contain sensitive context depending on what participants wrote.

Treat `.waystone/` exports as project-history records, not harmless cache files.

## Actor Metadata

Operation records are privacy-minimal by default.

Waystone may record Git config name and email. It records local OS username and hostname only when `--local` is explicitly used.

GitHub operations may record authenticated GitHub login. They do not record the token.

## Network Access

GitHub import, refresh and safe archive import may contact GitHub.

Local browse, search, status, timeline, verify and inspect commands read the local ledger only.

## Deletion

To remove local imported project history, delete the `.waystone/` directory.

To remove stored GitHub OAuth credentials:
```sh
waystone github auth logout
```

# Credential Storage

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone needs GitHub authentication for private repositories, higher API limits and safe source confirmation during archive import.

Authentication should not turn the ledger into a token store. Ledger files are project-history records and may be exported, copied or committed accidentally if users make mistakes.

Users may also prefer their own OAuth application rather than trusting Waystone's default client ID.

## Decision

Waystone will use the following credential precedence:
1. `GITHUB_TOKEN`
2. operating system credential store populated by `waystone github auth login`
3. explicit plaintext development fallback with `--plain-file-store`

`GITHUB_TOKEN` is process-local and is never persisted by Waystone.

The OAuth device flow is the default interactive login path. Users may provide their own OAuth client ID with `OAUTH_CLIENT_ID` or `--client-id`.

Ledger files must not contain tokens.

## Consequences

This decision means that:
- the normal login path avoids local plaintext token files
- `GITHUB_TOKEN` remains suitable for CI and one-shot commands
- plaintext token storage remains explicit and visibly unsafe
- operation records may record authentication mode and GitHub login but not tokens

Future auth providers should follow the same principle: credentials stay outside the ledger.

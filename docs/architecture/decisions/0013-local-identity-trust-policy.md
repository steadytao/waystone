# Local Identity Trust Policy

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone can sign operation records, source manifests and archive manifests.

Cryptographic validity is not the same as trust. A signature can be valid while being produced by an unknown key. Waystone needs to report that distinction without turning the project into a global identity system.

## Decision

Waystone uses a local trust policy stored in `trust.json`.

The trust policy records trusted Waystone identity IDs. `waystone identity init` creates and trusts the default local identity. Users can change local trust with:
- `waystone identity trust <identity-id>`
- `waystone identity untrust <identity-id>`

`waystone ledger verify --strict --signatures` reports valid signatures as trusted or untrusted according to the local trust policy.

Valid but untrusted signatures are trust findings. Invalid signatures remain integrity failures.

## Consequences

Trust decisions are explicit, local and inspectable.

Waystone does not bind signing keys to GitHub accounts, Git identities, SSH keys, PGP identities or any external identity provider in this first trust-policy step.

Old or imported unsigned ledgers remain readable while the format is experimental.

Future work may add richer trust policy, imported identities or external identity bindings, but those should be separate decisions.

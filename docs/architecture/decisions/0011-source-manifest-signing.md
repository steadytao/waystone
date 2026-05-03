# Source Manifest Signing

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Operation signatures prove that a local operation record was produced by a local signing identity.

Source manifests are the next ledger boundary. They bind a source namespace to object refs and operation refs. Without source manifest signatures, a local edit can change which objects or operations appear to belong to a source while leaving individual signed operations intact.

## Decision

Waystone will sign source manifests when a default local signing identity exists.

The signature covers the canonical source manifest representation with `signature` empty. It covers source identity, object refs and operation refs.

`waystone ledger verify --strict --signatures` verifies operation signatures and source manifest signatures.

Unsigned source manifests are reported, not rejected. Invalid source manifest signatures are integrity failures.

## Consequences

Source manifest signatures prove that the signed manifest indexed a particular set of object refs and operation refs.

They do not prove that imported GitHub content was true.

They do not replace per-object SHA-256 hashes. Strict verification still uses object hashes to detect changes to individual object files.

Trust policy, archive manifests and archive signatures remain deferred.

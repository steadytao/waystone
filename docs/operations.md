# Operations

Waystone operation records are the ledger's command history.

They answer:
- what command ran
- what ledger state changed or was verified
- which operation came before it
- whether recorded content still hashes correctly

## Identity

Each operation has an ID derived from command name and start timestamp.

Example:
```text
source-refresh-20260501T200741.516268000Z
```

The filename also includes a short hash suffix so operation files remain stable and filesystem-safe.

## Command And Arguments

`command` stores the logical command.

Examples:
```text
github import
source refresh
ledger verify --strict
source default
identity trust
```

`args` stores command arguments for auditability. Commands must avoid recording secrets.

## Actor And Auth

Operation actor metadata is privacy-minimal by default.

Waystone may record Git config name and email. Local OS username and hostname are only recorded with `--local`.

GitHub operations may record authentication mode and authenticated GitHub login. Tokens are never stored.

## Input, Output And Changes

`input` records the source or target being acted on.

`output` records ledger path, created/updated/deleted/unchanged counts and summary counts where relevant.

`changes` records file-level object changes:
- type: created, updated, deleted or verified
- object kind
- number or ID where available
- relative path
- SHA-256 where available

`ledger diff --source <source> --since <operation>` reads those changes to show what changed for a source after a prior operation. Verification-only changes are hidden by default; use `--include-verified` to include them.

## Chain Semantics

Each new operation links to the previous operation through `previous_operation`.

Each operation records `operation_hash`, computed over the operation with that hash field empty. Strict verification uses this to detect edits to operation records.

The chain is append-only by convention. A future compaction command must define how it preserves or summarises this history before it is implemented.

## Signing

If a default signing identity exists, Waystone signs new operation records,
source manifests and archive manifests.

The signature uses the same canonical operation representation as `operation_hash`, with both `operation_hash` and `signature` empty.

```sh
waystone identity init
waystone ledger verify --strict --signatures
```

`identity init` writes a signed operation record after creating and trusting
the identity. `identity trust` and `identity untrust` also write operation
records because they change ledger trust policy.

Unsigned operation records and source manifests are reported but remain
readable. Valid signatures are reported as trusted or untrusted according to
local trust policy. Invalid signatures fail verification.

Signing must not make local OS username or hostname implicit. Privacy defaults must remain unchanged.

# Local Issue Authoring

<!-- ![Proposed](https://img.shields.io/badge/status-proposed-informational?style=for-the-badge) -->
![Accepted](https://img.shields.io/badge/status-accepted-brightgreen?style=for-the-badge)
<!-- ![Superseded](https://img.shields.io/badge/status-superseded-yellow?style=for-the-badge) -->
<!-- ![Deprecated](https://img.shields.io/badge/status-deprecated-orange?style=for-the-badge) -->
<!-- ![Denied](https://img.shields.io/badge/status-denied-red?style=for-the-badge) -->

## Context

Waystone can preserve imported GitHub project history in a local ledger.

The next product step is to let the ledger create first-party local collaboration records without mutating imported forge sources.

## Decision

Waystone will support source-local issue creation under the `waystone:<owner>/<repo>` namespace.

Local issue creation will not write to imported forge sources. Imported sources such as `github:<owner>/<repo>` remain read-only evidence from the source forge.

The first implementation supports creating open local issues only. Later local lifecycle commands may add comments, close and reopen behaviour without changing the rule that imported forge sources remain read-only.

A local issue is stored as deterministic JSON under `.waystone/objects/waystone/<owner>/<repo>/issues/`.

The source manifest records the issue object hash. A local operation record records the command that created the issue and is signed when a local identity exists.

Future signed append-only events may replace or supplement direct object creation, but v0 keeps the first authoring path deliberately small.

## Consequences

Local issue creation begins Waystone's local collaboration model while keeping the first step narrow.

The `waystone:` namespace is reserved for records authored by Waystone itself. Imported source namespaces remain read-only.

Commands that only author local Waystone records may accept bare `owner/repo` source names as shorthand for `waystone:owner/repo`. Mixed-source commands should keep explicit source names to avoid ambiguity.

The command surface should not imply that Waystone can yet sync, publish or reconcile locally authored issues with any remote forge.

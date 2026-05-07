# Decisions

This directory contains Waystone's Architecture Decision Records. They are also called ADRs.

These records document material technical and project-boundary decisions in a way that remains understandable over time.

## Purpose

The purpose of this directory is to:
- record important decisions and their rationale
- reduce undocumented architectural drift
- preserve context for future maintainers
- distinguish deliberate choices from accidental behaviour

## Reading Order

The intended reading order is numerical.

Start with:
- [`0000-adr-process.md`](0000-adr-process.md)

Then continue through the later records in order.

Current records:
- [`0001-project-scope.md`](0001-project-scope.md)
- [`0002-language-choice.md`](0002-language-choice.md)
- [`0003-local-ledger-storage.md`](0003-local-ledger-storage.md)
- [`0004-github-import-first.md`](0004-github-import-first.md)
- [`0005-credential-storage.md`](0005-credential-storage.md)
- [`0006-operation-chain-and-hashes.md`](0006-operation-chain-and-hashes.md)
- [`0007-archive-format.md`](0007-archive-format.md)
- [`0008-signing-order.md`](0008-signing-order.md)
- [`0009-agent-instructions.md`](0009-agent-instructions.md)
- [`0010-local-operation-signing.md`](0010-local-operation-signing.md)
- [`0011-source-manifest-signing.md`](0011-source-manifest-signing.md)
- [`0012-archive-manifest-signing.md`](0012-archive-manifest-signing.md)
- [`0013-local-identity-trust-policy.md`](0013-local-identity-trust-policy.md)
- [`0014-local-issue-authoring.md`](0014-local-issue-authoring.md)
- [`0015-local-issue-lifecycle.md`](0015-local-issue-lifecycle.md)
- [`0016-local-issue-labels.md`](0016-local-issue-labels.md)
- [`0017-migration-mapping.md`](0017-migration-mapping.md)

## Notes

An ADR is not required for every change.

For the rules governing when an ADR is required and how ADRs should be written, see [`0000-adr-process.md`](0000-adr-process.md).

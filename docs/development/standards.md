# Standards

This document defines the current development standards for Waystone.

These standards exist to keep the codebase understandable, reviewable and appropriate for a serious infrastructure and project-portability tool.

## Purpose

Waystone can't accumulate code faster than it accumulates clarity.

The purpose of these standards is to ensure that contributions are:
- understandable
- well-scoped
- testable
- reviewable
- consistent with the project's security and architectural direction

## General Expectations

Contributions should:
- solve a real problem
- be scoped as narrowly as practical
- avoid unnecessary complexity
- avoid hidden behaviour
- keep trust and security implications visible
- be understandable to future maintainers

## Scope Discipline

I don't want code that expands Waystone beyond portable project history for Git repositories to be added casually.

If a change materially affects project scope, trust assumptions, ledger semantics, archive behaviour or public commands, it needs discussion and may require a recorded decision.

## Readability

Code should favour clarity over cleverness.

Contributors should prefer:
- explicit behaviour over surprising behaviour
- straightforward control flow over needless indirection
- descriptive names over short or ambiguous names
- simple data flow over opaque abstractions

## Error Handling

Errors should be handled deliberately.

Contributors should:
- return or surface errors clearly
- avoid swallowing meaningful failures
- provide useful context where it materially improves diagnosis
- avoid vague error messages where specifics are available

## Logging And Output

Output should be useful, deliberate and proportionate.

Contributors should:
- avoid printing sensitive information
- avoid excessive noisy output by default
- make important operational events understandable
- consider privacy and retention implications when adding new operation records

## Configuration And CLI Behaviour

CLI behaviour should be explicit and predictable.

Contributors should:
- avoid surprising defaults
- document new flags clearly
- treat command names, flag names and output shape as public behaviour
- prefer source-scoped behaviour when a command could otherwise be ambiguous

## Security-sensitive Changes

Changes affecting archive import, path handling, token handling, operation records, verification, hashes, signing, privacy or trust assumptions require extra care.

Such changes should be especially well documented and reviewable.

## Documentation Expectations

Documentation must be updated when changes affect:
- behaviour
- ledger format
- archive format
- command-line interfaces
- security properties or assumptions
- operational workflows

## Per-file Copyright And Licence Notices

Waystone intends to follow the OpenSSF Best Practices expectation that source files carry both a copyright statement and a licence statement.

For Waystone-owned source files, scripts and other copyright-affected files where a header is practical, contributors should place a short notice near the beginning of the file.

The standard form is:
```text
Copyright <year> The Waystone Authors
SPDX-License-Identifier: Apache-2.0
```

Use the comment syntax appropriate to the file type.

Examples:
```go
// Copyright <year> The Waystone Authors
// SPDX-License-Identifier: Apache-2.0
```
```sh
# Copyright <year> The Waystone Authors
# SPDX-License-Identifier: Apache-2.0
```

The goal is clarity, machine-readability and durable licensing evidence, not decorative boilerplate.

This expectation applies to new project-owned source files and similar files where the notice is practical. It does not require inappropriate retrofitting of:
- vendored third-party files
- generated files
- file formats where a normal comment header would be misleading or awkward

If a file already has a valid project-owned copyright and licence notice, preserve it unless there is a real reason to normalise it.

## AI-assisted Contributions

AI tools may be used to assist with research, drafting, refactoring, testing or documentation but their use must be disclosed clearly in the pull request.

The human contributor remains fully responsible for the contribution. This includes correctness, security, licensing, originality and fitness for inclusion in Waystone.

AI systems cannot sign off commits under the DCO. Every commit must be signed off by a human author who understands the change and has the legal right to submit it.

Any pull request materially assisted by AI must be reviewed by a human maintainer before it can be merged.

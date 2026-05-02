# Testing

This document defines the current testing expectations for Waystone.

Waystone is intended to preserve and verify project history. Testing is therefore part of the project's correctness, safety and reviewability.

## Purpose

The purpose of testing in Waystone is to:
- reduce the risk of incorrect behaviour
- make changes safer to review and merge
- protect ledger and archive semantics from accidental regression
- improve confidence in security-relevant behaviour
- keep the project maintainable over time

## General Expectations

Contributors should treat testing as part of the change, not as a later task.

Changes should be accompanied by tests when they affect:
- ledger format
- archive import or export
- path handling
- token handling
- operation records
- verification behaviour
- public command behaviour
- previously fixed defects

## Testing Principles

Testing in Waystone should favour:
- deterministic behaviour
- explicit expectations
- narrow and understandable test scope
- regression coverage for important defects
- confidence in security-relevant behaviour without pretending tests prove total safety

## Required Test Coverage by Change Type

### Bug fixes

Bug fixes should normally include a regression test that would have failed before the fix.

If a regression test is not practical, the pull request should explain why.

### Behaviour Changes

Changes to behaviour should include tests that show:
- the intended new behaviour
- any preserved old behaviour that still matters
- any edge conditions that are easy to misunderstand

### CLI Changes

Changes affecting commands or flags should include tests for:
- valid command behaviour
- invalid or ambiguous command handling
- default behaviour where defaults matter
- JSON output where output is intended for tooling

### Security-sensitive Changes

Changes affecting archive import, path traversal resistance, token handling, operation-chain integrity, hashes, signing, privacy or trust boundaries require especially careful testing.

Such changes should include tests where practical and should be reviewed with particular care even when tests exist.

## Types of Tests

As the implementation grows, Waystone is expected to use several kinds of tests, including:
- unit tests
- integration tests
- regression tests
- command-output tests
- archive round-trip tests
- fuzzing for suitable untrusted input surfaces

The exact mix will evolve with the implementation.

## Test Quality Expectations

Tests should be:
- readable
- deterministic where practical
- directly tied to behaviour that matters
- maintained alongside the code they protect

Tests should not exist only to inflate numbers or satisfy superficial coverage goals.

## Pull Request Expectations

A pull request should explain the testing performed.

Where useful, this may include:
- automated tests added or updated
- existing tests exercised
- manual validation performed
- reasons a test could not reasonably be added

## Broken or Missing Tests

Code that materially changes behaviour without adequate testing may be declined even if it appears technically correct.

A lack of tests is a quality and maintenance risk, especially in ledger, archive and security-sensitive areas.

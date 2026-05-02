# Contributing to Waystone

Thank you for your interest in contributing.

Waystone is intended to be a serious infrastructure and project-portability tool. Contributions should improve the project materially and be understandable, justified and reviewable.

## Before Opening an Issue

Before opening an issue, please make sure you have:
- read [`README.md`](README.md)
- read [`docs/architecture/design.md`](docs/architecture/design.md)
- read [`docs/ledger-format.md`](docs/ledger-format.md)
- checked whether the issue already exists
- confirmed that the issue is actually about Waystone

Issues that are too vague to act on may be closed.

## Bug Reports

Bug reports should include:
- what happened
- what you expected to happen
- how to reproduce it
- relevant environment details
- logs or command output in plain text, not screenshots where text would be clearer

## Feature Requests

Feature requests should include:
- the problem being solved
- why the current behaviour is insufficient
- the proposed direction
- expected trade-offs or risks, if known

Not every feature request will be accepted.

Waystone is intentionally scoped. Requests that conflict with the project's scope, security model, design principles or intended operational posture may be declined even if technically feasible.

## Before Opening a Pull Request

Before opening a pull request, make sure you have:
- read the relevant documentation
- scoped the change clearly
- added or updated tests where appropriate
- updated documentation where behaviour changes
- signed off every commit under the DCO

Large or architectural changes should usually begin with an issue or otherwise recorded decision first.

## Pull Request Expectations

Pull requests should be:
- well-scoped
- understandable
- justified by the problem they solve
- accompanied by tests when practical
- accompanied by documentation updates when behaviour, policy or interfaces change

## Commit sign-off

All commits must be signed off.

Use:
```bash
git commit -s
```

By signing off a commit, you certify the contribution under the Developer Certificate of Origin. See [`DCO.md`](DCO.md).

Pull requests containing unsigned commits will not be merged.

## Standards

Contributors must follow the standards documented in:
- [`docs/development/standards.md`](docs/development/standards.md)
- [`docs/development/testing.md`](docs/development/testing.md)

## File Headers and Licensing Notices

Waystone intends to follow the OpenSSF Best Practices expectation that source files carry both a copyright statement and a licence statement.

For new source files, scripts and other copyright-affected files that support comments in a normal way, contributors should add a short header near the beginning of the file using:
```text
Copyright <year> The Waystone Authors
SPDX-License-Identifier: Apache-2.0
```

Use the comment syntax appropriate to the file type.

For example:
```go
// Copyright <year> The Waystone Authors
// SPDX-License-Identifier: Apache-2.0
```
```python
# Copyright <year> The Waystone Authors
# SPDX-License-Identifier: Apache-2.0
```

This rule is intended for Waystone-owned source files and similar project files where a notice is practical.

It does not require maintainers to rewrite:
- vendored third-party material
- generated files where the header would be unstable or misleading
- files whose format makes a normal comment header impractical

Contributors should preserve existing valid headers and should not remove or weaken per-file licensing notices casually.

## AI-assisted Contributions

AI tools may be used to assist with research, drafting, refactoring, testing or documentation but their use must be disclosed clearly in the pull request.

The human contributor remains fully responsible for the contribution. This includes correctness, security, licensing, originality and fitness for inclusion in Waystone.

AI systems cannot sign off commits under the DCO. Every commit must be signed off by a human author who understands the change and has the legal right to submit it.

Any pull request materially assisted by AI must be reviewed by a human maintainer before it can be merged.

## Security issues

Do not open public issues for suspected vulnerabilities.

Use the process in [`SECURITY.md`](SECURITY.md) instead.

## Project direction

Contributions that conflict with the project's intended direction may be declined.

See:
- [`docs/architecture/design.md`](docs/architecture/design.md)
- [`docs/architecture/threat-model.md`](docs/architecture/threat-model.md)
- [`docs/security.md`](docs/security.md)
- [`GOVERNANCE.md`](GOVERNANCE.md)

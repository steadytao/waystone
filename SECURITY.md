# Security Policy

The Waystone project aims to treat security issues seriously and respond to well-scoped reports responsibly.

## Supported Versions

Waystone is currently pre-release software.

At this stage, only the latest development state is in scope. No compatibility or long-term support guarantees are made before a stable release policy is published.

## Reporting a Vulnerability

Do not report security vulnerabilities in public issues.

Report vulnerabilities privately through the repository's GitHub Security Advisory reporting path once the repository is public:
- `https://github.com/steadytao/waystone/security/advisories/new`

Use that GitHub path only for vulnerability reports.

A report should include:
- a clear description of the issue
- the affected component or behaviour
- the security impact
- reproduction steps or a minimal proof
- relevant environment details
- any important assumptions or preconditions

## In Scope

A valid security report should identify a security-relevant flaw in Waystone itself.

Examples of potentially valid reports include:
- unsafe archive import behaviour caused by Waystone
- path traversal or arbitrary file write during import/export
- unintended token persistence or credential exposure
- incorrect strict verification behaviour
- operation-chain or hash-verification bypasses
- trust-boundary violations caused by Waystone

## Out of Scope

The following are generally out of scope unless Waystone itself is the root cause:
- unsafe handling of already-public project discussion
- compromised hosts or already-compromised clients
- unrelated third-party infrastructure failures
- GitHub API behaviour outside Waystone's control
- findings that depend entirely on insecure external software outside Waystone's responsibility

## Disclosure Expectations

Please allow reasonable time for triage, confirmation and remediation before public disclosure.

If a report is valid, the project will aim to acknowledge it, assess impact and coordinate remediation responsibly.

## Related documents

- [`docs/security.md`](docs/security.md)
- [`docs/architecture/threat-model.md`](docs/architecture/threat-model.md)
- [`docs/privacy.md`](docs/privacy.md)

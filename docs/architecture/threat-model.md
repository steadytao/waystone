# Threat Model

Waystone stores and imports collaboration records. That creates a smaller abuse surface than a hosted forge but it still needs explicit boundaries.

## Assets

Important assets include:
- project collaboration history
- maintainer authority
- signing keys
- imported ledgers
- projected issue state
- imported external identities
- release records and review history

## Threats

Relevant threats include:
- spam issues and comments
- fake maintainer actions
- identity impersonation
- poisoned imports
- malicious ledgers
- oversized objects
- harassment or abusive content in comments
- forged release or review records
- rewritten or conflicting metadata
- malware links in issue bodies

## Core Rule

Importing Waystone data must never execute anything.

Waystone ledgers, imported forge records, local events, comments, issue bodies and review records are data. They must not trigger commands, hooks, scripts, CI jobs, webhooks or network requests during import.

## Credential Storage

Waystone won't persist environment-provided tokens.

For GitHub imports, `GITHUB_TOKEN` takes precedence and remains process-local. OAuth device-flow tokens should be stored in the operating system credential store by default. Plaintext file storage is only acceptable as an explicit development fallback and needs to be visibly labelled as such in the CLI.

Waystone needs a logout command that deletes stored credentials. Future implementations should avoid broad OAuth scopes by default and won't store refresh tokens unless a concrete workflow requires them.

Operation records need to be privacy-minimal by default. They may record project-facing Git identity and authenticated provider login but won't store local OS username or hostname unless the user explicitly opts in.

## Trust Model

Waystone needs to separate:
- valid signatures
- trusted identities
- authorised maintainers
- canonical project state

A valid signature only proves authorship. It does not prove that the event should affect the project.

## Authority Controls

The project needs to define which identities can perform authority-bearing actions.

Examples:
- maintainers can close issues
- maintainers can reopen issues
- trusted contributors may label issues when local policy grants that authority
- anyone may submit comment events, subject to local policy

Events outside the author's authority should be retained as untrusted or rejected but not silently accepted.

## Import Risks

Imports may contain:
- fake authors
- conflicting timestamps
- large bodies
- invalid event chains
- misleading external URLs
- hostile Markdown
- malformed JSON
- duplicate IDs

Importers should validate structure, enforce size limits and preserve source provenance.

## Mitigations

Initial mitigations:
- deterministic import output
- source provenance for imported records
- signed operation records and source manifests
- explicit project trust policy
- maintainer allowlist
- untrusted-event quarantine
- deterministic projection
- size limits for records
- attachment exclusion by default
- no code execution on import
- no CI execution in v0
- no automatic webhook execution
- no global discovery network

## Non-Goals

Waystone v0 does not attempt to solve:
- global identity
- harassment moderation across projects
- spam prevention across a public network
- encrypted private issue tracking
- hosted multi-tenant abuse controls
- legal takedown workflows

Those concerns become relevant only if Waystone later grows hosted or networked components.

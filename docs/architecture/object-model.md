# Object Model

Waystone v0 models imported project history, source-scoped local issue records and local issue lifecycle events as portable ledger records.

## Imported Record Types

The first ledger supports:
- project
- external identity
- issue
- comment
- pull request
- review comment
- label
- milestone
- release
- source manifest
- operation record

Every imported record should keep enough source context to remain useful without contacting the source forge again.

Common fields:
- Waystone ID
- source system
- source ID
- original URL
- author identity where available
- created timestamp
- updated timestamp where available
- source fields needed for local display and projection

## Project

A project identifies a repository-level source namespace.

Candidate fields:
- source system
- owner
- repository name
- repository URL
- default branch where available
- import timestamp

Project records must not embed credentials or mutable platform session state.

## Source Namespace

A source namespace identifies where a record belongs:
```text
<system>:<owner>/<repo>
```

Examples:
- `github:steadytao/waymark`
- `gitlab:steadytao/waymark`
- `forgejo:steadytao/waymark`
- `gitea:steadytao/waymark`
- `waystone:steadytao/waymark`

`waystone:` is reserved for repo-scoped local Waystone records. It is not a remote forge namespace.

Display numbers are source-local. An issue, pull request or milestone number must be interpreted with its source namespace. This avoids collisions when a ledger contains GitHub issues, GitLab merge requests, Forgejo or Gitea records, local Waystone records or manually imported history with overlapping numbers.

## External Identity

An external identity represents an imported author from a forge.

Candidate fields:
- source system
- source account ID
- login or username
- display name where available
- profile URL

External identities are evidence from the source forge. They are not local signing identities and they do not grant authority.

## Issue

An issue is imported project discussion.

Candidate fields:
- number
- title
- body
- state
- author
- labels
- milestone
- created timestamp
- updated timestamp
- closed timestamp where available
- original URL

## Comment

A comment belongs to an issue, pull request or review thread.

Candidate fields:
- source ID
- parent object reference
- author
- body
- created timestamp
- updated timestamp where available
- original URL

Imported comment edit state should be preserved where the source exposes it.

## Pull Request

A pull request is imported as project history, not as a live merge mechanism.

Candidate fields:
- number
- title
- body
- state
- author
- base branch
- head branch
- labels
- milestone
- created timestamp
- updated timestamp
- closed timestamp where available
- merged timestamp where available
- original URL

Waystone won't recreate a forge's full pull request or merge request workflow in v0.

## Review Comment

A review comment preserves imported code-review discussion.

Candidate fields:
- source ID
- pull request number
- author
- body
- file path where available
- line or position where available
- created timestamp
- original URL

Line mappings may become stale after repository history changes. Waystone preserves the source data without pretending it can always re-anchor comments perfectly.

## Label, Milestone And Release

Labels and milestones are supporting project metadata. Releases preserve published release records.

Release candidate fields:
- tag name
- name
- body
- author
- created timestamp
- published timestamp
- draft or prerelease flags
- original URL

## Local Events And Future Event Model

Waystone already stores local issue lifecycle events for created, edited, labelled, unlabelled, commented, closed and reopened issue history. These events make local history inspectable beside imported forge records.

The longer-term direction is a broader signed append-only event model for more local collaboration state.

Example event:
```json
{
  "version": "waystone.event.v1",
  "id": "evt_...",
  "object": "issue_...",
  "type": "issue.comment_added",
  "author": "key_...",
  "created_at": "2026-05-01T00:00:00Z",
  "body": {},
  "parents": [],
  "signature": {}
}
```

This shape is descriptive, not a frozen wire format.

## Authorship, Trust And Authority

Waystone must keep these concepts separate:
- authorship proves which key signed an event
- trust says whether the project recognises that identity
- authority says whether the event can affect canonical state

Examples:
- an untrusted identity may create a comment event
- a trusted contributor may add a label
- a maintainer may close an issue
- a non-maintainer closure event may be retained but excluded from canonical projection

The projector must not treat every valid signature as project authority.

## Determinism

Given the same accepted records, events and trust policy, Waystone should produce the same projected state.

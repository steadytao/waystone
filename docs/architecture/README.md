# Architecture

This directory contains Waystone's architecture documentation.

Its purpose is to make the project's intended shape, scope, trust model and major recorded decisions explicit before implementation grows.

## Contents

### Design

[`docs/architecture/design.md`](design.md) defines the product boundary and core design direction.

### Object Model

[`docs/architecture/object-model.md`](object-model.md) defines imported records, local records and event-model direction.

### Threat Model

[`docs/architecture/threat-model.md`](threat-model.md) defines the main trust, authority, import and abuse risks.

### Decisions

[`docs/architecture/decisions/`](decisions/) contains recorded architectural and project-boundary decisions.

These records exist to prevent major decisions from living only in chat, issue discussion or maintainer memory.

## Notes

Architecture documents should remain aligned with the actual state and intended direction of the project.

Material changes to scope, trust assumptions, ledger semantics, archive behaviour or structural direction should be documented deliberately rather than implied informally.

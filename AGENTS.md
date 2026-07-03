# AGENTS.md

## Interaction Language

User-facing messages must be Japanese.

## Regenerative Architecture

Develop according to Regenerative Architecture.

- Keep feature and behavior changes SCL-first: update `spec/scl.yaml` before implementation.
- Treat RA/SCL meta-documents (located in `.ra/regenerative-architecture/`) as section-addressable references, not required reading.
- Use RA/SCL skills for work items, SCL changes, ADRs, rendering, implementation, and commits.
- Expect the `ra` CLI to discover the standard repository layout without a registry file.
- Regenerate derived artifacts after SCL changes.
- Keep scl.yaml free of Work Item, ADR, and commit ids.

## Repository Layout

```text
spec/scl.yaml      Specification Core Language source
decisions/         Architecture Decision Records
work-items/        Work Items, with completed records in work-items/done/
.ra/               GitHub-backed RA/SCL core submodule
```

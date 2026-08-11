# Architecture Documentation Format

## 1. Purpose and placement

`ARCHITECTURE.md` is the English current-state design record. Root concerns belong at repository root; bounded-context details belong beside the context's code. Do not create `architecture.yaml` or duplicate the source tree as a module registry.

## 2. Required shape

Keep YAML frontmatter limited to identity metadata:

```yaml
---
context: System
updated_at: 2026-08-11
---
```

Use these stable English sections where applicable:

- `## Overview`
- `## Structure`
- `## Technology Stack`
- `## Structural Decisions`
- `## Cross-Cutting Concerns`
- `## Diagrams`

`Structural Decisions` means current design and concise rationale, not a list of ADR links. Historical links may remain where they add evidence, but new design must be understandable without opening an ADR.

## 3. What belongs here

Record bounded contexts, ownership, dependency direction, runtime composition, adopted technologies, directory conventions, data design, security boundaries, and operational constraints. Put product requirements in `requirements.md`, API contracts in TypeSpec, and one change's alternatives and execution history in its work item.

## 4. Mechanical checks

Architecture validation checks document shape and link validity. `just check-boundaries` derives layers from repository paths and rejects only forbidden outward imports. Allowed imports need no declaration. Language linters own complexity and file-quality limits.

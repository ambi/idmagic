# Specification Format

## 1. Layout

```text
spec/
  main.tsp
  tspconfig.yaml
  requirements.md
  idmagic.openapi.baseline.json
  contexts/<context>/
    models.tsp
    main.tsp
    requirements.md
```

`main.tsp` composes the TypeSpec program. `models.tsp` owns model declarations and `main.tsp` owns operations for a context. Generated files live under ignored `spec/generated/`.

## 2. TypeSpec scope

Use TypeSpec for models, constraints, API operations, HTTP routes, request and response shapes, status codes, error unions, deprecation metadata, and authentication mechanisms. Prefer standard TypeSpec libraries and emitters. Do not recreate these constructs in Markdown or a project-specific YAML dialect.

The `IdMagic.Contract` namespace preserves released wire model names across context-owned source files. Context files remain the ownership boundary.

## 3. Requirements Markdown scope

Use the root or owning context's `requirements.md` for:

- normative requirements and acceptance scenarios;
- glossary terms;
- adopted, optional, or excluded standards requirements;
- state-machine transition tables;
- cross-context product behavior that is not an API type.

Requirement headings use `### REQ-<CONTEXT>-NNN: <title>`. IDs are immutable once referenced. State tables use exactly:

```markdown
| From | Event | Guard | To | Effects |
|---|---|---|---|---|
```

## 4. Authorization

TypeSpec records whether an API uses bearer authentication or is public. Fine-grained authorization policy is outside the specification language for now and remains executable application behavior with tests. Do not add a new authorization DSL. Cedar adoption is a future evaluation, not a predetermined choice.

## 5. Validation

- `just check-spec` compiles TypeSpec and validates requirement IDs and transition tables.
- `just check-api-compat` compares generated OpenAPI with the released baseline.
- `just spec-render` regenerates ignored standard artifacts.

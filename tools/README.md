# Repository tools

This workspace contains deterministic tooling for the repository:

- `workspace` discovers TypeSpec, canonical specification documents, work items, and conventionally located
  OpenAPI artifacts without a product-name registry.
- `check` validates Markdown frontmatter and the remaining YAML formats, resolves the references a
  work item makes into the specification, rejects forbidden dependency directions, guards the command
  map against workflow steps calling recipes that no longer exist, and derives the normative diff
  between a git ref and the working tree.
- `check-api-compat` compares the TypeSpec-generated OpenAPI contract with the released baseline.
- `generate-contract` derives the small Go runtime operation catalog from TypeSpec OpenAPI.
- `render-spec-docs` builds the multi-page specification site, derived Mermaid diagrams, Swagger UI API
  reference, searchable TypeSpec model catalog, and the traceability view from scenario identifiers
  cited in code, tests, and work items, with local assets.

The tools derive application-specific values from canonical sources and standard paths. TypeSpec ownership
comes from compiler source locations, OpenAPI filenames are discovered under `spec/`, and Go import roots
come from `go.mod`; tool source must not embed an application name or module path.

Run all routine operations from the repository root through `just`.

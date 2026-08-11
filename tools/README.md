# Repository tools

This workspace contains deterministic tooling for the repository:

- `workspace` discovers TypeSpec, canonical specification documents, work items, and conventionally located
  OpenAPI artifacts without a product-name registry.
- `check` validates Markdown frontmatter and the remaining YAML formats.
- `check-api-compat` compares the TypeSpec-generated OpenAPI contract with the released baseline.
- `generate-contract` derives the small Go runtime operation catalog from TypeSpec OpenAPI.
- `render-spec-docs` builds the multi-page specification site, derived Mermaid diagrams, Swagger UI API
  reference, and searchable TypeSpec model catalog with local assets.

The tools derive application-specific values from canonical sources and standard paths. TypeSpec ownership
comes from compiler source locations, OpenAPI filenames are discovered under `spec/`, and Go import roots
come from `go.mod`; tool source must not embed an application name or module path.

Run all routine operations from the repository root through `just`.

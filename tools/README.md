# Repository tools

This workspace contains deterministic tooling for the repository:

- `workspace` discovers TypeSpec, canonical specification documents, and work items for repository checks.
- `check` validates Markdown frontmatter and the remaining YAML formats.
- `check-api-compat` compares the TypeSpec-generated OpenAPI contract with the released baseline.
- `generate-contract` derives the small Go runtime operation catalog from TypeSpec OpenAPI.
- `render-spec-docs` builds the navigation-linked specification and API HTML view.

Run all routine operations from the repository root through `just`.

# Repository tools

This workspace contains deterministic tooling for the repository:

- `ra` discovers and validates TypeSpec, requirements, architecture records, and work items.
- `check` validates Markdown frontmatter and the remaining YAML formats.
- `check-api-compat` compares the TypeSpec-generated OpenAPI contract with the released baseline.
- `generate-contract` derives the small Go runtime operation catalog from TypeSpec OpenAPI.

Run all routine operations from the repository root through `just`.

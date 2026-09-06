# Repository tools

This workspace contains deterministic tooling for the repository:

- `workspace` discovers TypeSpec, canonical specification documents, work items, and conventionally located
  OpenAPI artifacts without a product-name registry.
- `check` validates Markdown frontmatter and the remaining YAML formats, resolves the references a
  work item makes into the specification, rejects forbidden dependency directions, guards the command
  map against workflow steps calling tasks that no longer exist, and derives the normative diff
  between a git ref and the working tree.
- `check-api-compat` compares the TypeSpec-generated OpenAPI contract with the released baseline.
- `brief` answers what one work item should be read from: the declaration of each requirement it names, where
  TypeSpec declares each symbol, the tests and implementation that already name the requirement, the recent
  history of the same context, and a draft `initial_context`.
- `changed-packages` selects the Go packages the working tree changed together with everything that compiles
  them in, so the narrow gate covers what the change can break without running the whole module.
- `task-timing` runs the members of a verification suite one at a time and prints what each cost.
- `generate-contract` derives the small Go runtime operation catalog from TypeSpec OpenAPI.
- `render-spec-docs` builds the multi-page specification site, derived Mermaid diagrams, Swagger UI API
  reference, searchable TypeSpec model catalog, and the traceability view from scenario identifiers
  cited in code, tests, and work items, with local assets.

The tools derive application-specific values from canonical sources and standard paths. TypeSpec ownership
comes from compiler source locations, OpenAPI filenames are discovered under `spec/`, and Go import roots
come from `go.mod`; tool source must not embed an application name or module path.

Run all routine operations from the repository root through `mise run`.

`mise.toml` pins Go, Bun, golangci-lint, sqlc, and psqldef. `mise install` installs missing versions into mise's
shared user-level store. Renovate reads `mise.toml` with its built-in manager and proposes upgrades; CI uses the
same tool definitions. Operational PostgreSQL client tools are supplied by the backup or restore execution
environment rather than managed by this development toolchain.

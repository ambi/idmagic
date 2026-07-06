# RA/SCL tools

This workspace contains deterministic tools used by Regenerative Architecture
repositories:

- `yaml-check`: YAML linting, schema validation, context-map checks, and record id checks.
- `scl-to-html`: HTML views for SCL, ADRs, and Work Items.
- `scl-to-jsonschema`: JSON Schema generation from SCL models.
- `scl-to-openapi`: OpenAPI generation from SCL HTTP interfaces.
- `ra`: workspace discovery, validation, initialization, and rendering.

Run from the repository root through `just`, or directly from this directory:

```bash
bun install --frozen-lockfile
bun run typecheck
bun run lint
bun test
bun run ra -- --help
bun run yaml-check:all
bun run scl-render:workspace
```

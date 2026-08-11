# check

A deterministic parser, raw-text linter, and optional JSON Schema validator for
the repository's remaining YAML and Markdown-frontmatter records.

## Usage

```bash
bun run check/src/main.ts <file-or-glob>...
bun run check/src/main.ts --schema=<name> <file-or-glob>...
bun run check/src/main.ts --list-schemas
```

Packaged schemas are `work-item` and `architecture-doc`. TypeSpec is validated
by the TypeSpec compiler, and requirements Markdown has a dedicated checker.

Exit code 0 means every input is valid, 1 means findings were reported, and 2
means the command usage was invalid.

# AGENTS.md / CLAUDE.md

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
- If bounded contexts, global directory structures, or core architecture rules are added or modified, synchronize the map and details in [ARCHITECTURE.md](file:///Users/tn/src/idmagic/ARCHITECTURE.md).

## Commands via just

The `justfile` is the single command map for this repo. Run every basic
command — verify, build, test, lint, format, dev servers, demos, codegen —
through its `just` recipe, never by invoking the underlying tool (`bun`, `go`,
`golangci-lint`, `docker`, a `*.sh` script, …) directly. This applies to running
checks and to any command written into a Work Item's `verification` list.

- Verify: `just verify` (whole app), `just verify-ui`
  (format-check / lint / typecheck / build), `just verify-go`.
  Do not write `bun --cwd ui typecheck` / `lint` / `build` — use the recipe.
- Build / test: `just build-go`, `just build-ui`, `just test-go`,
  `just test-ui-e2e`.
- Dev / demo: `just dev` (API + UI stack), `just dev-api`, `just dev-ui`,
  `just demo`.
- SCL / Work Item YAML: `just yaml-check`, `just yaml-check-work-items`,
  `just check-ids`, `just scl-render`.
- Run `just --list` to discover the recipe before reaching for a raw tool. If a
  common command has no recipe yet, add one to the `justfile` instead of running
  it ad hoc (as was done for `./dev.sh` → `just dev`, `./demo.sh` → `just demo`).

## Repository Layout

```text
spec/scl.yaml      Specification Core Language source
decisions/         Architecture Decision Records
work-items/        Work Items, with completed records in work-items/done/
.ra/               GitHub-backed RA/SCL core submodule
```

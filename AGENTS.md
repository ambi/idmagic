# AGENTS.md / CLAUDE.md

## Rules

- Status updates, explanations, questions, summaries, and final responses of AI coding agents must be Japanese.
- The main `README.md` file must be written in English only.
- `ARCHITECTURE.md` (root and per-context design records) must be written in English only.
- Develop according to Regenerative Architecture.
  - Keep feature and behavior changes SCL-first: update `spec/scl.yaml` before implementation.
  - Treat RA/SCL meta-documents ([REGENERATIVE_ARCHITECTURE.md](file:///Users/tn/src/idmagic/REGENERATIVE_ARCHITECTURE.md), [SPECIFICATION_CORE_LANGUAGE.md](file:///Users/tn/src/idmagic/SPECIFICATION_CORE_LANGUAGE.md)) as section-addressable references, not required reading.
  - Use RA/SCL skills for work items, SCL changes, ADRs, rendering, implementation, and commits.
  - Expect the `ra` CLI to discover the standard repository layout without a registry file.
  - Regenerate derived artifacts after SCL changes.
  - If bounded contexts, global directory structures, or core architecture rules are added or modified, synchronize the design record ([ARCHITECTURE.md](file:///Users/tn/src/idmagic/ARCHITECTURE.md)) and the ledger beside it (`architecture.yaml`).
  - Write design in `ARCHITECTURE.md`, not in an ADR. Open an ADR only when a real fork existed and a rejected option is on record (ADR-143).

## Commands via just

The `justfile` is the single command map for this repo. Run every basic command — verify, build, test, lint, format, dev servers, demos, codegen — through its `just` recipe, never by invoking the underlying tool (`bun`, `go`, `golangci-lint`, `docker`, a `*.sh` script, …) directly.

- Run `just --list` to discover the recipe before reaching for a raw tool. If a common command has no recipe yet, add one to the `justfile` instead of running it ad hoc.

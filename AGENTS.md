# AGENTS.md / CLAUDE.md

## Rules

- Status updates, explanations, questions, summaries, and final responses of AI coding agents must be Japanese.
- The main `README.md` file must be written in English only.
- `ARCHITECTURE.md` (root and per-context design records) must be written in English only.
- Develop according to Regenerative Architecture.
  - Keep feature and behavior changes specification-first.
  - Put models, API interfaces, and authentication mechanisms in `spec/**/*.tsp`.
  - Put requirements, scenarios, glossary, standards, and language-independent state-transition tables in `spec/**/requirements.md`.
  - Use stable `REQ-<CONTEXT>-NNN` requirement IDs and TypeSpec symbols in work-item references.
  - Treat [REGENERATIVE_ARCHITECTURE.md](REGENERATIVE_ARCHITECTURE.md), [SPECIFICATION_FORMAT.md](SPECIFICATION_FORMAT.md), and [ARCHITECTURE_FORMAT.md](ARCHITECTURE_FORMAT.md) as section-addressable references, not required reading.
  - Expect the `ra` CLI to discover the standard repository layout without a registry file.
  - Regenerate untracked TypeSpec artifacts after specification changes.
  - If bounded contexts, global directory structures, adopted technologies, or core architecture rules change, synchronize the relevant `ARCHITECTURE.md`.
  - Do not create new ADRs. Put durable current design and rationale in `ARCHITECTURE.md`; put change-specific analysis, alternatives, and implementation history in the work item. Existing ADRs are read-only history.
  - Do not add `architecture.yaml`. Architectural checks infer structure from paths and reject forbidden dependencies only.

## Commands via just

The `justfile` is the single command map for this repo. Run every basic command — verify, build, test, lint, format, dev servers, demos, codegen — through its `just` recipe, never by invoking the underlying tool (`bun`, `go`, `golangci-lint`, `docker`, a `*.sh` script, …) directly.

- Run `just --list` to discover the recipe before reaching for a raw tool. If a common command has no recipe yet, add one to the `justfile` instead of running it ad hoc.

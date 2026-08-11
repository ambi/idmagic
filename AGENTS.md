# AGENTS.md / CLAUDE.md

## Rules

- Status updates, explanations, questions, summaries, and final responses of AI coding agents must be Japanese.
- The main `README.md` file must be written in English only.
- Use the specification-first development workflow.
  - Keep feature and behavior changes specification-first.
  - Put models, API interfaces, and authentication mechanisms in `spec/**/*.tsp`.
  - Put overview, glossary, standards, language-independent state transitions, current design,
    and scenarios in the owning `spec/**/SPECIFICATION.md`.
  - Use stable `REQ-<CONTEXT>-NNN` normative scenario IDs and TypeSpec symbols in work-item references.
  - Treat [DEVELOPMENT.md](DEVELOPMENT.md) and
    [SPECIFICATION_FORMAT.md](SPECIFICATION_FORMAT.md) as section-addressable references, not required reading.
  - Expect repository tools to discover the standard layout without a registry file; use `just` recipes
    rather than a methodology-specific CLI.
  - Regenerate untracked TypeSpec and HTML artifacts after specification changes.
  - If bounded contexts, global directory structures, adopted technologies, or core design rules change,
    synchronize the owning `SPECIFICATION.md` Design section.
  - Put durable current design and rationale in `SPECIFICATION.md`;
    put change-specific analysis, alternatives, and implementation history in the work item.
  - Do not add architecture ledgers. Boundary checks infer structure from paths and reject forbidden dependencies only.

## Commands via just

The `justfile` is the single command map for this repo. Run every basic command — verify, build, test, lint, format, dev servers, demos, codegen — through its `just` recipe, never by invoking the underlying tool (`bun`, `go`, `golangci-lint`, `docker`, a `*.sh` script, …) directly.

- Run `just --list` to discover the recipe before reaching for a raw tool. If a common command has no recipe yet, add one to the `justfile` instead of running it ad hoc.

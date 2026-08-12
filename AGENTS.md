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
- `golangci-lint` already enables the `govet` linter and formats with `gofumpt`+`goimports`, so `go vet`,
  `gofmt`, and `gofumpt` are redundant: use `just lint-go` / `just format-go` / `just test-go`.
  Do not hand-prefix `GOCACHE=` — the default build cache is correct and faster.

## Tooling

Reach for the purpose-built CLI before writing a script. Needing more than one command is a reason to
learn the tool's flags, not to open an editor — a throwaway script is never the shortest path to a
value that already lives in a structured file.

| Task | Use | Not |
|---|---|---|
| Query/extract JSON | `jq` | throwaway `python3 -c` / `bun -e` |
| Explore JSON of unknown shape | `gron file.json \| rg pattern` | reading the whole file |
| Query/extract YAML | `yq` | `grep` against indentation |
| Search file contents | `rg` | `grep -r`, `find … -exec grep` |
| Find files by name | `fd` | `find` |
| Search/rewrite code by structure | `ast-grep` | multi-file regex, the deprecated `sg` alias |
| Bulk literal replace across files | `sd` | `sed -i` |
| Read an HTML page, extract from the web | `ax` | `curl` + a parsing script |
| GitHub PRs, issues, API | `gh` | `curl` against api.github.com |

Run `just setup-cli-tools` if any of these are missing.

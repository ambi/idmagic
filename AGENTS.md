# AGENTS.md / CLAUDE.md

## Language

| What | Language |
|---|---|
| Repository-root `README.md`, headings included | Japanese |
| `CONFIGURATION.md`, `DEVELOPMENT.md`, `SPECIFICATION_FORMAT.md`, `WORK_ITEM_FORMAT.md`, this file | English |
| `DOCUMENTATION_GUIDE.md`, headings included | Japanese |
| `spec/**/*.tsp`, doc comments included | English |
| Prose in the canonical documents under `spec/`, `spec/README.md` and `spec/contexts/*/README.md` included | Japanese |
| `work-items/**` | Japanese |
| READMEs below the repository root except `spec/**` and `tools/**`, headings included | Japanese |
| `infra/runbooks/*`, headings included | Japanese |
| `tools/**` | English |
| Go and TypeScript comments | Japanese |
| Go and TypeScript identifiers, database columns, event names, error codes, scopes | English |
| `REQ-<CONTEXT>-NNN`, TypeSpec symbols, section headings, table headers, frontmatter keys | English |
| Headings and file names of the canonical documents under `spec/`, `README.md` included | English |
| API error messages, log messages, CLI help | English |
| User-facing UI text | Localized; `ja` and `en` both live in the `*.i18n.ts` dictionaries |
| Commit messages | English |
| Status updates, explanations, questions, summaries, and final responses of AI coding agents | Japanese |

For anything not listed: English if it leaves the repository or appears at runtime without being
localized, because such text can never be swapped for a translation. Japanese otherwise.

Write Japanese prose as natural technical Japanese, using established translations or transliterations for
ordinary vocabulary. Keep Latin spelling only when the exact original spelling is semantically required — an
identifier, literal value, path, command, product or protocol name, acronym — or when no settled Japanese
rendering exists. Familiarity among engineers, domain jargon, capitalization, or easier correspondence with an
English source does not make the original spelling necessary. Put exact code and configuration tokens in
backticks; otherwise translate or transliterate them. Review the whole sentence as Japanese rather than
substituting Japanese predicates around English nouns. When several Japanese renderings are reasonable,
follow the established repository usage consistently instead of choosing independently in each document.
Choose wording from the concept's role in context, never from a global word-for-word substitution table, and
do not collapse distinct source concepts into one Japanese term. Treat recognized named technical patterns and
standard-defined terms as terms of art, and retain their original names when translation would obscure the
concept being referenced.

Do not say the same thing twice in two languages. A Japanese paragraph and an English paragraph carrying
the same content are two documents drifting apart, not one bilingual document.

None of this is checked by tooling.

## Formatting

Markdown has no column limit. Japanese prose in particular is one paragraph per line, however long: a
newline inside a paragraph renders as a space, so a break between two Japanese characters inserts a
space that does not belong there. Tables, code blocks, and generated files keep whatever width their
own format requires.

Source files wrap at about 150 columns. Do not carry the old 80-column habit into them.

## Rules

- Use the specification-first development workflow.
  - Keep feature and behavior changes specification-first.
  - Put models, API interfaces, and authentication mechanisms in `spec/**/*.tsp`.
  - Put the boundary declaration, glossary, standards, language-independent state transitions,
    decisions, mechanism, and scenarios in the canonical file under `spec/` whose name says what that
    kind of content is. The file name is the division; there is no single per-context document.
  - Use stable `REQ-<CONTEXT>-NNN` normative scenario IDs and TypeSpec symbols in work-item references.
  - Treat these as section-addressable references, not required reading:
    [DEVELOPMENT.md](DEVELOPMENT.md) for the stage/skill/gate loop and the verification ladder,
    [SPECIFICATION_FORMAT.md](SPECIFICATION_FORMAT.md) for specification documents, and
    [WORK_ITEM_FORMAT.md](WORK_ITEM_FORMAT.md) for work items.
  - [DOCUMENTATION_GUIDE.md](DOCUMENTATION_GUIDE.md) describes the document system in general, without
    depending on this repository. The three documents above are what this repository enforces; where they
    differ, follow those.
  - Expect repository tools to discover the standard layout without a registry file; use `mise` tasks
    rather than a methodology-specific CLI.
  - Regenerate untracked TypeSpec and HTML artifacts after specification changes.
  - If bounded contexts, global directory structures, adopted technologies, or core design rules change,
    synchronize `spec/README.md` and `spec/structure.md`.
  - Put durable decisions in `decisions.md` and durable mechanism in `internals.md`;
    put change-specific analysis, alternatives, and implementation history in the work item.
  - Do not add architecture ledgers. Boundary checks infer structure from paths and reject forbidden dependencies only.

## Commands via mise

`mise.toml` is the single tool-version, environment, and command map for this repo. Install the pinned tools with `mise install`. Run every basic command — verify, build, test, lint, format, dev servers, demos, codegen — through `mise run <task>`, never by invoking the underlying tool (`bun`, `go`, `golangci-lint`, `docker`, a `*.sh` script, …) directly.

- Run `mise tasks` to discover the task before reaching for a raw tool. If a common command has no task yet, add one to `mise.toml` instead of running it ad hoc.
- `golangci-lint` already enables the `govet` linter and formats with `gofumpt`+`goimports`, so `go vet`,
  `gofmt`, and `gofumpt` are redundant: use `mise run lint-go` / `mise run format-go` / `mise run test-go`.
  Do not hand-prefix `GOCACHE=` — the default build cache is correct and faster.

## Tooling

Reach for the purpose-built tool before writing a script — when reading and editing files as much as
when extracting values. Needing more than one command is a reason to learn the tool's flags, not to
open an editor: a throwaway script is never the shortest path to a value that already lives in a
structured file, nor to an edit an editing tool already makes directly.

The first two rows are the ones every task touches, so they come first: an agent with file tools
reads and edits through those tools, and reaches for a shell only for what they cannot do.

| Task | Use | Not |
|---|---|---|
| Read a file, or part of one | the Read tool (`offset`/`limit`) | `cat`, `head`, `tail`, `sed -n 'N,Mp'` |
| Edit code | the Edit tool, or `sd` for one literal string across files | `sed -i`, a `python3` / `bun` heredoc rewriting files |
| Query/extract JSON | `jq` | throwaway `python3 -c` / `bun -e` |
| Explore JSON of unknown shape | `gron file.json \| rg pattern` | reading the whole file |
| Query/extract YAML | `yq` | `grep` against indentation |
| Search file contents | `rg` | `grep -r`, `find … -exec grep` |
| Find files by name | `fd` | `find` |
| Search/rewrite code by structure | `ast-grep` | multi-file regex, the deprecated `sg` alias |
| Bulk literal replace across files | `sd` | `sed -i` |
| Read an HTML page, extract from the web | `ax` | `curl` + a parsing script |
| GitHub PRs, issues, API | `gh` | `curl` against api.github.com |

Run `mise run setup-cli-tools` if any of these are missing.

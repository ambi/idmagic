# check

A deterministic parser, raw-text linter, and optional JSON Schema validator for
the repository's remaining YAML and Markdown-frontmatter records.

## Usage

```bash
bun run check/src/main.ts <file-or-glob>...
bun run check/src/main.ts --schema=<name> <file-or-glob>...
bun run check/src/main.ts --list-schemas
```

The packaged schema is `work-item`. TypeSpec is validated by the TypeSpec compiler,
and the canonical specification documents have a dedicated checker.

Exit code 0 means every input is valid, 1 means findings were reported, and 2
means the command usage was invalid.

## Coverage debt

Three ratchets record normative ids that no test names yet. Each list only
shrinks: an id on one that has grown a test has to come off, and an id declared
from now on is not admitted.

| File | Holds | Checked by |
|---|---|---|
| `security-refusal-debt.json` | Scenarios that declare a refusal | `check-security-controls` |
| `scenario-coverage-debt.json` | Every other live scenario | `check-spec` |
| `standards-coverage-debt.json` | Rows of any `standards.md` | `check-spec` |

The refusal list stays its own list because "a security control's refusal is
untested" is a louder fact than "a behavior is untested", and a list that mixes
the two loses the reason anyone reads it. The scenario list therefore never
repeats an id the refusal list already holds — the two are disjoint, not merely
separate, and `check-spec` says so when they overlap. Entries on the two
`check-spec` lists carry a reason, so the ones added later stay tellable from
the ones that were there when the check arrived.

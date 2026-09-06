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

Two ratchets record normative ids that no test names yet. Each list only
shrinks: an id on one that has grown a test has to come off, and an id declared
from now on is not admitted.

| File | Holds | Admission set | Checked by |
|---|---|---|---|
| `example-coverage-debt.json` | Executable scenario examples | `example-coverage-debt-baseline.json` | `check-spec` |
| `standards-coverage-debt.json` | Rows of any `standards.md` | `standards-coverage-debt-baseline.json` | `check-spec` |

The split is by where the id is declared, and that is the only split. Every
entry carries a reason, so the ones added later stay tellable from the ones that
were there when the check arrived.

The admission set is what makes "only shrinks" a checked property rather than a
claim. It is the ledger's contents on the day the ratchet arrived; removing an
entry from the ledger does not remove its id from the set, and an id that is not
in the set cannot be admitted at all. Without one, a row added to `standards.md`
today could be listed as untested and the check would pass, so burning the
ledger down and growing it would have been a race.

A third file, `security-refusal-debt.json`, used to hold the scenarios that
declare a refusal, on the ground that "a security control's refusal is untested"
is a louder fact than "a behavior is untested". It is louder, and it is still
reported: `mise run report-coverage-debt` weights each entry by whether its
scenario names an error type the contract answers a 403 with on a state-changing
operation. What the separate file cost was a second implementation of this
comparison, a disjointness invariant between the two lists, and a fifteen-word
prose classifier deciding which file an id belonged in — a classifier that
matched condition clauses and audit-field names, and whose result changed
nothing else in the system. See wi-490.

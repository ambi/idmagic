/**
 * Check that every normative id the specification declares is reached by a test
 * that names it.
 *
 * `standards.md` says of its own rows that each one has a corresponding test,
 * and `scenarios.md` gives every behavior an id so a test can point at it. Both
 * claims were unchecked: `WCAG22-KEYBOARD`, `GDPR-ERASURE`, and
 * `GDPR-CONSENT-WITHDRAWAL` appeared nowhere in `backend/` or `frontend/`, and
 * 191 of 306 live scenarios were named by no test at all.
 *
 * Declared ids, the ids a test names, and an allowed debt list, compared three
 * ways. Each debt entry carries a reason: a list this size stops being read the
 * moment its entries are indistinguishable from one another, and then it stops
 * shrinking.
 *
 * This is the only implementation of the comparison. A second one lived in
 * security-controls.ts as R3, over the ids a prose classifier read as refusals,
 * and the split cost two debt files plus an invariant keeping them disjoint. It
 * bought nothing: the rule on both sides was this one. Whether an id declares a
 * refusal is a fact worth reporting, so report-coverage-debt derives it from
 * the contract; it is not a reason to check the id differently.
 */

export type DeclaredId = { id: string; path: string }

export type DebtEntry = { id: string; reason: string }

export type CoverageFinding = { path: string; message: string }

export type CoverageInput = {
  declared: readonly DeclaredId[]
  /** The ids some test names. */
  cited: ReadonlySet<string>
  debt: readonly DebtEntry[]
  debtPath: string
}

function escapeForPattern(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * The declared ids named anywhere in the given test sources.
 *
 * The declared ids are the pattern rather than something a shape recognizes on
 * its own. A shape has to be guessed, and the guess was wrong: upper-case
 * segments joined by hyphens reads `REQ-OAUTH2-001` and `NIST63B4-PASSWORD-
 * MINIMUM` but not `SAML2Core-BearerAssertion` or `WSFed-PassiveSignIn`, so
 * thirteen SAML and WS-Federation rows could never have been credited with a
 * test however plainly one named them. Searching for what the specification
 * actually declares cannot be wrong about the ids that exist.
 *
 * Longest first, so `REQ-DEMO-001` cannot claim a mention of `REQ-DEMO-0011`,
 * and neither end of a match may sit against another id character.
 */
export function citedNormativeIds(
  sources: Iterable<string>,
  declared: Iterable<string>,
): Set<string> {
  const ids = [...new Set(declared)].sort((left, right) => right.length - left.length)
  const cited = new Set<string>()
  if (ids.length === 0) return cited
  const pattern = new RegExp(
    `(?<![A-Za-z0-9-])(?:${ids.map(escapeForPattern).join('|')})(?![A-Za-z0-9-])`,
    'g',
  )
  for (const source of sources) {
    for (const match of source.matchAll(pattern)) cited.add(match[0])
  }
  return cited
}

export function checkNormativeCoverage(input: CoverageInput): CoverageFinding[] {
  const { declared, cited, debt, debtPath } = input
  const findings: CoverageFinding[] = []
  const listed = new Set(debt.map((entry) => entry.id))

  for (const declaration of declared) {
    if (cited.has(declaration.id)) continue
    if (listed.has(declaration.id)) continue
    findings.push({
      path: declaration.path,
      message:
        `${declaration.id} is declared, but no test names it. ` +
        `Cite the id from the test that exercises it, or list it in ${debtPath} with a reason.`,
    })
  }

  const declaredIds = new Set(declared.map((declaration) => declaration.id))
  const seen = new Set<string>()
  let previous: string | undefined
  for (const entry of debt) {
    if (seen.has(entry.id)) {
      findings.push({
        path: debtPath,
        message: `${entry.id} is listed twice. Keep one entry per id.`,
      })
      continue
    }
    seen.add(entry.id)
    if (previous !== undefined && entry.id < previous) {
      findings.push({
        path: debtPath,
        message:
          `${entry.id} is listed after ${previous}. ` +
          'Keep the list in id order so its diffs stay readable.',
      })
    }
    previous = entry.id
    if (entry.reason.trim().length === 0) {
      findings.push({
        path: debtPath,
        message: `${entry.id} is listed without a reason. State why it has no test yet.`,
      })
    }
    if (!declaredIds.has(entry.id)) {
      findings.push({
        path: debtPath,
        message: `${entry.id} is listed as untested but nothing declares it any more. Remove it.`,
      })
      continue
    }
    if (cited.has(entry.id)) {
      findings.push({
        path: debtPath,
        message: `${entry.id} now has a test that names it. Remove it from the list; the list only shrinks.`,
      })
    }
  }
  return findings
}

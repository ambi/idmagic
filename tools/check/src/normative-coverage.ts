/**
 * Check that every normative id the specification declares is reached by a test
 * that names it.
 *
 * `standards.md` says of its own rows that each one has a corresponding test,
 * and `scenarios.feature.md` gives every behavior an id so a test can point at it. Both
 * claims were unchecked: `WCAG22-KEYBOARD`, `GDPR-ERASURE`, and
 * `GDPR-CONSENT-WITHDRAWAL` appeared nowhere in `backend/` or `frontend/`, and
 * 191 of 306 live scenarios were named by no test at all.
 *
 * Declared ids against the ids a test names, and nothing else. There is no
 * list of ids allowed to go untested. The standards ledger was emptied and
 * deleted by wi-495 and the examples ledger by wi-496, and with them went the
 * exemption itself: an id is named by a test or the check fails, so recreating
 * a ledger file under its old name admits nothing.
 *
 * This is the only implementation of the comparison. A second one lived in
 * security-controls.ts as R3, over the ids a prose classifier read as refusals.
 * It bought nothing: the rule on both sides was this one. Whether an id
 * declares a refusal is a fact worth reporting, not a reason to check the id
 * differently. See wi-490.
 */

export type DeclaredId = { id: string; path: string }

export type CoverageFinding = { path: string; message: string }

export type CoverageInput = {
  declared: readonly DeclaredId[]
  /** The ids some test names. */
  cited: ReadonlySet<string>
}

function escapeForPattern(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * The declared ids the given test sources cite, and where each was cited.
 *
 * A test claims coverage in exactly one of two shapes, and nothing else counts.
 *
 *   1. A `//spec:covers` directive naming the ids, then a colon, then what the
 *      test fixes. A long list may end its line with a separator and continue
 *      on the next comment line.
 *   2. A string literal whose whole value is the id, which is how a
 *      table-driven test enumerates the examples one function covers.
 *
 * Prose never counts, and cannot: no sentence contains `//spec:covers` by
 * accident. Before wi-567 any occurrence of a declared id in a test file
 * counted, so writing "this id belongs to another record" in a comment silently
 * claimed the id — that happened three times in one session. The first fix was
 * a canonical comment shape (ids at the head, then a colon), and it worked, but
 * it needed five subtleties to accommodate the corpus: head position, the
 * colon, three separators, wrapped lists, and the `(optional)` adoption
 * qualifier. Writing that matcher went wrong twice, once by crediting every
 * hyphenated string literal in the tree and inflating coverage fourfold. A
 * directive has one rule, and `//go:generate` and `//nolint:` already make the
 * shape familiar.
 *
 * The declared ids are the pattern rather than something a shape recognizes on
 * its own. A shape has to be guessed, and the guess was wrong: upper-case
 * segments joined by hyphens reads `REQ-OAUTH2-001` and `NIST63B4-PASSWORD-
 * MINIMUM` but not `SAML2Core-BearerAssertion` or `WSFed-PassiveSignIn`, so
 * thirteen SAML and WS-Federation rows could never have been credited with a
 * test however plainly one named them. Searching for what the specification
 * actually declares cannot be wrong about the ids that exist.
 *
 * Longest first, so `REQ-DEMO-001` cannot claim a citation of `REQ-DEMO-0011`,
 * and neither end of a match may sit against another id character.
 */
export function citedNormativeIds(
  sources: Iterable<string>,
  declared: Iterable<string>,
): Set<string> {
  const cited = new Set<string>()
  const ids = [...new Set(declared)].sort((left, right) => right.length - left.length)
  if (ids.length === 0) return cited

  // 文字列リテラルの側は宣言済みの id だけを見る。ここに「id の形」を使うと、
  // `"Content-Type"` のようなリテラルまで cited へ入る。宣言済み id の判定は変わらない
  // ので検査は緑のままだが、報告する被覆件数だけが嘘になる (実測で 346 が 1407 になった)。
  // 正しい数字と見分けがつかない数字を出すほうが、落ちるより悪い。
  const declaredAlternation = ids.map(escapeForPattern).join('|')
  const wholeLiteral = new RegExp(`["'\`](${declaredAlternation})["'\`]`, 'g')
  const idOnly = new RegExp(`(?<![A-Za-z0-9-])(?:${declaredAlternation})(?![A-Za-z0-9-])`, 'g')

  const directive = /^\s*\/\/spec:covers\s+(.*)$/
  const commentBody = /^\s*\/\/\s*(.*)$/

  for (const source of sources) {
    const lines = source.split('\n')
    for (const [index, text] of lines.entries()) {
      for (const match of text.matchAll(wholeLiteral)) {
        if (match[1]) cited.add(match[1])
      }
      const declaredHere = text.match(directive)?.[1]
      if (declaredHere === undefined) continue
      // 長い並びは 1 行に収まらない。区切りで終わる行は次のコメント行へ続く。
      let list = declaredHere
      for (let next = index + 1; /[,、/]\s*$/.test(list) && next < lines.length; next += 1) {
        const body = (lines[next] ?? '').match(commentBody)
        if (!body) break
        list = `${list.trimEnd()} ${body[1]}`
      }
      // コロンから先は人へ向けた説明であり、id は前半にしか置けない。
      const head = list.split(/[:：]/)[0] ?? ''
      for (const match of head.matchAll(idOnly)) cited.add(match[0])
    }
  }
  return cited
}

export function checkNormativeCoverage(input: CoverageInput): CoverageFinding[] {
  const { declared, cited } = input
  return declared
    .filter((declaration) => !cited.has(declaration.id))
    .map((declaration) => ({
      path: declaration.path,
      message:
        `${declaration.id} is declared, but no test names it. ` +
        'Cite the id from the test that exercises it.',
    }))
}

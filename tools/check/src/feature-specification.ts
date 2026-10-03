/**
 * 機能仕様（新しい形式の機能ノードの `README.md` と章）が宣言する規則を読む。
 *
 * 規則は `### REQ-…` または `#### REQ-…` の見出しで宣言し、題名は ID に続けて書く。
 * 廃止した規則は、題名の末尾に `(superseded by REQ-…)` を付ける。例は同じ機能ノードの
 * `examples.feature.md` に置くので、ここで読むのは宣言と本文だけである。
 */

import type { SpecificationFinding } from './specification-doc.ts'

export type SpecificationRule = {
  id: string
  /** ID に続く題名。廃止の印を含めて書いたとおりに返す。 */
  title: string
  line: number
  supersededBy?: string
}

const DECLARATION = /^(#{1,6}) (REQ-[A-Z0-9-]+)(?::)?(?:\s+(.*))?$/
const SUPERSEDED = /\(superseded by (REQ-[A-Z0-9-]+)\)$/

/** 規則の宣言として読む見出しの深さ。操作の節（H2）の下の操作（H3）か、その下（H4）に置く。 */
const DECLARATION_LEVELS = new Set([3, 4])

export function specificationRules(source: string): SpecificationRule[] {
  const rules: SpecificationRule[] = []
  for (const [index, text] of source.split('\n').entries()) {
    const match = text.match(DECLARATION)
    if (!match?.[2] || !DECLARATION_LEVELS.has(match[1]?.length ?? 0)) continue
    const title = (match[3] ?? '').trim()
    rules.push({
      id: match[2],
      title,
      line: index + 1,
      supersededBy: title.match(SUPERSEDED)?.[1],
    })
  }
  return rules
}

/** 宣言の形を外れた見出しを拒否する。読み落とした規則は、どの検査からも見えなくなる。 */
export function validateSpecificationDeclarations(source: string): SpecificationFinding[] {
  const findings: SpecificationFinding[] = []
  const seen = new Map<string, number>()
  for (const [index, text] of source.split('\n').entries()) {
    const line = index + 1
    const legacy = text.match(/^#{2,6} Rule: (REQ-[A-Z0-9-]+)/)
    if (legacy) {
      findings.push({
        line,
        message: `${legacy[1]} must be declared as a "#### ${legacy[1]} <title>" heading, without "Rule:"`,
      })
      continue
    }
    const match = text.match(DECLARATION)
    if (!match?.[2]) continue
    if (!DECLARATION_LEVELS.has(match[1]?.length ?? 0)) {
      findings.push({ line, message: `${match[2]} must be declared at heading level 3 or 4` })
      continue
    }
    if (!(match[3] ?? '').trim()) {
      findings.push({ line, message: `${match[2]} must have a title` })
    }
    const previous = seen.get(match[2])
    if (previous !== undefined) {
      findings.push({ line, message: `duplicate scenario id ${match[2]}` })
    } else {
      seen.set(match[2], line)
    }
  }
  return findings
}

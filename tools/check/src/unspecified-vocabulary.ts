/**
 * 契約とコードが宣言する語彙のうち、どの要件にも現れないものを探す。
 *
 * TypeSpec が宣言するエラーコードと、コードが宣言するドメインイベントは、どちらも外部から
 * 観測できる結果である。要件のどこにも現れなければ、その結果を返す条件、発行する条件を
 * 誰も決めていない。検査は「現れるか」だけを見て、要件の書き方の正しさは判定しない。
 */

import { TRANSITION_HEADER } from './specification-doc.ts'

/** エラーコードと、その `@doc` が付く ProblemDetails のモデル名。モデル名で書いた要件も記載とみなす。 */
export type ErrorCodes = Map<string, string[]>

export type Vocabulary = { errors: ErrorCodes; events: string[] }

/** 語彙が現れてよい本文。要件の本文と、状態遷移の表の行である。 */
export type SpecifiedTexts = { requirements: string[]; transitions: string[] }

export type UnspecifiedTerm = { kind: 'error' | 'event'; name: string }

/** 導入時点の違反。減る方向にだけ変える。 */
export type VocabularyDebt = { errors: string[]; events: string[] }

const ERROR_CODE = /urn:idmagic:error:([a-z0-9_]+)/g
const DOCUMENTED_MODEL = /@doc\("((?:[^"\\]|\\.)*)"\)\s*model\s+([A-Za-z_][A-Za-z0-9_]*)/g
const EVENT_TYPE = /EventType\(\)\s+string\s*\{\s*return\s+"([A-Za-z_][A-Za-z0-9_]*)"/g
const REQUIREMENT_HEADING = /^#{3,4} REQ-[A-Z0-9-]+/
const HEADING = /^#{1,6} /

export function collectErrorCodes(typespec: string): ErrorCodes {
  const codes: ErrorCodes = new Map()
  for (const match of typespec.matchAll(ERROR_CODE)) {
    const code = match[1] ?? ''
    if (!codes.has(code)) codes.set(code, [])
  }
  for (const match of typespec.matchAll(DOCUMENTED_MODEL)) {
    for (const code of (match[1] ?? '').matchAll(ERROR_CODE)) {
      codes.get(code[1] ?? '')?.push(match[2] ?? '')
    }
  }
  return codes
}

export function collectEventTypes(go: string): string[] {
  return [...go.matchAll(EVENT_TYPE)].map((match) => match[1] ?? '')
}

/**
 * 要件の見出しから次の見出しまでを要件の本文として、遷移の表の行を遷移として読む。
 * 遷移の表の `Event` 列はイベントを発行する条件を述べるので、イベントの記載に数える。
 */
export function specifiedTexts(source: string): SpecifiedTexts {
  const requirements: string[] = []
  const transitions: string[] = []
  let current = -1
  let inTransitions = false
  for (const line of source.split('\n')) {
    if (HEADING.test(line)) {
      current = REQUIREMENT_HEADING.test(line) ? requirements.push(line) - 1 : -1
    } else if (current >= 0) {
      requirements[current] += `\n${line}`
    }
    const row = line.trim()
    if (row === TRANSITION_HEADER) inTransitions = true
    else if (!row.startsWith('|')) inTransitions = false
    else if (inTransitions) transitions.push(row)
  }
  return { requirements, transitions }
}

export function findUnspecified(vocabulary: Vocabulary, texts: SpecifiedTexts): UnspecifiedTerm[] {
  const requirements = texts.requirements.join('\n')
  const withTransitions = `${requirements}\n${texts.transitions.join('\n')}`
  const unspecified: UnspecifiedTerm[] = []
  for (const [code, models] of vocabulary.errors) {
    if (![code, ...models].some((name) => mentions(requirements, name))) {
      unspecified.push({ kind: 'error', name: code })
    }
  }
  for (const event of vocabulary.events) {
    if (!mentions(withTransitions, event)) unspecified.push({ kind: 'event', name: event })
  }
  return unspecified
}

export function compareWithDebt(
  found: readonly UnspecifiedTerm[],
  debt: VocabularyDebt,
): { fresh: UnspecifiedTerm[]; stale: UnspecifiedTerm[] } {
  const listed = (term: UnspecifiedTerm) =>
    (term.kind === 'error' ? debt.errors : debt.events).includes(term.name)
  const foundKeys = new Set(found.map((term) => `${term.kind}:${term.name}`))
  const debtTerms: UnspecifiedTerm[] = [
    ...debt.errors.map((name) => ({ kind: 'error' as const, name })),
    ...debt.events.map((name) => ({ kind: 'event' as const, name })),
  ]
  return {
    fresh: found.filter((term) => !listed(term)),
    stale: debtTerms.filter((term) => !foundKeys.has(`${term.kind}:${term.name}`)),
  }
}

/** 語の一部ではなく、語として現れるか。`UserDisabled` は `UserDisabledAt` の中では数えない。 */
function mentions(text: string, name: string): boolean {
  return new RegExp(`(?<![A-Za-z0-9_])${name}(?![A-Za-z0-9_])`).test(text)
}

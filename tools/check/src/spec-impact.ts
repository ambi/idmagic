/**
 * 仕様影響の宣言と、仕様差分の矛盾を見つける。
 *
 * 変更が仕様を変えるかどうかを、実装から推論しない。変更者が work item またはコミットの
 * トレーラーに書いた宣言を読み、それが `spec-diff` の差分と食い違う場合だけを報告する。
 * ここにあるのは値の上の判断だけで、Git と作業ツリーの読み取りは check-spec-impact.ts が担う。
 */

import { citedNormativeIds } from './normative-coverage.ts'
import type { SpecificationDiff } from './spec-diff.ts'

export type Impact = 'modifies' | 'conforms'

export type SpecReference = {
  path: string
  requirement?: string
  symbol?: string
  impact: Impact
}

export type SpecImpactDeclaration =
  | { kind: 'affected'; references: SpecReference[] }
  | { kind: 'none'; reason: string }

export type WorkRange = {
  /** 作業範囲の起点から作業ツリーまでの仕様差分。 */
  diff: SpecificationDiff
  /** 同じ範囲で変更された別の work item が `modifies` として挙げた規範要素。 */
  claimedByOthers: readonly SpecReference[]
  /** 同じ範囲で追加または変更されたテストファイルの本文。 */
  changedTests: readonly string[]
}

export type CheckedWorkItem = {
  path: string
  declaration: SpecImpactDeclaration
  /** 起点を履歴から求められなかった場合は undefined。 */
  range: WorkRange | undefined
}

export type CheckedCommit = {
  sha: string
  subject: string
  productionPaths: readonly string[]
  /** 同じコミットで変更した work item、または checkpoint が名指す work item に宣言がある。 */
  workItemDeclared: boolean
  /** `Spec-Impact` トレーラーの値。トレーラーがなければ undefined。 */
  trailer: string | undefined
  /** このコミット自体の仕様差分。 */
  diff: SpecificationDiff
}

export type SpecImpactInput = {
  items: readonly CheckedWorkItem[]
  commits: readonly CheckedCommit[]
  /** 基準から作業ツリーまでの仕様差分。 */
  diff: SpecificationDiff
}

function object(value: unknown): Record<string, unknown> | undefined {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

/** work item の frontmatter から仕様影響の宣言を読む。どちらの宣言もなければ undefined。 */
export function specImpactDeclaration(
  record: Record<string, unknown>,
): SpecImpactDeclaration | undefined {
  const none = object(record.spec_impact)
  if (none?.kind === 'none') {
    return { kind: 'none', reason: typeof none.reason === 'string' ? none.reason : '' }
  }
  if (!Array.isArray(record.affected_spec)) return undefined
  const references: SpecReference[] = []
  for (const entry of record.affected_spec) {
    const reference = object(entry)
    // path のない項目は、完了済みの記録に残る廃止済みの参照形式である。
    if (typeof reference?.path !== 'string') continue
    references.push({
      path: reference.path,
      requirement: typeof reference.requirement === 'string' ? reference.requirement : undefined,
      symbol: typeof reference.symbol === 'string' ? reference.symbol : undefined,
      impact: reference.impact === 'conforms' ? 'conforms' : 'modifies',
    })
  }
  return { kind: 'affected', references }
}

/**
 * 参照が指す規範要素を、`spec-diff` が使う名前で返す。
 *
 * 例の ID は親の規則へ寄せる。`spec-diff` は例の手順を規則の事実に含めて比べるためである。
 */
export function referencedElement(reference: SpecReference): string | undefined {
  if (reference.symbol !== undefined) {
    return `${reference.path}:${reference.symbol.split('.').at(-1)}`
  }
  const requirement = reference.requirement
  if (requirement === undefined) return undefined
  if (requirement.startsWith('REQ-')) return requirement
  const example = requirement.match(/^EX-(.+)-\d+$/)
  if (example) return `REQ-${example[1]}`
  return `${reference.path}#${requirement}`
}

function label(reference: SpecReference): string {
  return `${reference.path} ${reference.requirement ?? reference.symbol ?? ''}`.trim()
}

/** 規範要素として `affected_spec` から名指せる差分。追加、変更、削除のすべて。 */
function touchedElements(diff: SpecificationDiff): Set<string> {
  return new Set([...claimableElements(diff), ...removedElements(diff)])
}

/** 申告を求める差分。削除した要素は `affected_spec` から解決できないので含めない。 */
function claimableElements(diff: SpecificationDiff): string[] {
  return [
    ...diff.addedScenarios,
    ...diff.changedScenarios,
    ...diff.addedStandards,
    ...diff.changedStandards,
    ...diff.addedDeclarations,
    ...diff.changedDeclarations,
  ]
}

function removedElements(diff: SpecificationDiff): string[] {
  return [...diff.removedScenarios, ...diff.removedStandards, ...diff.removedDeclarations]
}

/** 状態遷移の差分。`affected_spec` から名指す形がないので、どの work item のものかを決められない。 */
function transitionChanges(diff: SpecificationDiff): string[] {
  return diff.changedTransitions.map((machine) => `state transitions of ${machine}`)
}

/** 規範仕様の差分のすべて。 */
function normativeChanges(diff: SpecificationDiff): string[] {
  return [
    ...new Set([
      ...touchedElements(diff),
      ...diff.addedDeprecations,
      ...diff.removedDeprecations,
      ...transitionChanges(diff),
    ]),
  ].sort()
}

function modifiedElements(references: readonly SpecReference[]): Set<string> {
  const elements = new Set<string>()
  for (const reference of references) {
    const element = reference.impact === 'modifies' ? referencedElement(reference) : undefined
    if (element !== undefined) elements.add(element)
  }
  return elements
}

function escapeForPattern(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * 変更されたテストが、参照する規範要素を引いているか。
 *
 * 規則は、その規則自体か子の例のどれかを引けばよい。TypeSpec のシンボルには
 * `//spec:covers` で引く ID がないので、宣言名がテストに現れることを求める。
 */
function citedByChangedTest(reference: SpecReference, tests: readonly string[]): boolean {
  if (reference.symbol !== undefined) {
    const name = escapeForPattern(reference.symbol.split('.').at(-1) ?? '')
    const named = new RegExp(`(?<![A-Za-z0-9_])${name}(?![A-Za-z0-9_])`)
    return tests.some((source) => named.test(source))
  }
  const requirement = reference.requirement
  if (requirement === undefined) return false
  const candidates = [requirement]
  if (requirement.startsWith('REQ-')) {
    const example = new RegExp(`EX-${escapeForPattern(requirement.slice('REQ-'.length))}-\\d+`, 'g')
    for (const source of tests) candidates.push(...(source.match(example) ?? []))
  }
  return citedNormativeIds(tests, candidates).size > 0
}

const BOILERPLATE = [
  /リファクタリング|振る舞い|挙動|動作|仕様|機能|影響|変更|修正|内部|構造|整理|実装|コード/g,
  /単なる|純粋な|のみ|だけ|なし|ない|ません|しない|変え|変わら|与え|です|である|する|した/g,
  /\b(?:refactor(?:ing|ed)?|behaviou?r(?:al)?|functional(?:ity)?|spec(?:ification)?|impact)\b/gi,
  /\b(?:change[sd]?|unchanged|affect(?:s|ed)?|internal|structur(?:e|al)|implementation)\b/gi,
  /\b(?:clean(?:\s*up)?|code|pure(?:ly)?|only|just|simple|minor|same|none|n\/a|no|not)\b/gi,
  /\b(?:is|are|the|a|an|it|this|does|do|and|or|of|to|in)\b/gi,
]

/**
 * 理由が定型句だけで終わるか。
 *
 * 機械にできるのは、何が維持されるかを一つも名指さない理由を拒否するところまでである。
 * 名指したものが正しいかはレビューで判断する。
 */
export function isBoilerplateReason(reason: string): boolean {
  let rest = reason
  for (const phrase of BOILERPLATE) rest = rest.replace(phrase, ' ')
  // 助詞と句読点は、定型句を除いた後に何も名指していないことを隠す。
  const named = rest.replace(/[\s\p{P}\p{S}]|[はをにがもでとのへや]/gu, '')
  return named.length < 4
}

/** `Spec-Impact` トレーラーの値から理由を読む。値は `none — <理由>` の形だけを認める。 */
export function parseSpecImpactTrailer(value: string): { reason: string } | { error: string } {
  const match = value.trim().match(/^(\S+?)(?:\s*(?:—|–|--|-|:)\s*|\s+|$)([\s\S]*)$/)
  const kind = match?.[1] ?? ''
  if (kind !== 'none') {
    return {
      error:
        `Spec-Impact accepts only "none", not "${kind}". ` +
        'A change that alters the specification needs a work item.',
    }
  }
  return { reason: (match?.[2] ?? '').trim() }
}

const STARTED = new Set(['in_progress', 'completed', 'cancelled'])

/**
 * work item の作業範囲の起点を返す。
 *
 * `history` は work item のファイルを変更したコミットを古い順に並べたものである。状態が初めて
 * `pending` でなくなったコミットの親を起点にする。その遷移がまだコミットされていなければ
 * `head` を起点にする。コミット済みの状態が着手後なのに履歴から遷移を見つけられない場合は、
 * 基準リビジョンへ黙って戻さず undefined を返す。
 */
export function workRangeStart(
  history: ReadonlyArray<{ sha: string; status: string | undefined }>,
  committedStatus: string | undefined,
  head: string,
): string | undefined {
  const started = history.find(
    (commit) => commit.status !== undefined && STARTED.has(commit.status),
  )
  if (started) return `${started.sha}^`
  return committedStatus !== undefined && STARTED.has(committedStatus) ? undefined : head
}

function verifyWorkItem(item: CheckedWorkItem): string[] {
  const findings: string[] = []
  const { declaration, range } = item
  if (declaration.kind === 'none' && isBoilerplateReason(declaration.reason)) {
    findings.push(
      'spec_impact.reason names nothing that stays the same; say which results, state, events, or calls are preserved',
    )
  }
  if (range === undefined) {
    return [
      ...findings,
      'cannot find the commit where this work item left pending, so its work range is unknown',
    ]
  }
  const claimedByOthers = modifiedElements(range.claimedByOthers)
  if (declaration.kind === 'none') {
    // 状態遷移は名指せないので、同じ範囲に仕様を変える別の work item があれば、その作業のものとして扱う。
    const unattributable = new Set(claimedByOthers.size > 0 ? transitionChanges(range.diff) : [])
    const changes = normativeChanges(range.diff).filter(
      (change) => !claimedByOthers.has(change) && !unattributable.has(change),
    )
    if (changes.length > 0) {
      findings.push(
        `spec_impact is none, but the work range changes the normative specification: ${changes.join(', ')}`,
      )
    }
    return findings
  }
  const touched = touchedElements(range.diff)
  for (const reference of declaration.references) {
    const element = referencedElement(reference)
    if (element === undefined) continue
    if (reference.impact === 'modifies') {
      if (!touched.has(element)) {
        findings.push(
          `affected_spec ${label(reference)} is impact: modifies, but the work range does not change it; ` +
            'change the specification, or declare impact: conforms',
        )
      }
      continue
    }
    if (touched.has(element) && !claimedByOthers.has(element)) {
      findings.push(
        `affected_spec ${label(reference)} is impact: conforms, but the work range changes it`,
      )
    }
    if (!citedByChangedTest(reference, range.changedTests)) {
      findings.push(
        `affected_spec ${label(reference)} is impact: conforms, but no test added or changed in the work range names it`,
      )
    }
  }
  return findings
}

function verifyCommit(commit: CheckedCommit): string[] {
  const findings: string[] = []
  if (commit.trailer === undefined) {
    if (commit.productionPaths.length > 0 && !commit.workItemDeclared) {
      const [first, ...others] = commit.productionPaths
      const more = others.length > 0 ? ` and ${others.length} more` : ''
      findings.push(
        `changes production code (${first}${more}) without declaring its specification impact; ` +
          'change the work item that declares affected_spec or spec_impact in the same commit, ' +
          'or add a "Spec-Impact: none — <what stays the same>" trailer',
      )
    }
    return findings
  }
  const trailer = parseSpecImpactTrailer(commit.trailer)
  if ('error' in trailer) return [trailer.error]
  if (isBoilerplateReason(trailer.reason)) {
    findings.push(
      'Spec-Impact reason names nothing that stays the same; say which results, state, events, or calls are preserved',
    )
  }
  const changes = normativeChanges(commit.diff)
  if (changes.length > 0) {
    findings.push(
      `declares Spec-Impact: none, but changes the normative specification: ${changes.join(', ')}`,
    )
  }
  return findings
}

export function verifySpecImpact(input: SpecImpactInput): string[] {
  const findings: string[] = []
  for (const item of input.items) {
    findings.push(...verifyWorkItem(item).map((finding) => `${item.path}: ${finding}`))
  }
  for (const commit of input.commits) {
    findings.push(
      ...verifyCommit(commit).map(
        (finding) => `${commit.sha.slice(0, 8)} ${commit.subject}: ${finding}`,
      ),
    )
  }
  const claimed = modifiedElements(
    input.items.flatMap((item) =>
      item.declaration.kind === 'affected' ? item.declaration.references : [],
    ),
  )
  const unclaimed = [...new Set(claimableElements(input.diff))]
    .filter((element) => !claimed.has(element))
    .sort()
  for (const element of unclaimed) {
    findings.push(
      `${element}: the specification changes here, but no work item changed in this range lists it in affected_spec as impact: modifies`,
    )
  }
  return findings
}

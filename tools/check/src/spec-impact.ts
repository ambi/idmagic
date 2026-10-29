/**
 * 仕様影響の宣言と、仕様差分の矛盾を見つける。
 *
 * 変更が仕様を変えるかどうかを、実装から推論しない。変更者が work item またはコミットの
 * トレーラーに書いた宣言を読み、それが `spec-diff` の差分と食い違う場合だけを報告する。
 * ここにあるのは値の上の判断だけで、Git と作業ツリーの読み取りは check-spec-impact.ts が担う。
 */

import type { SpecificationDiff } from './spec-diff.ts'

/** `affected_spec` の一項目。この work item が触れる規範要素を名指す。 */
export type SpecReference = {
  path: string
  requirement?: string
  symbol?: string
}

export type SpecImpactDeclaration =
  | { kind: 'affected'; references: SpecReference[] }
  | { kind: 'none'; reason: string }

export type WorkRange = {
  /** 作業範囲から完了した作業の区間を除いた、区間ごとの仕様差分。 */
  diffs: readonly SpecificationDiff[]
  /** 同じ範囲で変更された、進行中の別の work item が `affected_spec` に挙げた規範要素。 */
  claimedByOthers: readonly SpecReference[]
  /** 作業範囲の起点に存在し、区間の中で変わったか消えた特性化テスト。 */
  changedCharacterizations: readonly string[]
}

export type CheckedWorkItem = {
  path: string
  declaration: SpecImpactDeclaration
  /** 起点を履歴から求められなかった場合は undefined。 */
  range: WorkRange | undefined
}

/** どの作業範囲にも完了した作業の区間にも入らないコミット。 */
export type CheckedCommit = {
  sha: string
  subject: string
  productionPaths: readonly string[]
  /** `Spec-Impact` トレーラーの値。トレーラーがなければ undefined。 */
  trailer: string | undefined
  /** このコミット自体の仕様差分。 */
  diff: SpecificationDiff
  /** 親コミットに存在し、このコミットで変わったか消えた特性化テスト。 */
  changedCharacterizations: readonly string[]
}

export type SpecImpactInput = {
  items: readonly CheckedWorkItem[]
  commits: readonly CheckedCommit[]
  /** 基準から作業ツリーまでのうち、完了した作業の区間を除いた区間ごとの仕様差分。 */
  diffs: readonly SpecificationDiff[]
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
function normativeChanges(diffs: readonly SpecificationDiff[]): string[] {
  return [
    ...new Set(
      diffs.flatMap((diff) => [
        ...claimableElements(diff),
        ...removedElements(diff),
        ...diff.addedDeprecations,
        ...diff.removedDeprecations,
        ...transitionChanges(diff),
      ]),
    ),
  ].sort()
}

function claimedElements(references: readonly SpecReference[]): Set<string> {
  return new Set(references.flatMap((reference) => referencedElement(reference) ?? []))
}

/** `Spec-Impact` トレーラーの値を確かめる。値は `none` だけを認め、理由は読む人に任せる。 */
export function parseSpecImpactTrailer(value: string): { error: string } | undefined {
  const kind = value.trim().match(/^[^\s—–:-]*/)?.[0] ?? ''
  if (kind === 'none') return undefined
  return {
    error:
      `Spec-Impact accepts only "none", not "${kind}". ` +
      'A change that alters the specification needs a work item.',
  }
}

const CHARACTERIZATION = /^func (TestCharacterize\w*)\(/gm

/**
 * Go のテストファイルから、特性化テストの関数ごとの本文を読む。
 *
 * gofmt 済みのトップレベルの関数は、行頭の `}` で閉じる。テストの意味は解析しない。
 */
function characterizations(source: string): Map<string, string> {
  const bodies = new Map<string, string>()
  for (const match of source.matchAll(CHARACTERIZATION)) {
    const end = source.indexOf('\n}', match.index)
    bodies.set(match[1] ?? '', source.slice(match.index, end < 0 ? undefined : end + 2))
  }
  return bodies
}

/** `before` にあった特性化テストのうち、`after` で本文が変わったか消えたものの名前。 */
export function changedCharacterizations(
  before: string | undefined,
  after: string | undefined,
): string[] {
  if (before === undefined) return []
  const kept = characterizations(after ?? '')
  return [...characterizations(before)]
    .filter(([name, body]) => kept.get(name) !== body)
    .map(([name]) => name)
}

const STARTED = new Set(['in_progress', 'completed', 'cancelled'])

/** 状態が pending でなくなった後のものか。 */
export function isStarted(status: string | undefined): boolean {
  return status !== undefined && STARTED.has(status)
}

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
  const started = history.find((commit) => isStarted(commit.status))
  if (started) return `${started.sha}^`
  return isStarted(committedStatus) ? undefined : head
}

function verifyWorkItem(item: CheckedWorkItem): string[] {
  const { declaration, range } = item
  if (range === undefined) {
    return [
      'cannot find the commit where this work item left pending, so its work range is unknown',
    ]
  }
  if (declaration.kind === 'affected') return []
  const findings: string[] = []
  const claimedByOthers = claimedElements(range.claimedByOthers)
  // 状態遷移と削除した宣言は affected_spec で名指せないので、同じ範囲に仕様を変える別の
  // work item があれば、その作業のものとして扱う。
  const unattributable = new Set(
    claimedByOthers.size > 0
      ? range.diffs.flatMap((diff) => [...transitionChanges(diff), ...removedElements(diff)])
      : [],
  )
  const changes = normativeChanges(range.diffs).filter(
    (change) => !claimedByOthers.has(change) && !unattributable.has(change),
  )
  if (changes.length > 0) {
    findings.push(
      `spec_impact is none, but the work range changes the normative specification: ${changes.join(', ')}`,
    )
  }
  if (range.changedCharacterizations.length > 0) {
    findings.push(
      `spec_impact is none, but the work range changes or removes characterization tests: ${range.changedCharacterizations.join(', ')}`,
    )
  }
  return findings
}

function verifyCommit(commit: CheckedCommit): string[] {
  const findings: string[] = []
  if (commit.trailer === undefined) {
    if (commit.productionPaths.length > 0) {
      const [first, ...others] = commit.productionPaths
      const more = others.length > 0 ? ` and ${others.length} more` : ''
      findings.push(
        `changes production code (${first}${more}) without declaring its specification impact; ` +
          'commit it within the work range of a work item that declares affected_spec or spec_impact, ' +
          'or add a "Spec-Impact: none — <what stays the same>" trailer',
      )
    }
    // 宣言のないコミットで書き換えられると、特性化テストが検出した振る舞いの変化を消せる。
    if (commit.changedCharacterizations.length > 0) {
      findings.push(
        `changes or removes characterization tests (${commit.changedCharacterizations.join(', ')}) ` +
          'without a work item that declares affected_spec',
      )
    }
    return findings
  }
  const trailer = parseSpecImpactTrailer(commit.trailer)
  if (trailer) return [trailer.error]
  const changes = normativeChanges([commit.diff])
  if (changes.length > 0) {
    findings.push(
      `declares Spec-Impact: none, but changes the normative specification: ${changes.join(', ')}`,
    )
  }
  if (commit.changedCharacterizations.length > 0) {
    findings.push(
      `declares Spec-Impact: none, but changes or removes characterization tests: ${commit.changedCharacterizations.join(', ')}`,
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
  const claimed = claimedElements(
    input.items.flatMap((item) =>
      item.declaration.kind === 'affected' ? item.declaration.references : [],
    ),
  )
  const unclaimed = [...new Set(input.diffs.flatMap(claimableElements))]
    .filter((element) => !claimed.has(element))
    .sort()
  for (const element of unclaimed) {
    findings.push(
      `${element}: the specification changes here, but no work item changed in this range lists it in affected_spec`,
    )
  }
  return findings
}

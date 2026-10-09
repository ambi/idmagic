/**
 * コミットごとの変更ファイルから、同じコミットで変わったモジュールの組と回数を数える。
 *
 * 数値は境界の判断で変更シナリオを探す補助であり、合否を決めない。
 * 同じコミットに入ることは同じ変更理由を証明せず、共変更が少ないことも独立性を証明しない。
 */

import { type ArchitectureModel, ownerOf } from '../../check/src/boundary-fitness.ts'

export type CommitChange = {
  id: string
  /** 第一親との差分に現れたパス。改名は旧パスと新パスの両方を含む。 */
  paths: string[]
}

/** パスの分類。`ignored` は数える対象外のファイル、`unclassified` は現在の割り当てで解決できない対象。 */
export type PathClass =
  | { kind: 'module'; name: string }
  | { kind: 'ignored' }
  | { kind: 'unclassified' }

export type ModulePair = {
  first: string
  second: string
  firstChanges: number
  secondChanges: number
  together: number
}

export type CouplingReport = {
  scanned: number
  counted: number
  withoutModule: number
  unclassifiedPaths: number
  excludeAt: number
  excluded: { id: string; modules: number }[]
  pairs: ModulePair[]
}

/** sqlc がクエリ入力から生成するファイル名。生成先のディレクトリにあるものだけを生成物とみなす。 */
const SQLC_GENERATED = /^(?:db|models|querier)\.go$|\.sql\.go$/

/**
 * 現在の責務表と sqlc の設定でパスを分類する。対象はモジュール配下の本番 `.go` と sqlc のクエリ入力に限る。
 * 文書、テスト、生成物、共有ライブラリ、組み立て地点は数えない。
 */
export function pathClassifier(input: {
  architecture: ArchitectureModel
  queryDirectories: ReadonlySet<string>
  generatedDirectories: ReadonlySet<string>
}): (path: string) => PathClass {
  return (path) => {
    if (!path.startsWith('backend/')) return { kind: 'ignored' }
    const directory = path.split('/').slice(0, -1).join('/')
    const name = path.split('/').at(-1) ?? ''
    if (path.endsWith('.sql')) {
      if (!input.queryDirectories.has(directory)) return { kind: 'unclassified' }
    } else if (!path.endsWith('.go') || path.endsWith('_test.go')) {
      return { kind: 'ignored' }
    } else if (input.generatedDirectories.has(directory) && SQLC_GENERATED.test(name)) {
      return { kind: 'ignored' }
    }
    const owner = ownerOf(directory, input.architecture)
    if (!owner) return { kind: 'unclassified' }
    return owner.kind === 'module'
      ? { kind: 'module', name: owner.module.name }
      : { kind: 'ignored' }
  }
}

/** `excludeAt` 個以上のモジュールを変えるコミットは、組の回数と各モジュールの回数の双方から除く。 */
export function changeCoupling(
  commits: readonly CommitChange[],
  classify: (path: string) => PathClass,
  excludeAt: number,
): CouplingReport {
  const moduleChanges = new Map<string, number>()
  const together = new Map<string, number>()
  const excluded: { id: string; modules: number }[] = []
  let counted = 0
  let withoutModule = 0
  let unclassifiedPaths = 0
  for (const commit of commits) {
    const modules = new Set<string>()
    for (const path of new Set(commit.paths)) {
      const pathClass = classify(path)
      if (pathClass.kind === 'module') modules.add(pathClass.name)
      if (pathClass.kind === 'unclassified') unclassifiedPaths += 1
    }
    if (modules.size === 0) {
      withoutModule += 1
      continue
    }
    if (modules.size >= excludeAt) {
      excluded.push({ id: commit.id, modules: modules.size })
      continue
    }
    counted += 1
    const names = [...modules].sort()
    for (const name of names) moduleChanges.set(name, (moduleChanges.get(name) ?? 0) + 1)
    for (let i = 0; i < names.length; i += 1) {
      for (let j = i + 1; j < names.length; j += 1) {
        const key = `${names[i]}\0${names[j]}`
        together.set(key, (together.get(key) ?? 0) + 1)
      }
    }
  }
  const pairs = [...together].map(([key, count]): ModulePair => {
    const [first, second] = key.split('\0') as [string, string]
    return {
      first,
      second,
      firstChanges: moduleChanges.get(first) ?? 0,
      secondChanges: moduleChanges.get(second) ?? 0,
      together: count,
    }
  })
  pairs.sort(
    (left, right) =>
      right.together - left.together ||
      left.first.localeCompare(right.first) ||
      left.second.localeCompare(right.second),
  )
  return {
    scanned: commits.length,
    counted,
    withoutModule,
    unclassifiedPaths,
    excludeAt,
    excluded,
    pairs,
  }
}

export function renderCouplingReport(
  report: CouplingReport,
  scan: { revision: string; limit: number },
): string {
  const ratio = (part: number, whole: number) => (whole === 0 ? '-' : (part / whole).toFixed(2))
  const lines = [
    `revision: ${scan.revision}`,
    `scan: first-parent, up to ${scan.limit} commit(s), exclude commits changing ${report.excludeAt} or more modules`,
    `commits: ${report.scanned} scanned, ${report.counted} counted, ${report.withoutModule} without a module change, ${report.excluded.length} excluded`,
    `unclassified paths: ${report.unclassifiedPaths}`,
  ]
  if (report.scanned < scan.limit) {
    lines.push(`history: only ${report.scanned} commit(s) exist below the revision`)
  }
  for (const commit of report.excluded) {
    lines.push(`excluded: ${commit.id} (${commit.modules} modules)`)
  }
  lines.push('', 'A\tB\tn(A)\tn(B)\tn(A,B)\tn(A,B)/n(A)\tn(A,B)/n(B)')
  for (const pair of report.pairs) {
    lines.push(
      [
        pair.first,
        pair.second,
        pair.firstChanges,
        pair.secondChanges,
        pair.together,
        ratio(pair.together, pair.firstChanges),
        ratio(pair.together, pair.secondChanges),
      ].join('\t'),
    )
  }
  return lines.join('\n')
}

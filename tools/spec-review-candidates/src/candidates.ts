/**
 * 仕様 ID を引くテストが実行しない本番コードを、レビューの候補として求める。
 *
 * 実行されないコードが実装詳細の場合もあり、実行されるコードに仕様にない挙動がある場合も
 * ある。ここで求めるのは、仕様漏れを探して読み始める位置であって、判定ではない。
 * Go の起動とファイルの読み取りは main.ts が担い、このファイルは値の上の計算だけを置く。
 */

import { citedNormativeIds } from '../../check/src/normative-coverage.ts'

export type CoverBlock = {
  file: string
  startLine: number
  startColumn: number
  endLine: number
  endColumn: number
  count: number
}

export type LineRange = { start: number; end: number }

export type PackageImports = {
  importPath: string
  imports: readonly string[]
  /** パッケージ内のテストと外部テストパッケージが import するもの。 */
  testImports: readonly string[]
}

const TOP_LEVEL_FUNCTION = /^func\s+(\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*[([]/
/** `go test` が実行する関数の名前。接頭辞の次が小文字の関数はテストではない。 */
const TEST_NAME = /^(?:Test|Fuzz)(?:[^a-z]|$)/

/**
 * Go のテストファイルから、宣言済みの規範 ID を引くテスト関数の名前を返す。
 *
 * 引用は、関数の直上の `//spec:covers` と、関数の本体にある ID だけの文字列リテラルに限る。
 * 関数の外に置いた表のリテラルは、どのテストが使うかを字面から決められないので数えない。
 * 数えると、仕様を引かないテストが実行したコードまで候補から外れる。
 */
export function specCitingTests(source: string, declaredIds: readonly string[]): string[] {
  const lines = source.split('\n')
  const citing: string[] = []
  for (const [index, line] of lines.entries()) {
    const declaration = line.match(TOP_LEVEL_FUNCTION)
    const name = declaration?.[2]
    if (!name || declaration[1] !== undefined || !TEST_NAME.test(name)) continue

    let first = index
    while (first > 0 && lines[first - 1]!.startsWith('//')) first -= 1
    let last = index
    // 一行で閉じる関数でなければ、行頭の閉じ括弧までが本体である。
    if (!line.trimEnd().endsWith('}')) {
      while (last < lines.length - 1 && lines[last] !== '}') last += 1
    }
    const text = lines.slice(first, last + 1).join('\n')
    if (citedNormativeIds([text], declaredIds).size > 0) citing.push(name)
  }
  return citing
}

function within(target: string, importPath: string): boolean {
  return importPath === target || importPath.startsWith(`${target}/`)
}

/**
 * 対象のパッケージと、そのテストが対象のコードを実行し得るパッケージを返す。
 *
 * `-coverpkg` は別パッケージのテストによる実行も数える。対象へ到達しないパッケージのテストは
 * 対象を一行も実行しないので、起動しない。
 */
export function packagesReaching(target: string, graph: readonly PackageImports[]): string[] {
  const packages = new Map(graph.map((one) => [one.importPath, one]))
  const reaches = new Map<string, boolean>()
  const reachesByImports = (importPath: string): boolean => {
    const known = reaches.get(importPath)
    if (known !== undefined) return known
    // 先に偽を置き、import の循環で止まらないようにする。
    reaches.set(importPath, false)
    const result =
      within(target, importPath) ||
      (packages.get(importPath)?.imports ?? []).some((next) => reachesByImports(next))
    reaches.set(importPath, result)
    return result
  }
  return graph
    .filter(
      (one) =>
        reachesByImports(one.importPath) || one.testImports.some((next) => reachesByImports(next)),
    )
    .map((one) => one.importPath)
    .sort()
}

/**
 * 一つのパッケージで、名指したテストだけを、対象の被覆つきで実行する `go` の引数。
 *
 * `-run` を外すと仕様を引かないテストが実行したコードまで候補から外れ、`-coverpkg` を外すと
 * 別パッケージのテストによる実行を数えなくなる。
 */
export function goTestArguments(
  packageDirectory: string,
  tests: readonly string[],
  target: string,
  profile: string,
): string[] {
  return [
    'test',
    `-run=^(${tests.join('|')})$`,
    `-coverpkg=./${target}/...`,
    `-coverprofile=${profile}`,
    `./${packageDirectory}`,
  ]
}

export function parseCoverProfile(profile: string): CoverBlock[] {
  const blocks: CoverBlock[] = []
  for (const line of profile.split('\n')) {
    const match = line.match(/^(.+):(\d+)\.(\d+),(\d+)\.(\d+) \d+ (\d+)$/)
    if (!match) continue
    blocks.push({
      file: match[1]!,
      startLine: Number(match[2]),
      startColumn: Number(match[3]),
      endLine: Number(match[4]),
      endColumn: Number(match[5]),
      count: Number(match[6]),
    })
  }
  return blocks
}

/** どのプロファイルでも実行されなかったブロックを、ファイルごとの行の範囲にまとめる。 */
export function unexecutedRanges(blocks: readonly CoverBlock[]): Map<string, LineRange[]> {
  const executed = new Map<string, { block: CoverBlock; count: number }>()
  for (const block of blocks) {
    const key = `${block.file}:${block.startLine}.${block.startColumn},${block.endLine}.${block.endColumn}`
    const known = executed.get(key)
    executed.set(key, { block, count: (known?.count ?? 0) + block.count })
  }
  const byFile = new Map<string, LineRange[]>()
  for (const { block, count } of executed.values()) {
    if (count > 0) continue
    const ranges = byFile.get(block.file) ?? []
    ranges.push({ start: block.startLine, end: block.endLine })
    byFile.set(block.file, ranges)
  }
  const merged = new Map<string, LineRange[]>()
  for (const file of [...byFile.keys()].sort()) {
    const ranges: LineRange[] = []
    for (const range of byFile.get(file)!.sort((left, right) => left.start - right.start)) {
      const previous = ranges.at(-1)
      if (previous && range.start <= previous.end + 1) {
        previous.end = Math.max(previous.end, range.end)
      } else {
        ranges.push({ ...range })
      }
    }
    merged.set(file, ranges)
  }
  return merged
}

export type Report = {
  /** 対象のパッケージディレクトリ。 */
  target: string
  /** パッケージディレクトリから、実行した仕様を引くテストの名前。 */
  tests: ReadonlyMap<string, readonly string[]>
  /** テストが失敗した、またはビルドできなかったパッケージディレクトリ。 */
  failedPackages: readonly string[]
  /** リポジトリ相対のファイルパスから、実行されなかった行の範囲。 */
  ranges: ReadonlyMap<string, readonly LineRange[]>
}

function directory(path: string): string {
  return path.slice(0, Math.max(0, path.lastIndexOf('/')))
}

export function formatReport(report: Report): string {
  const counts = new Map<string, number>()
  const locations: string[] = []
  for (const [file, ranges] of report.ranges) {
    counts.set(directory(file), (counts.get(directory(file)) ?? 0) + ranges.length)
    for (const range of ranges) {
      locations.push(
        range.start === range.end
          ? `${file}:${range.start}`
          : `${file}:${range.start}-${range.end}`,
      )
    }
  }
  const testCount = [...report.tests.values()].reduce((sum, names) => sum + names.length, 0)
  const width = Math.max(0, ...[...counts.keys()].map((name) => name.length))
  const sections = [
    `spec-review candidates for ${report.target}`,
    [
      'No test that cites a normative id executed the locations below. They are candidates to review',
      'for behavior the specification may be missing, not a finding about what is specified: code that',
      'was not executed can be an implementation detail, and code that was executed can still carry',
      'behavior no rule states.',
    ].join('\n'),
    `${locations.length} location(s) in ${counts.size} package(s), from ${testCount} test(s) in ${report.tests.size} package(s)`,
    `locations per package:\n${[...counts].map(([name, count]) => `  ${name.padEnd(width)}  ${count}`).join('\n')}`,
    `locations:\n${locations.map((location) => `  ${location}`).join('\n')}`,
    `tests run:\n${[...report.tests]
      .flatMap(([name, names]) => names.map((test) => `  ${name} ${test}`))
      .join('\n')}`,
  ]
  if (report.failedPackages.length > 0) {
    sections.push(
      `packages whose tests failed or did not build (their coverage may be incomplete):\n${report.failedPackages
        .map((name) => `  ${name}`)
        .join('\n')}`,
    )
  }
  return `${sections.join('\n\n')}\n`
}

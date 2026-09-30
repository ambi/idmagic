/**
 * 仕様影響の宣言を求める「本番コード」の範囲を決める。
 *
 * 範囲の判定はここにだけ置く。検査と報告が別々に判定を持つと、片方だけが除外を増やしたときに
 * 宣言を求めるコードと候補として報告するコードがずれる。
 */

export type ProductionCodeExclusions = {
  /** 本番のエントリーポイントから import されない Go パッケージのディレクトリ。 */
  testSupportPackages: ReadonlySet<string>
  /** 生成されたことをヘッダーで宣言する Go ファイル。 */
  generatedFiles: ReadonlySet<string>
}

const GENERATED_HEADER = /^\/\/ Code generated .* DO NOT EDIT\.$/m
const ENTRY_POINT_TREE = 'backend/cmd/'

function directory(path: string): string {
  return path.slice(0, Math.max(0, path.lastIndexOf('/')))
}

function escapeForPattern(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * 作業ツリーの Go ソースから、本番コードに数えないパッケージとファイルを求める。
 *
 * `sources` は `backend/` 配下の `.go` のパスから本文への対応とする。テストファイルの import は
 * たどらない。テストだけが使うパッケージを本番コードにしないためである。
 */
export function productionCodeExclusions(
  modulePath: string,
  sources: ReadonlyMap<string, string>,
): ProductionCodeExclusions {
  const imported = new RegExp(`"${escapeForPattern(modulePath)}/(backend/[^"]+)"`, 'g')
  const imports = new Map<string, Set<string>>()
  const entryPoints = new Set<string>()
  const generatedFiles = new Set<string>()
  for (const [path, source] of sources) {
    if (!path.endsWith('.go') || path.endsWith('_test.go')) continue
    if (GENERATED_HEADER.test(source)) generatedFiles.add(path)
    const packageDirectory = directory(path)
    const targets = imports.get(packageDirectory) ?? new Set<string>()
    for (const match of source.matchAll(imported)) if (match[1]) targets.add(match[1])
    imports.set(packageDirectory, targets)
    if (path.startsWith(ENTRY_POINT_TREE) && /^package main\b/m.test(source)) {
      entryPoints.add(packageDirectory)
    }
  }

  const reachable = new Set<string>()
  const pending = [...entryPoints]
  while (pending.length > 0) {
    const packageDirectory = pending.pop()!
    if (reachable.has(packageDirectory)) continue
    reachable.add(packageDirectory)
    pending.push(...(imports.get(packageDirectory) ?? []))
  }
  return {
    testSupportPackages: new Set([...imports.keys()].filter((one) => !reachable.has(one))),
    generatedFiles,
  }
}

function isBackendProductionCode(path: string, exclusions: ProductionCodeExclusions): boolean {
  if (!path.endsWith('.go') || path.endsWith('_test.go')) return false
  if (path.split('/').includes('testdata')) return false
  if (exclusions.generatedFiles.has(path)) return false
  return !exclusions.testSupportPackages.has(directory(path))
}

function isFrontendProductionCode(path: string): boolean {
  if (path.startsWith('frontend/src/test/')) return false
  return !/(?:\.(?:test|spec)\.tsx?|\.gen\.ts|\.d\.ts)$/.test(path)
}

export function isProductionCode(path: string, exclusions: ProductionCodeExclusions): boolean {
  if (path.startsWith('backend/')) return isBackendProductionCode(path, exclusions)
  if (path.startsWith('frontend/src/')) return isFrontendProductionCode(path)
  return false
}

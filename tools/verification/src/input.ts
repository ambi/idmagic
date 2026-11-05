import { createHash } from 'node:crypto'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'

export function git(root: string, ...args: string[]): string {
  const result = Bun.spawnSync(['git', ...args], { cwd: root, stderr: 'pipe' })
  if (result.exitCode !== 0)
    throw new Error(`git ${args.join(' ')} failed: ${result.stderr.toString()}`)
  return result.stdout.toString()
}

/**
 * 検証の結果を左右し得る入力の範囲。
 *
 * `workspace` は revision と、追跡対象と未追跡の全ファイルである。作業項目を読む静的検査が使う。
 * `test-execution` は作業項目（`work-items/`）を除いたファイルだけである。作業項目は計画と記録であり、
 * テストの実行器が起動するどのテストも読まないので、書き換えてもテストの結果は変わらない。
 * ファイルの内容をすべて含むので revision も入力に加えない。加えると、作業項目だけのコミットでも
 * 実行記録が古くなる。
 */
export type InputSet = 'workspace' | 'test-execution'

const testExecutionExcludedDirectories = ['work-items/']

function includedIn(inputSet: InputSet, path: string): boolean {
  return (
    inputSet === 'workspace' ||
    !testExecutionExcludedDirectories.some((directory) => path.startsWith(directory))
  )
}

/** 入力の範囲に属するファイルの内容を固定し、実行中の編集も検出できる同一性を返す。 */
export async function verificationInput(
  snapshot: WorkspaceSnapshot,
  baseRevision?: string,
  inputSet: InputSet = 'workspace',
) {
  const revision = git(snapshot.root, 'rev-parse', 'HEAD').trim()
  // 範囲名を先に入れ、二つの範囲の値が偶然一致しないようにする。
  const hash = createHash('sha256').update(inputSet).update('\0')
  if (inputSet === 'workspace') hash.update(revision)
  const files = git(snapshot.root, 'ls-files', '--cached', '--others', '--exclude-standard', '-z')
    .split('\0')
    .filter((path) => path && includedIn(inputSet, path))
  for (const path of [...new Set(files)].sort()) {
    hash.update(path).update('\0')
    const file = Bun.file(snapshot.path(path))
    if (await file.exists()) hash.update(new Uint8Array(await file.arrayBuffer()))
    else hash.update('deleted')
    hash.update('\0')
  }
  return {
    revision,
    fingerprint: hash.digest('hex'),
    baseRevision,
    dirty: git(snapshot.root, 'status', '--porcelain').length > 0,
    inputSet,
  }
}

export function changedFiles(root: string, baseRevision: string): string[] {
  const base = git(root, 'merge-base', baseRevision, 'HEAD').trim()
  return [
    ...new Set(
      [
        ...git(root, 'diff', '--name-only', '-z', base, '--').split('\0'),
        ...git(root, 'ls-files', '--others', '--exclude-standard', '-z').split('\0'),
      ].filter(Boolean),
    ),
  ]
}

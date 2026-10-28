import { dirname, join, normalize } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { CheckOutcome } from './runner.ts'

/**
 * tools のテストが読むリポジトリのファイルを返す。テストの位置（`import.meta.dir`）から
 * `tools/` の外へ出る相対パスと、リポジトリの根を開く引数なしの `createWorkspaceSnapshot()` を
 * 数える。期待する HTML や fixture の Markdown のリンクは読み取りではないので、
 * `import.meta.dir` と同じ行の相対パスだけを見る。コメントの中は数えない。
 */
export function repositoryReads(
  path: string,
  source: string,
): Array<{ line: number; value: string }> {
  const found: Array<{ line: number; value: string }> = []
  for (const [index, raw] of source.split('\n').entries()) {
    const line = raw.replace(/\/\/.*$/, '')
    if (!line.includes('import.meta.dir')) {
      if (/\bcreateWorkspaceSnapshot\(\s*\)/.test(line))
        found.push({ line: index + 1, value: 'createWorkspaceSnapshot()' })
      continue
    }
    for (const match of line.matchAll(/(['"`])(\.\.?\/[^'"`]*|\.\.)\1/g)) {
      const value = match[2] ?? ''
      if (!normalize(join(dirname(path), value)).startsWith('tools/')) {
        found.push({ line: index + 1, value })
      }
    }
    if (/\bcreateWorkspaceSnapshot\(\s*\)/.test(line)) {
      found.push({ line: index + 1, value: 'createWorkspaceSnapshot()' })
    }
  }
  return found
}

/**
 * tools のテストが実リポジトリの文書や設定を読むと、それらだけの変更が tools のテストを
 * 失敗させ、変更の影響からテストを選べなくなる。リポジトリの状態の検査はリポジトリ検査に置き、
 * ツールのテストは fixture を使う。
 */
export async function checkToolTestInputs(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const lines: string[] = []
  // この検査自身のテストは、拒否する書き方を fixture の文字列として持つ。
  const tests = (await snapshot.files('tools')).filter(
    (path) =>
      path.endsWith('.test.ts') &&
      !path.includes('/node_modules/') &&
      path !== 'tools/check/src/tool-test-inputs.test.ts',
  )
  for (const path of tests) {
    for (const read of repositoryReads(path, await snapshot.read(path)))
      lines.push(
        `${path}:${read.line}: test reads the repository through ${read.value}; use a fixture`,
      )
  }
  return lines.length
    ? { ok: false, lines }
    : { ok: true, lines: [`ok  ${tests.length} tool test file(s) read no repository state`] }
}

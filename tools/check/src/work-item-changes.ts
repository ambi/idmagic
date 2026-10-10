/**
 * 完了済みの作業項目は当時の作業の記録であり、後のファイルの移動や形式の変更に追従させない。
 * そこで、記録を完了させる変更の中で一度だけ検証する。完了の変更は記録を
 * `work-items/done/` へ移すので、必ず変わった記録の集合に入る。
 */

import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'

/**
 * main とのマージベースから、または作業ツリーで変わったリポジトリ相対パス。
 * 完了の変更が何を触れたかを表す。Git の外の workspace では求められないので undefined を返す。
 */
export function changedPaths(
  snapshot: WorkspaceSnapshot,
  ...pathspec: string[]
): ReadonlySet<string> | undefined {
  const changedFromMain = Bun.spawnSync(
    ['git', 'diff', '--name-only', 'main...', '--', ...pathspec],
    { cwd: snapshot.root },
  )
  // 完了の変更では `work-items/done/` を初めて作ることがある。既定の status は新しい
  // ディレクトリだけを示し、その中の記録の名前を示さない。
  const status = Bun.spawnSync(
    ['git', 'status', '--porcelain', '--untracked-files=all', '--', ...pathspec],
    { cwd: snapshot.root },
  )
  if (changedFromMain.exitCode !== 0 && status.exitCode !== 0) return undefined
  const changed = new Set<string>()
  const add = (path: string) => {
    if (path.trim()) changed.add(path.trim())
  }
  for (const line of changedFromMain.stdout.toString().split('\n')) add(line)
  for (const line of status.stdout.toString().split('\n')) {
    for (const part of line.slice(3).split(' -> ')) add(part)
  }
  return changed
}

/**
 * main とのマージベースから、または作業ツリーで変わった作業項目の識別子（ファイル名の語幹）。
 * Git の外の workspace では求められないので undefined を返す。
 */
export function changedWorkItemRecords(
  snapshot: WorkspaceSnapshot,
): ReadonlySet<string> | undefined {
  const paths = changedPaths(snapshot, 'work-items')
  if (paths === undefined) return undefined
  const changed = new Set<string>()
  for (const path of paths) {
    const name = path.match(/([^/\\]+)\.md$/)?.[1]
    if (name) changed.add(name)
  }
  return changed
}

/**
 * このリポジトリ相対パスの文書を、いま検証するか。完了済みの記録は変わったときだけ検証する。
 * 変わった記録を求められない workspace では、すべてを検証する。
 */
export function verifiedNow(path: string, changed: ReadonlySet<string> | undefined): boolean {
  const record = path.match(/^work-items\/done\/([^/]+)\.md$/)?.[1]
  return record === undefined || changed === undefined || changed.has(record)
}

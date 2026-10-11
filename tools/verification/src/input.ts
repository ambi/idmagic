import { createHash } from 'node:crypto'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'

export function git(root: string, ...args: string[]): string {
  const result = Bun.spawnSync(['git', ...args], { cwd: root, stderr: 'pipe' })
  if (result.exitCode !== 0)
    throw new Error(`git ${args.join(' ')} failed: ${result.stderr.toString()}`)
  return result.stdout.toString()
}

/** revision に加え、追跡対象と未追跡の入力を固定し、実行中の編集も検出する。 */
export async function verificationInput(snapshot: WorkspaceSnapshot, baseRevision?: string) {
  const revision = git(snapshot.root, 'rev-parse', 'HEAD').trim()
  const hash = createHash('sha256').update(revision)
  const files = git(snapshot.root, 'ls-files', '--cached', '--others', '--exclude-standard', '-z')
    .split('\0')
    .filter(Boolean)
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

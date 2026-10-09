import type { CommitChange } from './coupling.ts'

/**
 * `revision` から第一親だけをたどり、各コミットの第一親との差分を読む。
 * マージは取り込んだ枝を第一親との差分一回として数え、ルートコミットは空の木との差分になる。
 */
export function readFirstParentChanges(
  root: string,
  revision: string,
  limit: number,
): { revision: string; commits: CommitChange[] } {
  const resolved = git(root, ['rev-parse', '--verify', `${revision}^{commit}`]).trim()
  const output = git(root, [
    'log',
    '--first-parent',
    '--diff-merges=first-parent',
    '--root',
    '-M',
    '--name-status',
    '--format=commit %H',
    `--max-count=${limit}`,
    resolved,
  ])
  const commits: CommitChange[] = []
  for (const line of output.split('\n')) {
    const header = line.match(/^commit ([0-9a-f]+)$/)
    if (header) {
      commits.push({ id: header[1]!, paths: [] })
      continue
    }
    if (line.trim() === '') continue
    // 状態の列に続く一つまたは二つ（改名と複製）のパス。
    const [, ...paths] = line.split('\t')
    commits.at(-1)?.paths.push(...paths)
  }
  return { revision: resolved, commits }
}

function git(root: string, args: string[]): string {
  const result = Bun.spawnSync(['git', ...args], { cwd: root })
  if (result.exitCode !== 0) {
    throw new Error(`git ${args[0]}: ${result.stderr.toString().trim()}`)
  }
  return result.stdout.toString()
}

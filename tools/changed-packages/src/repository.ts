/** Read the Go package graph and the changed paths of a Git workspace. */

import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { type GoPackage, parseGoList } from './changed-packages.ts'
import { type MiseChange, miseChange } from './mise-change.ts'

function run(root: string, command: string[]): string {
  const result = Bun.spawnSync(command, { cwd: root, stderr: 'pipe' })
  if (result.exitCode !== 0) {
    throw new Error(`${command.join(' ')} failed: ${result.stderr.toString().trim()}`)
  }
  return result.stdout.toString()
}

function paths(output: string): string[] {
  return output.split('\n').filter((line) => line !== '')
}

/** Every package with the dependencies its test binary compiles in. */
export function goPackages(root: string): GoPackage[] {
  const format = '{{.ImportPath}}\t{{.Dir}}\t{{join .Deps ","}}'
  return parseGoList(run(root, ['go', 'list', '-e', '-test', '-f', format, './...']), root)
}

/** Paths that differ between two revisions, old and new names of a rename both included. */
export function changedBetween(root: string, from: string, to: string): string[] {
  return paths(run(root, ['git', 'diff', '--name-only', '--no-renames', from, to, '--']))
}

/**
 * Paths the working tree changes since it left `base`: committed, staged,
 * unstaged, and untracked alike, measured from the merge base so that what
 * `base` gained meanwhile is not counted.
 */
export function changedSince(root: string, base: string): string[] {
  const start = run(root, ['git', 'merge-base', base, 'HEAD']).trim()
  return [
    ...new Set([
      ...paths(run(root, ['git', 'diff', '--name-only', '--no-renames', start, '--'])),
      ...paths(run(root, ['git', 'ls-files', '--others', '--exclude-standard'])),
    ]),
  ].sort()
}

/** The commit a revision names, or an error naming the revision. */
export function commit(root: string, revision: string): string {
  return run(root, [
    'git',
    'rev-parse',
    '--verify',
    '--end-of-options',
    `${revision}^{commit}`,
  ]).trim()
}

export function mergeBase(root: string, left: string, right: string): string {
  return run(root, ['git', 'merge-base', left, right]).trim()
}

/** to を省略した場合は、ステージの状態ではなく実際の作業ツリーを読む。 */
export function readMiseChange(root: string, from: string, to?: string): MiseChange {
  try {
    const before = run(root, ['git', 'show', `${from}:mise.toml`])
    const after = to
      ? run(root, ['git', 'show', `${to}:mise.toml`])
      : readFileSync(join(root, 'mise.toml'), 'utf8')
    return miseChange(before, after)
  } catch {
    return { all: true, reasons: ['mise.toml cannot be read on both sides'], layers: [] }
  }
}

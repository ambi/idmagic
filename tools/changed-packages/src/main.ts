#!/usr/bin/env bun

/**
 * Print the Go packages the working tree changes can break, one per line.
 *
 * The change set is what `git status` reports against HEAD — staged, unstaged,
 * and untracked alike — because the narrow gate is run on a working tree, not
 * on a commit.
 */

import { resolve } from 'node:path'
import { changedGoPackages, parseGoList } from './changed-packages.ts'

const root = resolve(import.meta.dir, '../../..')

function git(...args: string[]): string {
  const result = Bun.spawnSync(['git', ...args], { cwd: root })
  if (result.exitCode !== 0) {
    throw new Error(`git ${args.join(' ')} failed: ${result.stderr.toString()}`)
  }
  return result.stdout.toString()
}

const base = process.argv[2]
const changedFiles = new Set<string>()
for (const line of git('status', '--porcelain', '-z').split('\0')) {
  // "XY <path>", and for a rename the record that follows holds the old path.
  const path = line.length > 3 ? line.slice(3) : ''
  if (path !== '') changedFiles.add(path)
}
if (base !== undefined && base !== '') {
  for (const line of git('diff', '--name-only', `${base}...`).split('\n')) {
    if (line !== '') changedFiles.add(line)
  }
}

const format = [
  '{{.ImportPath}}',
  '{{.Dir}}',
  '{{join .Deps ","}}',
  '{{join .TestImports ","}}',
  '{{join .XTestImports ","}}',
].join('\t')
const list = Bun.spawnSync(['go', 'list', '-e', '-f', format, './...'], { cwd: root })
if (list.exitCode !== 0) {
  process.stderr.write(list.stderr.toString())
  process.exit(list.exitCode)
}
const packages = parseGoList(list.stdout.toString(), root)

for (const importPath of changedGoPackages([...changedFiles], packages)) {
  process.stdout.write(`${importPath}\n`)
}

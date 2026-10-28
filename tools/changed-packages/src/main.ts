#!/usr/bin/env bun

/**
 * Print the Go packages a change can break, one per line.
 *
 * The change set is what the working tree changed since it left the base
 * revision (HEAD when omitted): committed, staged, unstaged, and untracked
 * alike, because the narrow gate is run on a working tree, not on a commit.
 */

import { resolve } from 'node:path'
import { changedGoPackages } from './changed-packages.ts'
import { changedSince, goPackages } from './repository.ts'

const root = resolve(import.meta.dir, '../../..')
const base = process.argv[2] || 'HEAD'

for (const importPath of changedGoPackages(changedSince(root, base), goPackages(root))) {
  process.stdout.write(`${importPath}\n`)
}

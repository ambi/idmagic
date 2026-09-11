#!/usr/bin/env bun

import { resolve } from 'node:path'
import {
  affectedSpecFromChangedWorkItems,
  diffWorkspaceSpecifications,
  formatSpecificationDiff,
  unreferencedStandardChanges,
} from '../../check/src/spec-diff.ts'

const root = resolve(import.meta.dir, '../../..')
const ref = process.argv[2] ?? 'main'
const diff = await diffWorkspaceSpecifications(root, ref)
console.log(formatSpecificationDiff(diff, ref))
const missing = unreferencedStandardChanges(diff, await affectedSpecFromChangedWorkItems(root, ref))
if (missing.length > 0) {
  console.error(
    `standards requirements missing from affected_spec:\n${missing.map((one) => `  ${one}`).join('\n')}`,
  )
  process.exitCode = 1
}

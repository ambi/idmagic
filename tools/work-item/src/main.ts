#!/usr/bin/env bun
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'
import { moveWorkItem } from './move.ts'

const id = process.argv[2]
if (!id || process.argv.length !== 3) {
  console.error('Usage: move-work-item <work-item>')
  process.exit(2)
}
await moveWorkItem(WORKSPACE_ROOT, id)

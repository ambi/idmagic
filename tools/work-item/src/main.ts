#!/usr/bin/env bun
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'
import { moveWorkItem } from './move.ts'

const [id, flag, ...rest] = process.argv.slice(2)
if (!id || (flag !== undefined && flag !== '--cancel') || rest.length > 0) {
  console.error('Usage: move-work-item <work-item> [--cancel]')
  process.exit(2)
}
await moveWorkItem(WORKSPACE_ROOT, id, flag === '--cancel' ? 'cancelled' : 'completed')

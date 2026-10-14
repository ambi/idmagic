#!/usr/bin/env bun
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'
import { archiveWorkItems } from './archive.ts'
import { moveWorkItem } from './move.ts'

const id = process.argv[2]
if (!id || process.argv.length !== 3) {
  console.error('Usage: move-work-item <work-item>')
  process.exit(2)
}
await moveWorkItem(WORKSPACE_ROOT, id)
// 完了のたびに、main に取り込み済みの記録を縮める。移した記録は作業中の変更に含まれるので縮めない。
const archived = await archiveWorkItems(WORKSPACE_ROOT)
if (archived.length > 0) console.info(`archived ${archived.length} completed work item(s)`)

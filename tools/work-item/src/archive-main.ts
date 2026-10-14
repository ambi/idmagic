#!/usr/bin/env bun
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'
import { archiveWorkItems } from './archive.ts'

const archived = await archiveWorkItems(WORKSPACE_ROOT)
console.info(`archived ${archived.length} completed work item(s)`)

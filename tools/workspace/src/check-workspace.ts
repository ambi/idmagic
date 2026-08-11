#!/usr/bin/env bun

import { existsSync } from 'node:fs'
import { readFile, readdir } from 'node:fs/promises'
import { basename, extname, join } from 'node:path'
import {
  type WorkItemDependencyRecord,
  verifyWorkItemDependencies,
} from '../../check/src/work-item-dependencies.ts'
import { loadWorkspaceConfig, rootPath, runTool } from './workspace.ts'

const args = new Set(process.argv.slice(2))
if (args.has('--help') || args.has('-h')) {
  process.stdout.write('Usage: check-workspace [--work-items] [--ids] [--documents]\n')
  process.exit(0)
}
const valid = new Set(['--work-items', '--ids', '--documents'])
for (const arg of args) {
  if (!valid.has(arg)) throw new Error(`unknown option ${arg}`)
}
const all = args.size === 0
const config = await loadWorkspaceConfig()

async function workItemFiles(root: string): Promise<string[]> {
  const result: string[] = []
  for (const dir of [root, join(root, 'done')]) {
    try {
      for (const entry of await readdir(dir, { withFileTypes: true })) {
        if (entry.isFile() && extname(entry.name) === '.md') result.push(join(dir, entry.name))
      }
    } catch {
      // An absent done directory is valid.
    }
  }
  return result.sort()
}

if ((all || args.has('--work-items')) && config.workItems) {
  const files = await workItemFiles(rootPath(config.workItems))
  await runTool(['check/src/main.ts', '--schema=work-item', ...files])
  const records: WorkItemDependencyRecord[] = []
  for (const path of files) {
    const source = await readFile(path, 'utf8')
    const yaml = source.match(/^---\s*\r?\n([\s\S]*?)\r?\n---\s*\r?\n/)?.[1]
    const data = (yaml ? Bun.YAML.parse(yaml) : {}) as {
      status?: unknown
      depends_on?: unknown
      affected_spec?: unknown
    }
    records.push({
      id: basename(path, '.md'),
      path,
      depends_on: Array.isArray(data.depends_on)
        ? data.depends_on.filter((item): item is string => typeof item === 'string')
        : [],
    })

    if (Array.isArray(data.affected_spec)) {
      for (const reference of data.affected_spec) {
        if (!reference || typeof reference !== 'object' || Array.isArray(reference)) continue
        const ref = reference as Record<string, unknown>
        if (typeof ref.path !== 'string') {
          if (data.status === 'pending' || data.status === 'in_progress') {
            console.error(`${path}: active work item contains a legacy specification reference`)
            process.exit(1)
          }
          continue
        }
        const target = rootPath(ref.path)
        if (!existsSync(target)) {
          console.error(`${path}: affected_spec path does not exist: ${ref.path}`)
          process.exit(1)
        }
        const targetSource = await readFile(target, 'utf8')
        if (typeof ref.requirement === 'string' && !targetSource.includes(ref.requirement)) {
          console.error(`${path}: requirement does not resolve in ${ref.path}: ${ref.requirement}`)
          process.exit(1)
        }
        if (typeof ref.symbol === 'string') {
          const name = ref.symbol.split('.').at(-1) ?? ''
          const declaration = new RegExp(`\\b(?:alias|enum|model|op|scalar|union)\\s+${name}\\b`)
          if (!declaration.test(targetSource)) {
            console.error(`${path}: TypeSpec symbol does not resolve in ${ref.path}: ${ref.symbol}`)
            process.exit(1)
          }
        }
      }
    }
  }
  const findings = verifyWorkItemDependencies(records)
  if (findings.length) {
    for (const finding of findings) console.error(`${finding.path}: ${finding.message}`)
    process.exit(1)
  }
  console.log(`ok  ${records.length} work-item dependency record(s)`)
}

if ((all || args.has('--documents')) && config.documents.length > 0) {
  await runTool(['check/src/check-specifications.ts', ...config.documents.map(rootPath)])
}

if ((all || args.has('--ids')) && config.workItems) {
  await runTool(['check/src/check-ids.ts', '--work-items', rootPath(config.workItems)])
}

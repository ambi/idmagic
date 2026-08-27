#!/usr/bin/env bun

import { existsSync, readFileSync } from 'node:fs'
import { readFile, readdir } from 'node:fs/promises'
import { basename, extname, join } from 'node:path'
import { verifyCanonicalDocumentSet } from '../../check/src/canonical-document-set.ts'
import {
  type WorkItemDependencyRecord,
  verifyWorkItemDependencies,
} from '../../check/src/work-item-dependencies.ts'
import {
  type ReferenceEnvironment,
  verifyWorkItemReferences,
} from '../../check/src/work-item-references.ts'
import { listCanonicalDirectories, loadWorkspaceConfig, rootPath, runTool } from './workspace.ts'

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

const repository: ReferenceEnvironment = {
  exists: (path) => existsSync(rootPath(path)),
  read: (path) => {
    try {
      return readFileSync(rootPath(path), 'utf8')
    } catch {
      return undefined
    }
  },
}

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
      initial_context?: unknown
    }
    records.push({
      id: basename(path, '.md'),
      path,
      depends_on: Array.isArray(data.depends_on)
        ? data.depends_on.filter((item): item is string => typeof item === 'string')
        : [],
    })

    for (const finding of verifyWorkItemReferences(data, repository)) {
      console.error(`${path}: ${finding}`)
      process.exit(1)
    }
  }
  const findings = verifyWorkItemDependencies(records)
  if (findings.length) {
    for (const finding of findings) console.error(`${finding.path}: ${finding.message}`)
    process.exit(1)
  }
  console.log(`ok  ${records.length} work-item dependency record(s)`)
}

if (all || args.has('--documents')) {
  // 閉じた集合の強制は、検証すべき文書が 1 件も見つからないときこそ働かなければ
  // ならない。名前を全部打ち間違えた作業ツリーは、集めた結果が空になるという形で
  // 現れる。ここを文書の件数で条件付けると、その最悪の場合だけ検査が飛ぶ。
  const findings = verifyCanonicalDocumentSet(await listCanonicalDirectories())
  if (findings.length) {
    for (const finding of findings) console.error(`fail  ${finding.path}: ${finding.message}`)
    process.exit(1)
  }
  if (config.documents.length > 0) {
    await runTool(['check/src/check-specifications.ts', ...config.documents.map(rootPath)])
  }
}

if ((all || args.has('--ids')) && config.workItems) {
  await runTool(['check/src/check-ids.ts', '--work-items', rootPath(config.workItems)])
}

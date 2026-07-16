#!/usr/bin/env bun
import { existsSync } from 'node:fs'
import { readFile, readdir } from 'node:fs/promises'
import { basename, dirname, extname, join, resolve } from 'node:path'
import {
  collectSclElements,
  parseArchitectureDoc,
  verifyArchitecture,
} from '../../yaml-check/src/arch-check.ts'
import { loadWorkspaceConfig, rootPath, runTool } from './workspace.ts'
import {
  type WorkItemDependencyRecord,
  verifyWorkItemDependencies,
} from '../../yaml-check/src/work-item-dependencies.ts'
import { buildSclWorkspaceIndex } from '../../yaml-check/src/scl-element-reference.ts'

const args = new Set(process.argv.slice(2))
if (args.has('--help') || args.has('-h')) {
  process.stdout.write(
    [
      'Usage: yaml-check-workspace [--work-items] [--scl] [--ids] [--architecture]',
      '',
      'Without flags, runs all discovered checks.',
      '',
    ].join('\n'),
  )
  process.exit(0)
}
const validArgs = new Set(['--work-items', '--scl', '--ids', '--architecture'])
for (const arg of args) {
  if (!validArgs.has(arg)) {
    console.error(`yaml-check-workspace: unknown option ${arg}`)
    process.exit(2)
  }
}
const runAll = args.size === 0
const runWorkItems = runAll || args.has('--work-items')
const runScl = runAll || args.has('--scl')
const runIds = runAll || args.has('--ids')
const runArchitecture = runAll || args.has('--architecture')

const config = await loadWorkspaceConfig()

async function hasMdFiles(dir: string): Promise<boolean> {
  try {
    const entries = await readdir(dir, { withFileTypes: true })
    return entries.some((entry) => entry.isFile() && extname(entry.name) === '.md')
  } catch {
    return false
  }
}

async function hasWorkItems(dir: string): Promise<boolean> {
  return (await hasMdFiles(dir)) || (await hasMdFiles(join(dir, 'done')))
}

const workItemPatterns: string[] = []
const workItemRoots: string[] = []
if (config.repositoryWorkItems && (await hasWorkItems(rootPath(config.repositoryWorkItems)))) {
  workItemRoots.push(rootPath(config.repositoryWorkItems))
  workItemPatterns.push(
    rootPath(`${config.repositoryWorkItems}/*.md`),
    rootPath(`${config.repositoryWorkItems}/done/*.md`),
  )
}
for (const app of config.apps) {
  if (!app.workItems) continue
  if (!(await hasWorkItems(rootPath(app.workItems)))) continue
  workItemRoots.push(rootPath(app.workItems))
  workItemPatterns.push(rootPath(`${app.workItems}/*.md`), rootPath(`${app.workItems}/done/*.md`))
}
if (runWorkItems && workItemPatterns.length > 0) {
  await runTool(['yaml-check/src/main.ts', '--schema=work-item', ...workItemPatterns])
  const records: WorkItemDependencyRecord[] = []
  for (const root of workItemRoots) {
    for (const dir of [root, join(root, 'done')]) {
      for (const path of await listWorkItemFiles(dir)) records.push(await dependencyRecord(path))
    }
  }
  const findings = verifyWorkItemDependencies(records)
  if (findings.length > 0) {
    for (const finding of findings) {
      console.error(`${finding.path}:${finding.line}:${finding.column}: ${finding.message}`)
    }
    process.exit(1)
  }
  console.log(`ok  ${records.length} work-item dependency record(s)`)
}

async function listWorkItemFiles(dir: string): Promise<string[]> {
  try {
    const entries = await readdir(dir, { withFileTypes: true })
    return entries
      .filter((entry) => entry.isFile() && extname(entry.name) === '.md')
      .map((entry) => join(dir, entry.name))
      .sort()
  } catch {
    return []
  }
}

async function dependencyRecord(path: string): Promise<WorkItemDependencyRecord> {
  const text = await readFile(path, 'utf8')
  const frontmatter = text.match(/^---\s*\r?\n([\s\S]*?)\r?\n---\s*\r?\n/)
  const data = (frontmatter?.[1] ? Bun.YAML.parse(frontmatter[1]) : {}) as {
    depends_on?: unknown
  }
  const depends_on = Array.isArray(data?.depends_on)
    ? data.depends_on.filter((id: unknown): id is string => typeof id === 'string')
    : []
  const depends_on_line = text.split('\n').findIndex((line) => /^depends_on\s*:/.test(line)) + 1
  return {
    id: basename(path, '.md'),
    path,
    depends_on,
    ...(depends_on_line > 0 ? { depends_on_line } : {}),
  }
}

const sclPatterns: string[] = []
for (const app of config.apps) {
  sclPatterns.push(rootPath(app.scl))
  if (app.contextGlob) sclPatterns.push(rootPath(app.contextGlob))
}
for (const spec of config.toolSpecs ?? []) sclPatterns.push(rootPath(spec))
if (runScl && sclPatterns.length > 0) {
  await runTool(['yaml-check/src/main.ts', '--schema=scl', ...sclPatterns])

  let referenceIndexFailed = false
  for (const app of config.apps) {
    const rootFile = rootPath(app.scl)
    const root = Bun.YAML.parse(await readFile(rootFile, 'utf8')) as Record<string, unknown>
    if (!root.context_map || typeof root.context_map !== 'object') continue
    const documents: Record<string, unknown> = {}
    for (const [context, entryValue] of Object.entries(
      root.context_map as Record<string, unknown>,
    )) {
      const path = (entryValue as { path?: unknown } | null)?.path
      if (typeof path !== 'string') continue
      try {
        documents[context] = Bun.YAML.parse(
          await readFile(resolve(dirname(rootFile), path), 'utf8'),
        )
      } catch {
        // The index reports the unavailable context with its stable context name.
      }
    }
    const built = buildSclWorkspaceIndex(root, documents)
    if (built.ok) {
      console.log(`ok  ${app.scl} (${built.index.contexts.size} SCL reference contexts)`)
      continue
    }
    referenceIndexFailed = true
    for (const finding of built.errors) {
      console.error(`${app.scl}: scl-element-reference: ${finding.message}`)
    }
  }
  if (referenceIndexFailed) process.exit(1)
}

const checkIdsArgs: string[] = ['yaml-check/src/check-ids.ts']
if (config.repositoryWorkItems)
  checkIdsArgs.push('--work-items', rootPath(config.repositoryWorkItems))
for (const app of config.apps) {
  if (app.workItems) checkIdsArgs.push('--work-items', rootPath(app.workItems))
}
for (const app of config.apps) {
  if (app.decisions) checkIdsArgs.push('--decisions', rootPath(app.decisions))
}
if (runIds && checkIdsArgs.length > 1) await runTool(checkIdsArgs)

// Architecture maps: schema-validate the frontmatter, then cross-check that the
// map has not drifted from the workspace it describes (ARCHITECTURE_FORMAT.md §4).
async function collectWorkspaceSclElements(): Promise<Set<string>> {
  const set = new Set<string>()
  const sclFiles: string[] = []
  for (const app of config.apps) {
    sclFiles.push(rootPath(app.scl))
    if (app.contextGlob) {
      const glob = new Bun.Glob(app.contextGlob)
      for await (const match of glob.scan({ cwd: rootPath('.'), absolute: true })) {
        sclFiles.push(match)
      }
    }
  }
  for (const spec of config.toolSpecs ?? []) sclFiles.push(rootPath(spec))
  for (const file of sclFiles) {
    try {
      const doc = Bun.YAML.parse(await readFile(file, 'utf8'))
      for (const ref of collectSclElements(doc)) set.add(ref)
    } catch {
      // A malformed SCL file is reported by the --scl pass above; skip it here.
    }
  }
  return set
}

const architectureDocs = config.architectureDocs ?? []
if (runArchitecture && architectureDocs.length > 0) {
  await runTool([
    'yaml-check/src/main.ts',
    '--schema=architecture',
    ...architectureDocs.map((doc) => rootPath(doc)),
  ])

  const sclElements = await collectWorkspaceSclElements()
  let archFailed = 0
  for (const rel of architectureDocs) {
    const abs = rootPath(rel)
    const text = await readFile(abs, 'utf8')
    const data = parseArchitectureDoc(text)
    const { errors } = verifyArchitecture(data, {
      text,
      archDir: dirname(abs),
      workspaceRoot: rootPath('.'),
      sclElements,
      pathExists: existsSync,
      join,
    })
    if (errors.length === 0) {
      console.log(`ok  ${rel} (architecture cross-check)`)
      continue
    }
    archFailed++
    console.log(`FAIL ${rel}`)
    for (const e of errors) console.log(`${rel}:${e.line}:${e.column}: ${e.message}`)
  }
  if (archFailed > 0) {
    console.error(`\n${archFailed} architecture doc(s) failed cross-check.`)
    process.exit(1)
  }
}

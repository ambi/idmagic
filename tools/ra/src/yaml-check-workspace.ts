#!/usr/bin/env bun
import { existsSync } from 'node:fs'
import { readFile, readdir } from 'node:fs/promises'
import { dirname, extname, join } from 'node:path'
import {
  collectSclElements,
  parseArchitectureDoc,
  verifyArchitecture,
} from '../../yaml-check/src/arch-check.ts'
import { loadWorkspaceConfig, rootPath, runTool } from './workspace.ts'

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
if (config.repositoryWorkItems && (await hasWorkItems(rootPath(config.repositoryWorkItems)))) {
  workItemPatterns.push(
    rootPath(`${config.repositoryWorkItems}/*.md`),
    rootPath(`${config.repositoryWorkItems}/done/*.md`),
  )
}
for (const app of config.apps) {
  if (!app.workItems) continue
  if (!(await hasWorkItems(rootPath(app.workItems)))) continue
  workItemPatterns.push(rootPath(`${app.workItems}/*.md`), rootPath(`${app.workItems}/done/*.md`))
}
if (runWorkItems && workItemPatterns.length > 0) {
  await runTool(['yaml-check/src/main.ts', '--schema=work-item', ...workItemPatterns])
}

const sclPatterns: string[] = []
for (const app of config.apps) {
  sclPatterns.push(rootPath(app.scl))
  if (app.contextGlob) sclPatterns.push(rootPath(app.contextGlob))
}
for (const spec of config.toolSpecs ?? []) sclPatterns.push(rootPath(spec))
if (runScl && sclPatterns.length > 0) {
  await runTool(['yaml-check/src/main.ts', '--schema=scl', ...sclPatterns])
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

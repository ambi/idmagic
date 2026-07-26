#!/usr/bin/env bun
import { existsSync } from 'node:fs'
import { readFile, readdir } from 'node:fs/promises'
import { basename, dirname, extname, join, relative, resolve } from 'node:path'
import { parseArchitectureDoc, verifyArchitecture } from '../../check/src/arch-check.ts'
import { loadWorkspaceConfig, rootPath, runTool } from './workspace.ts'
import {
  type WorkItemDependencyRecord,
  verifyWorkItemDependencies,
} from '../../check/src/work-item-dependencies.ts'
import { buildSclWorkspaceIndex } from '../../check/src/scl-element-reference.ts'
import { resolveSclElementReference } from '../../check/src/scl-element-reference.ts'
import { loadWorkspaceSclIndex } from './workspace-scl-index.ts'
import {
  collectArchitectureWorkspace,
  evaluateArchitectureWorkspace,
} from './architecture-workspace.ts'
import {
  type ArchitectureLedgerFile,
  ledgerDirectory,
  mergeArchitectureLedgers,
} from './architecture-ledger.ts'

type Dict = Record<string, unknown>

const args = new Set(process.argv.slice(2))
if (args.has('--help') || args.has('-h')) {
  process.stdout.write(
    [
      'Usage: check-workspace [--work-items] [--scl] [--ids] [--architecture] [--traceability]',
      '',
      'Without flags, runs all discovered checks.',
      '',
    ].join('\n'),
  )
  process.exit(0)
}
const validArgs = new Set(['--work-items', '--scl', '--ids', '--architecture', '--traceability'])
for (const arg of args) {
  if (!validArgs.has(arg)) {
    console.error(`check-workspace: unknown option ${arg}`)
    process.exit(2)
  }
}
const runAll = args.size === 0
const runWorkItems = runAll || args.has('--work-items')
const runScl = runAll || args.has('--scl')
const runIds = runAll || args.has('--ids')
const runArchitecture = runAll || args.has('--architecture')
const runTraceability = runAll || args.has('--traceability')

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
  await runTool(['check/src/main.ts', '--schema=work-item', ...workItemPatterns])
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

  const { index, errors } = await loadWorkspaceSclIndex(config)
  if (errors.length > 0) {
    for (const error of errors) console.error(`scl-element-reference: ${error.message}`)
    process.exit(1)
  }
  let affectedSpecFailed = false
  for (const root of workItemRoots) {
    for (const dir of [root, join(root, 'done')]) {
      for (const path of await listWorkItemFiles(dir)) {
        const text = await readFile(path, 'utf8')
        const yaml = text.match(/^---\s*\r?\n([\s\S]*?)\r?\n---\s*\r?\n/)?.[1]
        if (!yaml) continue
        const frontmatter = Bun.YAML.parse(yaml) as { affected_spec?: unknown }
        if (!Array.isArray(frontmatter.affected_spec)) continue
        for (const target of frontmatter.affected_spec) {
          const resolved = resolveSclElementReference(index, target)
          if (resolved.ok) continue
          affectedSpecFailed = true
          console.error(`${path}: affected_spec: ${resolved.error.message}`)
        }
      }
    }
  }
  if (affectedSpecFailed) process.exit(1)
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
  await runTool(['check/src/main.ts', '--schema=scl', ...sclPatterns])

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

if (runTraceability && config.verificationManifest) {
  await runTool([
    'check/src/main.ts',
    '--schema=verification-manifest',
    rootPath(config.verificationManifest),
  ])
  if (config.verificationEvidence) {
    await runTool([
      'check/src/main.ts',
      '--schema=verification-evidence',
      rootPath(config.verificationEvidence),
    ])
  }
}

const checkIdsArgs: string[] = ['check/src/check-ids.ts']
if (config.repositoryWorkItems)
  checkIdsArgs.push('--work-items', rootPath(config.repositoryWorkItems))
for (const app of config.apps) {
  if (app.workItems) checkIdsArgs.push('--work-items', rootPath(app.workItems))
}
for (const app of config.apps) {
  if (app.decisions) checkIdsArgs.push('--decisions', rootPath(app.decisions))
}
// Design records and work items cite ADRs by path; verify those paths resolve.
// Work-item files come from the --work-items directories already passed above.
checkIdsArgs.push('--root', rootPath('.'))
const architectureDocPaths = (config.architectureDocs ?? []).map((doc) => rootPath(doc))
if (architectureDocPaths.length > 0) checkIdsArgs.push('--links', ...architectureDocPaths)
if (runIds && checkIdsArgs.length > 1) await runTool(checkIdsArgs)

// Architecture maps: schema-validate the frontmatter, then cross-check that the
// map has not drifted from the workspace it describes (ARCHITECTURE_FORMAT.md §4).
async function collectExpectedArchitectureContexts(): Promise<Map<string, string>> {
  const contexts = new Map<string, string>()
  for (const app of config.apps) {
    const rootFile = rootPath(app.scl)
    const root = Bun.YAML.parse(await readFile(rootFile, 'utf8')) as Record<string, unknown>
    const contextMap = root.context_map as Record<string, unknown> | undefined
    if (contextMap && Object.keys(contextMap).length > 0) {
      for (const [context, entry] of Object.entries(contextMap)) {
        const path = (entry as { path?: unknown } | null)?.path
        if (typeof path !== 'string') continue
        contexts.set(context, relative(rootPath('.'), resolve(dirname(rootFile), path)))
      }
      continue
    }
    const context = root.context ?? root.system
    if (typeof context === 'string') contexts.set(context, app.scl)
  }
  for (const spec of config.toolSpecs ?? []) {
    const document = Bun.YAML.parse(await readFile(rootPath(spec), 'utf8')) as Record<
      string,
      unknown
    >
    const context = document.context ?? document.system
    if (typeof context === 'string') contexts.set(context, spec)
  }
  return contexts
}

const architectureDocs = config.architectureDocs ?? []
const architectureLedgers = config.architectureLedgers ?? []
if (runArchitecture && (architectureDocs.length > 0 || architectureLedgers.length > 0)) {
  if (architectureLedgers.length > 0) {
    await runTool([
      'check/src/main.ts',
      '--schema=architecture-map',
      ...architectureLedgers.map((ledger) => rootPath(ledger)),
    ])
  }
  if (architectureDocs.length > 0) {
    await runTool([
      'check/src/main.ts',
      '--schema=architecture-doc',
      ...architectureDocs.map((doc) => rootPath(doc)),
    ])
  }

  const { index: sclIndex, errors: sclIndexErrors } = await loadWorkspaceSclIndex(config)
  if (sclIndexErrors.length > 0) {
    for (const error of sclIndexErrors) {
      console.error(`scl-element-reference: ${error.message}`)
    }
    process.exit(1)
  }

  // Ledgers merge into one workspace-rooted map so the cross-checks below see a
  // single module graph regardless of how many files declare it (ADR-143).
  const ledgerFiles: ArchitectureLedgerFile[] = []
  for (const rel of architectureLedgers) {
    ledgerFiles.push({
      path: rel,
      doc: Bun.YAML.parse(await readFile(rootPath(rel), 'utf8')),
    })
  }
  const { doc: architectureMap, findings: ledgerFindings } = mergeArchitectureLedgers(ledgerFiles)

  const expectedContexts = await collectExpectedArchitectureContexts()
  const architectureWorkspace = await collectArchitectureWorkspace(rootPath('.'))
  const { errors: mapErrors } = verifyArchitecture(architectureMap, {
    archDir: rootPath('.'),
    workspaceRoot: rootPath('.'),
    sclIndex,
    expectedContexts,
    pathExists: existsSync,
    join,
  })
  const workspaceFindings = evaluateArchitectureWorkspace(architectureMap, architectureWorkspace)

  let archFailed = 0
  if (ledgerFindings.length > 0 || mapErrors.length > 0 || workspaceFindings.length > 0) {
    archFailed++
    console.log('FAIL architecture map')
    for (const finding of ledgerFindings) console.log(`${finding.path}:1:1: ${finding.message}`)
    for (const e of mapErrors) console.log(`architecture.yaml:${e.line}:${e.column}: ${e.message}`)
    for (const finding of workspaceFindings) {
      console.log(`architecture.yaml:1:1: ${finding.message} (${finding.path})`)
    }
  } else {
    const moduleCount = Object.keys(architectureMap.modules ?? {}).length
    console.log(
      `ok  ${architectureLedgers.length} architecture ledger(s), ${moduleCount} module(s) (cross-check)`,
    )
  }

  // A design record and the ledger beside it describe the same boundary, so
  // their context prefixes must agree (ARCHITECTURE_FORMAT.md §1).
  const ledgerContextByDir = new Map<string, unknown>()
  for (const ledger of ledgerFiles) {
    ledgerContextByDir.set(ledgerDirectory(ledger.path), (ledger.doc as Dict | null)?.context)
  }
  for (const rel of architectureDocs) {
    const dir = ledgerDirectory(rel)
    const data = parseArchitectureDoc(await readFile(rootPath(rel), 'utf8'))
    if (!ledgerContextByDir.has(dir)) {
      archFailed++
      console.log(`${rel}:1:1: architecture: no architecture.yaml beside this design record`)
      continue
    }
    const ledgerContext = ledgerContextByDir.get(dir)
    if (data.context !== ledgerContext) {
      archFailed++
      console.log(
        `${rel}:1:1: architecture: context '${String(data.context)}' does not match '${String(ledgerContext)}' in ${dir === '.' ? 'architecture.yaml' : `${dir}/architecture.yaml`}`,
      )
      continue
    }
    console.log(`ok  ${rel} (design record)`)
  }

  if (archFailed > 0) {
    console.error('\narchitecture cross-check failed.')
    process.exit(1)
  }
}

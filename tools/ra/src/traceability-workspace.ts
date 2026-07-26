#!/usr/bin/env bun
import { existsSync } from 'node:fs'
import { readFileSync } from 'node:fs'
import { readFile as readFileAsync } from 'node:fs/promises'
import { type ArchitectureLedgerFile, mergeArchitectureLedgers } from './architecture-ledger.ts'
import {
  buildTraceabilityReport,
  type TraceabilityEvidence,
  type TraceabilityManifest,
} from './traceability.ts'
import { loadWorkspaceConfig, rootPath } from './workspace.ts'
import { loadWorkspaceSclIndex } from './workspace-scl-index.ts'

const args = process.argv.slice(2)
const strict = args.includes('--strict')
const json = args.includes('--json')
const revisionArg = args.find((arg) => arg.startsWith('--revision='))
const sourceRevision = revisionArg?.slice('--revision='.length) || 'working-tree'
for (const arg of args) {
  if (arg !== '--strict' && arg !== '--json' && !arg.startsWith('--revision=')) {
    console.error(`traceability: unknown option ${arg}`)
    process.exit(2)
  }
}

const config = await loadWorkspaceConfig()
if (!config.verificationManifest) {
  console.error('traceability: verification/manifest.yaml was not discovered')
  process.exit(2)
}
const manifest = Bun.YAML.parse(
  await readFileAsync(rootPath(config.verificationManifest), 'utf8'),
) as TraceabilityManifest
let evidence: TraceabilityEvidence | undefined
if (config.verificationEvidence && existsSync(rootPath(config.verificationEvidence))) {
  evidence = Bun.YAML.parse(
    await readFileAsync(rootPath(config.verificationEvidence), 'utf8'),
  ) as TraceabilityEvidence
}

const { index, errors } = await loadWorkspaceSclIndex(config)
if (errors.length > 0) {
  for (const error of errors) console.error(`traceability: ${error.message}`)
  process.exit(1)
}
// Modules are declared by the ledgers, not by the prose design record (ADR-143).
// Several ledgers merge into one workspace-rooted module graph, exactly as
// `ra check --architecture` sees it.
const architectureLedgers = config.architectureLedgers ?? []
if (architectureLedgers.length === 0) {
  console.error('traceability: no architecture.yaml ledger was discovered')
  process.exit(2)
}
const ledgerFiles: ArchitectureLedgerFile[] = []
for (const path of architectureLedgers) {
  ledgerFiles.push({ path, doc: Bun.YAML.parse(await readFileAsync(rootPath(path), 'utf8')) })
}
const { doc: architectureMap } = mergeArchitectureLedgers(ledgerFiles)
const modules = new Set(
  Object.keys((architectureMap.modules as Record<string, unknown> | undefined) ?? {}),
)
const report = buildTraceabilityReport({
  manifest,
  evidence,
  index,
  architectureModules: modules,
  sourceRevision,
  strict,
  availableRecipes: new Set(
    readFileSync(rootPath('justfile'), 'utf8')
      .split('\n')
      .flatMap((line) => line.match(/^([a-zA-Z0-9_-]+)(?:\s+[^:]*)?:/)?.[1] ?? []),
  ),
  implementationContains: (path, symbol) => {
    try {
      return readFileSync(rootPath(path), 'utf8').includes(symbol)
    } catch {
      return false
    }
  },
})

if (json) {
  process.stdout.write(`${JSON.stringify(report, null, 2)}\n`)
} else {
  for (const finding of report.findings) {
    const baseline = finding.baseline ? ` [baseline: ${finding.baseline}]` : ''
    console.log(`${finding.code}: ${finding.target}: ${finding.detail}${baseline}`)
  }
  console.log(
    `traceability: ${report.passed ? 'passed' : 'failed'} (${report.findings.length} finding(s))`,
  )
}
process.exit(report.passed ? 0 : 1)

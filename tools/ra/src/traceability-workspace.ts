#!/usr/bin/env bun
import { existsSync } from 'node:fs'
import { readFileSync } from 'node:fs'
import { readFile as readFileAsync } from 'node:fs/promises'
import { parseArchitectureDoc } from '../../yaml-check/src/arch-check.ts'
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
const modules = new Set<string>()
for (const path of config.architectureDocs ?? []) {
  const doc = parseArchitectureDoc(await readFileAsync(rootPath(path), 'utf8'))
  for (const id of Object.keys((doc.modules as Record<string, unknown> | undefined) ?? {})) {
    modules.add(id)
  }
}
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

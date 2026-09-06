#!/usr/bin/env bun

/**
 * Entry point for the security-control rules. See security-controls.ts for what
 * each rule is for; this file only gathers the inputs from the working tree.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import {
  browserRefusalTypesNamedByPlatformScenario,
  checkContractRefusalsAreDeclared,
  checkSecurityGuards,
  contractRefusalsOfStateChanges,
  errorTypesNamedByScenarios,
  insufficientScopeTypeNamedByApiTokenScenario,
  type Finding,
  type GoFile,
} from './security-controls.ts'

const root = resolve(import.meta.dir, '../../..')
const excluded = new Set(['.git', 'node_modules', 'vendor', 'dist', 'build', 'generated'])

async function walk(dir: string, result: string[] = []): Promise<string[]> {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.isDirectory() && excluded.has(entry.name)) continue
    const path = resolve(dir, entry.name)
    if (entry.isDirectory()) await walk(path, result)
    else if (entry.isFile()) result.push(path)
  }
  return result
}

const goPaths = (await walk(resolve(root, 'backend'))).filter((path) => path.endsWith('.go'))
const goFiles: GoFile[] = await Promise.all(
  goPaths.map(async (path) => ({
    path: relative(root, path),
    source: await readFile(path, 'utf8'),
  })),
)

const findings: Finding[] = [...checkSecurityGuards(goFiles)]

// R4 reads the scenarios against the 403 responses TypeSpec declares. Prose and
// contract live in mirrored trees: scenarios under docs/contexts, TypeSpec
// under spec/contexts. Reading both from one of them makes R4 check nothing and
// still report success.
const contextsDir = resolve(root, 'docs/contexts')
const contractDir = resolve(root, 'spec/contexts')
const platformScenarios = await readFile(resolve(root, 'docs/scenarios.feature.md'), 'utf8')
const platformRefusals = browserRefusalTypesNamedByPlatformScenario(platformScenarios)
const apiTokenScenarios = await readFile(
  resolve(contextsDir, 'api-tokens/scenarios.feature.md'),
  'utf8',
)
const apiTokenRefusals = insufficientScopeTypeNamedByApiTokenScenario(apiTokenScenarios)
const sharedRefusals = new Set([...platformRefusals, ...apiTokenRefusals])
let declared = sharedRefusals.size
let promised = 0
for (const context of await readdir(contextsDir)) {
  const dir = resolve(contextsDir, context)
  const source = await readFile(resolve(dir, 'scenarios.feature.md'), 'utf8').catch(() => undefined)
  if (!source) continue
  const local = errorTypesNamedByScenarios(source)
  const named = new Set([...local, ...sharedRefusals])
  declared += local.size

  const contract = new Map<string, string[]>()
  const contractFiles = await readdir(resolve(contractDir, context)).catch(() => [])
  for (const entry of contractFiles) {
    if (!entry.endsWith('.tsp')) continue
    const typespec = await readFile(resolve(contractDir, context, entry), 'utf8')
    for (const [type, operations] of contractRefusalsOfStateChanges(typespec)) {
      contract.set(type, [...(contract.get(type) ?? []), ...operations])
    }
  }
  promised += contract.size
  findings.push(...checkContractRefusalsAreDeclared(context, contract, named))
}

for (const finding of findings) {
  console.error(`${finding.path}: [${finding.rule}] ${finding.message}`)
}
if (findings.length > 0) process.exit(1)
console.log(
  `ok  security controls (${promised} refusal(s) promised by a 403 on a state change, ` +
    `${declared} error type(s) named by the scenarios)`,
)

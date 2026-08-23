#!/usr/bin/env bun

/**
 * Entry point for the security-control rules. See security-controls.ts for what
 * each rule is for; this file only gathers the inputs from the working tree.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import {
  checkContractRefusalsAreDeclared,
  checkRefusalCoverage,
  checkSecurityGuards,
  contractRefusalsOfStateChanges,
  declaredRefusalTypes,
  type Finding,
  type GoFile,
  refusalScenarioIds,
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

// R3 reads the refusals the specification declares and the ids the tests name.
// R4 reads the same scenarios against the 403 responses TypeSpec declares.
const contextsDir = resolve(root, 'docs/contexts')
const declared: string[] = []
let promised = 0
for (const context of await readdir(contextsDir)) {
  const dir = resolve(contextsDir, context)
  const source = await readFile(resolve(dir, 'scenarios.md'), 'utf8').catch(() => undefined)
  if (!source) continue
  declared.push(...refusalScenarioIds(source))

  const contract = new Map<string, string[]>()
  for (const entry of await readdir(dir)) {
    if (!entry.endsWith('.tsp')) continue
    const typespec = await readFile(resolve(dir, entry), 'utf8')
    for (const [type, operations] of contractRefusalsOfStateChanges(typespec)) {
      contract.set(type, [...(contract.get(type) ?? []), ...operations])
    }
  }
  promised += contract.size
  findings.push(...checkContractRefusalsAreDeclared(context, contract, declaredRefusalTypes(source)))
}

const cited = new Set<string>()
for (const file of goFiles) {
  if (!file.path.endsWith('_test.go')) continue
  for (const match of file.source.matchAll(/REQ-[A-Z0-9]+-\d+/g)) cited.add(match[0])
}

const debtPath = resolve(root, 'tools/check/security-refusal-debt.json')
const debt = JSON.parse(await readFile(debtPath, 'utf8')) as { untested: string[] }
findings.push(...checkRefusalCoverage(declared, cited, debt.untested))

for (const finding of findings) {
  console.error(`${finding.path}: [${finding.rule}] ${finding.message}`)
}
if (findings.length > 0) process.exit(1)
console.log(
  `ok  security controls (${declared.length} declared refusal(s), ${promised} promised by a 403 on a ` +
    `state change, ${debt.untested.length} awaiting a test)`,
)

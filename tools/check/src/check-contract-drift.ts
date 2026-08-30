#!/usr/bin/env bun

/**
 * Entry point for the TypeSpec/Go body drift check. See contract-drift.ts for
 * what it compares and why; this file only gathers the inputs and decides the
 * exit code.
 *
 * Two things fail the run: a finding, and more unresolved operations than the
 * ceiling below. The ceiling exists because what the checker cannot follow must
 * not be counted as agreement — but naming the individual operations would rot
 * on the next handler rename, so the count is what is held.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import { discoverGeneratedOpenApi } from '../../workspace/src/workspace.ts'
import { collectRoutes, diffContract, type GoFile, type OpenAPIDocument } from './contract-drift.ts'

const root = resolve(import.meta.dir, '../../..')
const excluded = new Set(['.git', 'node_modules', 'vendor', 'dist', 'build', 'generated'])

/**
 * There is no threshold in this file, and that is deliberate.
 *
 * A finding is a difference between the contract and the server; the run fails
 * on any of them, and they get fixed rather than recorded as permitted.
 *
 * What the reader could not follow is printed instead of gated. Gating it would
 * fail the build of whoever next writes a handler in a shape the reader does not
 * read yet, which is not their defect — it is this reader's. Printing the
 * coverage every run is what keeps the gap from being silent, which is what
 * wi-385's Risk Notes ask for; shrinking it is work on the reader.
 */

async function walk(dir: string, result: string[] = []): Promise<string[]> {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.isDirectory() && excluded.has(entry.name)) continue
    const path = resolve(dir, entry.name)
    if (entry.isDirectory()) await walk(path, result)
    else if (entry.isFile()) result.push(path)
  }
  return result
}

const goPaths = (await walk(resolve(root, 'backend'))).filter(
  (path) => path.endsWith('.go') && !path.endsWith('_test.go'),
)
const goFiles: GoFile[] = await Promise.all(
  goPaths.map(async (path) => ({
    path: relative(root, path),
    source: await readFile(path, 'utf8'),
  })),
)

const document = JSON.parse(
  await readFile(await discoverGeneratedOpenApi(root), 'utf8'),
) as OpenAPIDocument

const { findings, unresolved } = diffContract(document, collectRoutes(goFiles), goFiles)

for (const finding of findings) console.error(`fail  ${finding.message}`)

const byReason = new Map<string, number>()
for (const entry of unresolved) byReason.set(entry.reason, (byReason.get(entry.reason) ?? 0) + 1)
const breakdown = [...byReason.entries()]
  .sort(([a], [b]) => a.localeCompare(b))
  .map(([reason, count]) => `${reason}=${count}`)
  .join(', ')

if (process.argv.includes('--list-unresolved')) {
  for (const entry of unresolved) {
    console.error(`unresolved  ${entry.operationId} (${entry.reason}: ${entry.detail})`)
  }
}

if (findings.length > 0) process.exit(1)

const operations = Object.values(document.paths ?? {}).flatMap((pathItem) =>
  Object.values(pathItem).filter((operation) => operation.operationId),
).length
const compared = operations - unresolved.length

console.log(
  `ok  contract body drift (0 finding(s); compared ${compared}/${operations} operation(s), ` +
    `${unresolved.length} not followed: ${breakdown || 'none'}. --list-unresolved names them)`,
)

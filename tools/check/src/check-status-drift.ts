#!/usr/bin/env bun

/**
 * Entry point for the declared-status-code check. See status-drift.ts for what
 * it compares and why; this file gathers the inputs and decides the exit code.
 *
 * As with check-contract-drift.ts there is no threshold here. A finding is a
 * difference between what the contract declares and what the server writes, and
 * it gets fixed rather than recorded as permitted. What the reader could not
 * follow is printed instead of gated: failing the build of whoever next writes a
 * handler in a shape this reader does not read yet would be charging them for
 * this reader's gap. Printing the coverage every run is what keeps that gap from
 * being silent.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import { discoverGeneratedOpenApi } from '../../workspace/src/workspace.ts'
import type { GoFile } from './contract-drift.ts'
import { diffStatusCodes, type OpenAPIDocument } from './status-drift.ts'

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

const { findings, unresolved, unread } = diffStatusCodes(document, goFiles)

for (const finding of findings) console.error(`fail  ${finding.message}`)

if (process.argv.includes('--list-unresolved')) {
  for (const entry of unresolved) {
    console.error(`unresolved  ${entry.operationId} (${entry.reason}: ${entry.detail})`)
  }
  for (const entry of unread) {
    console.error(`partial     ${entry.operationId} (not read: ${entry.writers.join(', ')})`)
  }
}

if (findings.length > 0) process.exit(1)

const operations = Object.values(document.paths ?? {}).flatMap((pathItem) =>
  Object.values(pathItem).filter((operation) => operation.operationId),
).length

const byReason = new Map<string, number>()
for (const entry of unresolved) byReason.set(entry.reason, (byReason.get(entry.reason) ?? 0) + 1)
const breakdown = [...byReason.entries()]
  .sort(([a], [b]) => a.localeCompare(b))
  .map(([reason, count]) => `${reason}=${count}`)
  .join(', ')

// Three populations, not two. An operation whose writers were all read answers
// both rules; one that called an error mapper answers only the first; one whose
// route or handler was not found answers neither.
const fully = operations - unresolved.length - unread.length
console.log(
  `ok  declared status codes (0 finding(s); ${fully}/${operations} operation(s) read in full, ` +
    `${unread.length} read in part, ${unresolved.length} not reached${breakdown ? `: ${breakdown}` : ''}. ` +
    '--list-unresolved names them)',
)

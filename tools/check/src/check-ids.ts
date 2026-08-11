#!/usr/bin/env bun
/**
 * Guard against duplicate work-item ids (WORK_ITEM_FORMAT.md).
 *
 *   check-ids --work-items <dir>...
 *
 * For each `--work-items` directory it scans the directory and its `done/`
 * subdirectory (both share one id namespace, the filename stem). Within each
 * namespace, an id used by more than one record fails the run.
 *
 * Pure logic lives in `./record-ids.ts`; this file is the CLI shell only.
 * Exits non-zero when any duplicate id is found.
 */

import { readdir } from 'node:fs/promises'
import { isAbsolute, join, relative, resolve } from 'node:path'
import { type IdFinding, type RecordRef, findDuplicates, workItemRef } from './record-ids.ts'

const DONE_SUBDIR = 'done'

function resolveDir(p: string): string {
  if (isAbsolute(p)) return p
  return resolve(process.cwd(), p)
}

async function listFiles(dir: string, ext: string): Promise<string[]> {
  try {
    const entries = await readdir(dir, { withFileTypes: true })
    return entries.filter((e) => e.isFile() && e.name.endsWith(ext)).map((e) => join(dir, e.name))
  } catch {
    return []
  }
}

async function collectWorkItems(
  dir: string,
): Promise<{ refs: RecordRef[]; findings: IdFinding[] }> {
  const namespace = resolveDir(dir)
  const files = [
    ...(await listFiles(namespace, '.md')),
    ...(await listFiles(join(namespace, DONE_SUBDIR), '.md')),
  ]
  const refs: RecordRef[] = []
  const findings: IdFinding[] = []
  for (const path of files) {
    const result = workItemRef(path, namespace)
    if (result.ref) refs.push(result.ref)
    findings.push(...result.findings)
  }
  return { refs, findings }
}

function parseArgs(argv: string[]): { workItems: string[] } {
  const workItems: string[] = []
  let bucket: string[] | null = null
  for (const arg of argv) {
    if (arg === '--work-items') bucket = workItems
    else if (bucket) bucket.push(arg)
    else throw new Error(`unexpected argument '${arg}' (expected --work-items first)`)
  }
  return { workItems }
}

const { workItems } = parseArgs(process.argv.slice(2))
if (workItems.length === 0) {
  console.error('check-ids: nothing to check — pass --work-items <dir>')
  process.exit(2)
}

const refs: RecordRef[] = []
const findings: IdFinding[] = []
for (const dir of workItems) {
  const r = await collectWorkItems(dir)
  refs.push(...r.refs)
  findings.push(...r.findings)
}
findings.push(...findDuplicates(refs))

if (findings.length > 0) {
  for (const f of findings) {
    const rel = relative(process.cwd(), f.path) || f.path
    console.error(`${rel}: ${f.message}`)
  }
  console.error(`\n${findings.length} id problem(s) found across ${refs.length} record(s).`)
  process.exit(1)
}
console.error(`All ${refs.length} record id(s) OK.`)

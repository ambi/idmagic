#!/usr/bin/env bun
/**
 * Guard against duplicate work-item and ADR ids (WORK_ITEM_FORMAT.md / ADR_FORMAT.md).
 *
 *   check-ids --work-items <dir>... --decisions <dir>...
 *
 * For each `--work-items` directory it scans the directory and its `done/`
 * subdirectory (both share one id namespace, the filename stem). For each
 * `--decisions` directory it scans the ADR markdown files. Within each
 * namespace, an id used by more than one record fails the run. CONCEPTION*.md
 * and other non-ADR files are ignored.
 *
 * Pure logic lives in `./record-ids.ts`; this file is the CLI shell only.
 * Exits non-zero when any duplicate or malformed ADR filename is found.
 */

import { readFile, readdir } from 'node:fs/promises'
import { isAbsolute, join, relative, resolve } from 'node:path'
import {
  type AdrSupersessionRecord,
  checkAdrSupersession,
  extractAdrIds,
} from './adr-supersession.ts'
import {
  type IdFinding,
  type RecordRef,
  adrRef,
  findDuplicates,
  workItemRef,
} from './record-ids.ts'

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

/** A directory holds ADRs when a file matches `*ADR-N*.md`; CONCEPTION files
 *  and plain markdown are skipped so they are never treated as records. */
const ADR_CANDIDATE_RE = /ADR-\d/i

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

async function adrSupersessionRecord(path: string, id: string): Promise<AdrSupersessionRecord> {
  const text = await readFile(path, 'utf8')
  const yaml = text.match(/^---\s*\r?\n([\s\S]*?)\r?\n---\s*\r?\n/)?.[1]
  const fm = (yaml ? Bun.YAML.parse(yaml) : {}) as {
    status?: unknown
    supersedes?: unknown
    superseded_by?: unknown
  }
  return {
    id,
    path,
    status: typeof fm.status === 'string' ? fm.status : undefined,
    supersedes: extractAdrIds(fm.supersedes),
    supersededBy: extractAdrIds(fm.superseded_by),
  }
}

async function collectAdrs(dir: string): Promise<{ refs: RecordRef[]; findings: IdFinding[] }> {
  const namespace = resolveDir(dir)
  const files = (await listFiles(namespace, '.md')).filter((p) => ADR_CANDIDATE_RE.test(p))
  const refs: RecordRef[] = []
  const findings: IdFinding[] = []
  const supersession: AdrSupersessionRecord[] = []
  for (const path of files) {
    const result = adrRef(path, namespace)
    if (result.ref) {
      refs.push(result.ref)
      supersession.push(await adrSupersessionRecord(path, result.ref.id))
    }
    findings.push(...result.findings)
  }
  findings.push(...checkAdrSupersession(supersession))
  return { refs, findings }
}

function parseArgs(argv: string[]): { workItems: string[]; decisions: string[] } {
  const workItems: string[] = []
  const decisions: string[] = []
  let bucket: string[] | null = null
  for (const arg of argv) {
    if (arg === '--work-items') bucket = workItems
    else if (arg === '--decisions') bucket = decisions
    else if (bucket) bucket.push(arg)
    else
      throw new Error(`unexpected argument '${arg}' (expected --work-items or --decisions first)`)
  }
  return { workItems, decisions }
}

const { workItems, decisions } = parseArgs(process.argv.slice(2))
if (workItems.length === 0 && decisions.length === 0) {
  console.error('check-ids: nothing to check — pass --work-items <dir> and/or --decisions <dir>')
  process.exit(2)
}

const refs: RecordRef[] = []
const findings: IdFinding[] = []
for (const dir of workItems) {
  const r = await collectWorkItems(dir)
  refs.push(...r.refs)
  findings.push(...r.findings)
}
for (const dir of decisions) {
  const r = await collectAdrs(dir)
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

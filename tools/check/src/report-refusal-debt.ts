#!/usr/bin/env bun

/**
 * Split the declared refusals that no test names into the two kinds they are
 * made of, so they can be worked through instead of stared at.
 *
 * The list in security-refusal-debt.json says only "no test names this id". Two
 * very different situations produce that line: the refusal is exercised by a
 * test that never cites the id, and the refusal is exercised by nothing at all.
 * The first is a missing annotation; the second is a missing control test, and
 * it is the reason the annotation check is worth its friction (wi-390 found
 * REQ-WSFEDERATION-001 that way).
 *
 * Deciding which one an entry is means reading the test, so this does not
 * decide. It reports what can be established mechanically — whether the context
 * that owns the refusal has any test that asserts the same refusal at all — and
 * orders the work: an entry with no candidate is cheap to confirm and likely a
 * real gap; an entry with candidates needs one test read.
 *
 * It reports; it never fails. The check that fails is R3.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'

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

/** `AccessDeniedError` -> `access_denied`, the code the handlers write. */
function errorCode(type: string): string {
  return type
    .replace(/Error$/, '')
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .toLowerCase()
}

type Scenario = { id: string; title: string; context: string; step: string; types: string[] }

/** Read every scenario that declares a refusal, with the step that declares it. */
async function scenarios(): Promise<Map<string, Scenario>> {
  const found = new Map<string, Scenario>()
  const contextsDir = resolve(root, 'docs/contexts')
  for (const context of await readdir(contextsDir)) {
    const source = await readFile(resolve(contextsDir, context, 'scenarios.md'), 'utf8').catch(
      () => undefined,
    )
    if (!source) continue
    let current: Scenario | undefined
    for (const line of source.split('\n')) {
      const heading = line.match(/^### (REQ-[A-Z0-9]+-\d+): (.*)$/)
      if (heading) {
        current = { id: heading[1] ?? '', title: heading[2] ?? '', context, step: '', types: [] }
        found.set(current.id, current)
        continue
      }
      if (!current) continue
      if (!/^\s+- ALT /.test(line) && !/^- THEN /.test(line)) continue
      const types = [...line.matchAll(/\b([A-Z][A-Za-z0-9]*Error)\b/g)].map((m) => m[1] ?? '')
      if (types.length === 0) continue
      if (current.step === '') current.step = line.trim()
      for (const type of types) if (!current.types.includes(type)) current.types.push(type)
    }
  }
  return found
}

/**
 * The Go package that owns a context's behavior.
 *
 * The requirement prefix and the directory are the same word in every context
 * that has one; `apitoken` is the sole singular. Nothing is guessed beyond
 * that: a prefix that matches no directory is reported as unresolved rather
 * than pointed at an arbitrary package.
 */
function packageFor(id: string, backendDirs: string[]): string | undefined {
  const prefix = (id.match(/^REQ-([A-Z0-9]+)-/)?.[1] ?? '').toLowerCase()
  return backendDirs.find((dir) => dir === prefix) ?? backendDirs.find((dir) => prefix.startsWith(dir))
}

/** Cut top-level test functions out of a file; gofumpt closes them at column zero. */
function testFunctions(source: string): Array<{ name: string; body: string }> {
  const out: Array<{ name: string; body: string }> = []
  const lines = source.split('\n')
  for (let i = 0; i < lines.length; i += 1) {
    const header = (lines[i] ?? '').match(/^func (Test\w+)\(t \*testing\.T\) \{/)
    if (!header) continue
    let end = i
    while (end < lines.length && lines[end] !== '}') end += 1
    out.push({ name: header[1] ?? '', body: lines.slice(i, end + 1).join('\n') })
    i = end
  }
  return out
}

const debt = JSON.parse(
  await readFile(resolve(root, 'tools/check/security-refusal-debt.json'), 'utf8'),
) as { untested: string[] }
const declared = await scenarios()
const backendDirs = (await readdir(resolve(root, 'backend'), { withFileTypes: true }))
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)

// Tests are read once and grouped by the package whose behavior they exercise.
// A test counts for a package when it lives under it or imports it: the admin
// tests for the signing keys live in backend/oauth2/handlers_http and reach
// backend/signingkeys only through its import.
type Candidate = { path: string; name: string; body: string; inPackage: boolean }
const byPackage = new Map<string, Candidate[]>()
for (const path of (await walk(resolve(root, 'backend'))).filter((p) => p.endsWith('_test.go'))) {
  const rel = relative(root, path)
  const source = await readFile(path, 'utf8')
  const own = rel.split('/')[1] ?? ''
  const packages = new Set<string>([own])
  for (const imported of source.matchAll(/"[^"\n]*\/backend\/([a-z0-9]+)(?:\/[^"\n]*)?"/g)) {
    packages.add(imported[1] ?? '')
  }
  for (const pkg of packages) {
    const tests = byPackage.get(pkg) ?? []
    for (const fn of testFunctions(source)) {
      tests.push({ path: rel, name: fn.name, body: fn.body, inPackage: pkg === own })
    }
    byPackage.set(pkg, tests)
  }
}

const REFUSAL = /StatusForbidden|StatusUnauthorized|StatusConflict|StatusNotFound|StatusBadRequest|StatusUnprocessableEntity|Denied|Forbidden|Unauthorized|Rejects|Refuses/

type Row = { id: string; scenario?: Scenario; pkg?: string; named: string[]; nearby: string[] }
const rows: Row[] = []
for (const id of debt.untested) {
  const scenario = declared.get(id)
  const pkg = packageFor(id, backendDirs)
  const tests = pkg ? (byPackage.get(pkg) ?? []) : []
  const codes = (scenario?.types ?? []).map(errorCode)
  const refusing = tests.filter((test) => REFUSAL.test(test.body))
  // Naming the same error the scenario names is the strong signal; any other
  // refusal asserted against the same package is a place to start reading.
  const named = refusing.filter(
    (test) =>
      codes.some((code) => test.body.includes(code)) ||
      (scenario?.types ?? []).some((type) => test.body.includes(type)),
  )
  const namedKeys = new Set(named.map((test) => `${test.path}${test.name}`))
  // Tests inside the owning package come first: a test that reaches the context
  // only through an import is more often a neighbouring flow than this refusal.
  const label = (tests: Candidate[]) =>
    tests
      .sort((a, b) => Number(b.inPackage) - Number(a.inPackage))
      .map((test) => `${test.path}  ${test.name}${test.inPackage ? '' : '  (via import)'}`)
  rows.push({
    id,
    scenario,
    pkg,
    named: label(named),
    nearby: label(refusing.filter((test) => !namedKeys.has(`${test.path}${test.name}`))),
  })
}

const classOf = (row: Row) =>
  row.named.length > 0 ? 'named' : row.nearby.length > 0 ? 'nearby' : 'none'

console.log(`declared refusals awaiting a test  : ${rows.length}`)
console.log(
  `... a test names the same error     : ${rows.filter((r) => classOf(r) === 'named').length}  (most likely only the annotation is missing)`,
)
console.log(
  `... other refusal tests to read     : ${rows.filter((r) => classOf(r) === 'nearby').length}  (read one test to decide)`,
)
console.log(
  `... no refusal test reaches it      : ${rows.filter((r) => classOf(r) === 'none').length}  (most likely no test at all)`,
)
console.log('')

const byContext = new Map<string, { named: number; nearby: number; none: number }>()
for (const row of rows) {
  const context = row.scenario?.context ?? '(unknown)'
  const counts = byContext.get(context) ?? { named: 0, nearby: 0, none: 0 }
  counts[classOf(row)] += 1
  byContext.set(context, counts)
}
console.log(' named nearby  none  context')
for (const [context, counts] of [...byContext.entries()].sort((a, b) => b[1].none - a[1].none)) {
  console.log(
    `${String(counts.named).padStart(6)}${String(counts.nearby).padStart(7)}${String(counts.none).padStart(6)}  ${context}`,
  )
}

if (process.argv.includes('--list')) {
  console.log('')
  for (const row of rows) {
    console.log(`${row.id} [${classOf(row)}]: ${row.scenario?.title ?? '(no scenario)'}`)
    console.log(`  package: backend/${row.pkg ?? '(unresolved)'}`)
    if (row.scenario?.step) console.log(`  refusal: ${row.scenario.step}`)
    for (const candidate of row.named.slice(0, 3)) console.log(`  names the error: ${candidate}`)
    if (row.named.length === 0) {
      for (const candidate of row.nearby.slice(0, 3)) console.log(`  nearby: ${candidate}`)
    }
  }
}

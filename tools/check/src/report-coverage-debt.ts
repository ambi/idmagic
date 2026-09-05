#!/usr/bin/env bun

/**
 * Order the normative-coverage debt so it can be worked through instead of
 * stared at.
 *
 * The ledger says only "no test names this id". That one line covers two very
 * different situations: the behavior is exercised by a test that never cites
 * the id, and the behavior is exercised by nothing at all. The first is a
 * missing annotation; the second is a missing test, and it is the reason the
 * annotation check is worth its friction (wi-390 found REQ-WSFEDERATION-001
 * that way). Deciding which one an entry is means reading the test, so this
 * does not decide. It reports what can be established mechanically and orders
 * the work.
 *
 * The weight is derived from the contract, not from the prose. TypeSpec says
 * which operations answer a 403 and with which error body, and a 403 on a
 * state-changing operation is the contract naming a control the product relies
 * on. An id whose scenario names one of those types is carrying a control;
 * everything else is carrying a behavior. Both are debt. One is louder.
 *
 * The weight is deliberately narrow, and it under-reports. A control that
 * refuses with something other than a 403 on a non-GET operation does not reach
 * it: the workload attestation rejections, the data-key fail-closed branches,
 * and everything in seeding, which has no HTTP boundary at all, all weigh zero
 * here while being exactly the kind of refusal wi-390 was written for. Widening
 * it means widening what the contract states, not what this script guesses.
 *
 * Deriving it is the point. The weight used to be stored, as membership in a
 * separate `security-refusal-debt.json`, and what routed an id into that file
 * was fifteen words matched against Japanese prose. It matched condition
 * clauses ("cannot recover" in REQ-SYSTEM-015, whose outcome refuses nothing)
 * and field lists ("refusal reason" among REQ-AUTHORIZATION-009's audit
 * fields), and it missed every refusal phrased without one of the fifteen. A
 * derived weight can be wrong too, but it is wrong the same way the contract
 * is, and correcting it does not mean editing a ledger. See wi-490.
 *
 * It reports; it never fails. The checks that fail are coverage and R4.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import { parseScenarioDocument } from './gherkin-scenarios.ts'
import { contractRefusalsOfStateChanges } from './security-controls.ts'

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

/** Every scenario, with the error types its steps name. */
async function scenarios(): Promise<Map<string, Scenario>> {
  const found = new Map<string, Scenario>()
  const contextsDir = resolve(root, 'docs/contexts')
  for (const context of await readdir(contextsDir)) {
    const source = await readFile(
      resolve(contextsDir, context, 'scenarios.feature.md'),
      'utf8',
    ).catch(() => undefined)
    if (!source) continue
    const parsed = parseScenarioDocument(source)
    for (const rule of parsed.rules) {
      for (const example of rule.examples) {
        const outcome = example.steps.filter((step) => step.kind === 'outcome')
        const types = [
          ...new Set(
            outcome.flatMap((step) =>
              [...step.text.matchAll(/\b([A-Z][A-Za-z0-9]*Error)\b/g)].map(
                (match) => match[1] ?? '',
              ),
            ),
          ),
        ]
        found.set(example.id, {
          id: example.id,
          title: example.name,
          context,
          step: outcome[0]?.text ?? '',
          types,
        })
      }
    }
  }
  return found
}

/** Per context, the error types a 403 on a state-changing operation promises. */
async function promisedTypes(): Promise<Map<string, Set<string>>> {
  const promised = new Map<string, Set<string>>()
  const contractDir = resolve(root, 'spec/contexts')
  for (const context of await readdir(contractDir).catch(() => [])) {
    const types = new Set<string>()
    for (const entry of await readdir(resolve(contractDir, context)).catch(() => [])) {
      if (!entry.endsWith('.tsp')) continue
      const typespec = await readFile(resolve(contractDir, context, entry), 'utf8')
      for (const type of contractRefusalsOfStateChanges(typespec).keys()) types.add(type)
    }
    promised.set(context, types)
  }
  return promised
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
  const prefix = (id.match(/^(?:REQ|EX)-([A-Z0-9]+)-/)?.[1] ?? '').toLowerCase()
  return (
    backendDirs.find((dir) => dir === prefix) ?? backendDirs.find((dir) => prefix.startsWith(dir))
  )
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

type DebtEntry = { id: string; reason: string }

const debt = (
  JSON.parse(await readFile(resolve(root, 'tools/check/example-coverage-debt.json'), 'utf8')) as {
    untested: DebtEntry[]
  }
).untested
const declared = await scenarios()
const promised = await promisedTypes()
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

const REFUSAL =
  /StatusForbidden|StatusUnauthorized|StatusConflict|StatusNotFound|StatusBadRequest|StatusUnprocessableEntity|Denied|Forbidden|Unauthorized|Rejects|Refuses/

type Row = {
  id: string
  scenario?: Scenario
  pkg?: string
  control: boolean
  named: string[]
  nearby: string[]
}
const rows: Row[] = []
for (const entry of debt) {
  const scenario = declared.get(entry.id)
  const pkg = packageFor(entry.id, backendDirs)
  const tests = pkg ? (byPackage.get(pkg) ?? []) : []
  const types = scenario?.types ?? []
  const codes = types.map(errorCode)
  // The weight: does this scenario name a type the contract answers a 403 with
  // on an operation that changes state?
  const contract = promised.get(scenario?.context ?? '') ?? new Set<string>()
  const control = types.some((type) => contract.has(type))
  const refusing = tests.filter((test) => REFUSAL.test(test.body))
  // Naming the same error the scenario names is the strong signal; any other
  // refusal asserted against the same package is a place to start reading.
  const named = refusing.filter(
    (test) =>
      codes.some((code) => test.body.includes(code)) ||
      types.some((type) => test.body.includes(type)),
  )
  const namedKeys = new Set(named.map((test) => `${test.path}${test.name}`))
  // Tests inside the owning package come first: a test that reaches the context
  // only through an import is more often a neighbouring flow than this one.
  const label = (tests: Candidate[]) =>
    tests
      .sort((a, b) => Number(b.inPackage) - Number(a.inPackage))
      .map((test) => `${test.path}  ${test.name}${test.inPackage ? '' : '  (via import)'}`)
  rows.push({
    id: entry.id,
    scenario,
    pkg,
    control,
    named: label(named),
    nearby: label(refusing.filter((test) => !namedKeys.has(`${test.path}${test.name}`))),
  })
}

const classOf = (row: Row) =>
  row.named.length > 0 ? 'named' : row.nearby.length > 0 ? 'nearby' : 'none'

const controls = rows.filter((row) => row.control)
console.log(`scenario examples awaiting a test  : ${rows.length}`)
console.log(
  `... carrying a contract-promised 403: ${controls.length}  (narrow: a 403 on a non-GET operation, so a fail-closed branch outside HTTP weighs zero)`,
)
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

const byContext = new Map<
  string,
  { control: number; named: number; nearby: number; none: number }
>()
for (const row of rows) {
  const context = row.scenario?.context ?? '(unknown)'
  const counts = byContext.get(context) ?? { control: 0, named: 0, nearby: 0, none: 0 }
  counts[classOf(row)] += 1
  if (row.control) counts.control += 1
  byContext.set(context, counts)
}
console.log('control  named nearby  none  context')
for (const [context, counts] of [...byContext.entries()].sort(
  (a, b) => b[1].control - a[1].control || b[1].none - a[1].none,
)) {
  console.log(
    `${String(counts.control).padStart(7)}${String(counts.named).padStart(7)}${String(counts.nearby).padStart(7)}${String(counts.none).padStart(6)}  ${context}`,
  )
}

if (process.argv.includes('--list')) {
  console.log('')
  for (const row of rows) {
    const weight = row.control ? 'control' : 'behavior'
    console.log(`${row.id} [${weight}/${classOf(row)}]: ${row.scenario?.title ?? '(no scenario)'}`)
    console.log(`  package: backend/${row.pkg ?? '(unresolved)'}`)
    if (row.scenario?.step) console.log(`  step: ${row.scenario.step}`)
    for (const candidate of row.named.slice(0, 3)) console.log(`  names the error: ${candidate}`)
    if (row.named.length === 0) {
      for (const candidate of row.nearby.slice(0, 3)) console.log(`  nearby: ${candidate}`)
    }
  }
}

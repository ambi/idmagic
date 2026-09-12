#!/usr/bin/env bun

/**
 * Answer "where would I observe this?" for one normative id.
 *
 * Backing a declared example with a test costs two things: deciding what to
 * assert, and finding where to assert it. The second is pure search, and it was
 * being paid by hand, per example, and then paid again by the next work item
 * reaching for the neighbouring example in the same rule. wi-538 measured six
 * examples in the cheapest rule group of the largest context and every one of
 * them started with the same walk through the routes, the generated contract,
 * and the handler.
 *
 * Nothing here is new information. TypeSpec declares, per operation, the
 * method, the path, the granular scopes, and the error types each status
 * answers with; the scenarios name those same facts in their steps; the tests
 * name the ids they cover. This joins the three for one id. The join itself is
 * in route.ts; this file is the IO around it.
 *
 * It is deliberately not stored. An index in the repository would be a second
 * answer that goes stale against the first, which is the mistake
 * `security-refusal-debt.json` made before wi-490 replaced it with a
 * derivation. The ledger keeps only what cannot be derived: the `blocked_by`
 * and `finding` fields in normative-coverage.ts.
 *
 * It reports; it is not evidence. A candidate operation is a place to look, not
 * proof that a test exists or that one is unnecessary. Removing an id from the
 * ledger still means reading the test. See wi-565.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import { parseScenarioDocument } from '../../check/src/gherkin-scenarios.ts'
import {
  joinableFacts,
  parseDeclaredErrors,
  parseDeclaredOperations,
  parseGeneratedContract,
  rankCandidates,
  ruleOf,
} from './route.ts'

const root = resolve(import.meta.dir, '../../..')
const excluded = new Set(['.git', 'node_modules', 'vendor', 'dist', 'build', 'generated'])
const SHOWN = 12

async function walk(dir: string, result: string[] = []): Promise<string[]> {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.isDirectory() && excluded.has(entry.name)) continue
    const path = resolve(dir, entry.name)
    if (entry.isDirectory()) await walk(path, result)
    else if (entry.isFile()) result.push(path)
  }
  return result
}

const query = process.argv[2]
if (!query) {
  console.error('usage: mise run spec-route -- <EX-CONTEXT-NNN-MM | REQ-CONTEXT-NNN>')
  process.exit(2)
}
const wantedRule = ruleOf(query)

// The declaring document is found by searching, not by mapping the id prefix to
// a directory name. A mapping is a guess, it is wrong for exactly the contexts
// whose prefix and directory disagree, and being wrong there is silent.
const contextDirs = (await readdir(resolve(root, 'docs/contexts'), { withFileTypes: true }))
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
let located:
  | {
      contextDir: string
      docPath: string
      rule: ReturnType<typeof parseScenarioDocument>['rules'][number]
    }
  | undefined
for (const contextDir of [...contextDirs, '']) {
  const docPath = contextDir
    ? `docs/contexts/${contextDir}/scenarios.feature.md`
    : 'docs/scenarios.feature.md'
  const source = await readFile(resolve(root, docPath), 'utf8').catch(() => undefined)
  if (source === undefined) continue
  const rule = parseScenarioDocument(source).rules.find((candidate) => candidate.id === wantedRule)
  if (rule) {
    located = { contextDir, docPath, rule }
    break
  }
}
if (!located) {
  console.error(`${query}: no scenarios document declares ${wantedRule}`)
  process.exit(1)
}
const { contextDir, docPath, rule } = located

const examples = rule.examples.filter((example) => !query.startsWith('EX-') || example.id === query)
if (examples.length === 0) {
  console.error(`${query}: ${rule.id} declares no such example`)
  process.exit(1)
}

console.log(`# ${query}\n`)
console.log(`## Rule ${rule.id} — ${docPath}:${rule.line}`)
console.log(`${rule.name}\n`)
for (const example of examples) {
  console.log(`### ${example.name}  (${docPath}:${example.line})`)
  for (const step of example.steps) console.log(`- ${step.keyword} ${step.text}`)
  console.log()
}

const join = joinableFacts(examples.flatMap((example) => example.steps.map((step) => step.text)))

const typespec = (
  await Promise.all(
    (
      await walk(resolve(root, 'spec/contexts', contextDir || '.')).catch(() => [])
    )
      .filter((path) => path.endsWith('.tsp'))
      .map((path) => readFile(path, 'utf8')),
  )
).join('\n')
const declared = parseDeclaredOperations(typespec)
const errors = parseDeclaredErrors(typespec)
const operations = parseGeneratedContract(
  await readFile(resolve(root, 'backend/shared/spec/operations_gen.go'), 'utf8'),
).filter((operation) => declared.has(operation.name))

const ranked = rankCandidates(operations, errors, join)
const close = ranked.filter((candidate) => candidate.matched.length > 0)
const joined = [...join.errorTypes, ...join.paths, ...join.scopes]
console.log(
  close.length > 0
    ? `## Candidate operations — the contract joins on ${joined.join(', ')}`
    : '## Operations of this context — the steps name no error type, endpoint or scope, so nothing narrows them',
)
const shown = close.length > 0 ? close : ranked
if (shown.length === 0) console.log('- (none; this rule has no HTTP boundary in the contract)')
for (const { operation, matched } of shown.slice(0, SHOWN)) {
  const scopes = operation.scopes.length > 0 ? `  scopes: ${operation.scopes.join(' ')}` : ''
  const why = matched.length > 0 ? `  [${matched.join(', ')}]` : ''
  console.log(`- ${operation.name}: ${operation.method} ${operation.path}${scopes}${why}`)
}
if (shown.length > SHOWN) console.log(`- … and ${shown.length - SHOWN} more`)
console.log()

// A test already naming a sibling id of the same rule is where the observation
// most likely belongs, and it carries a fixture that already reaches the entry
// point.
const siblings = new Set<string>([rule.id, ...rule.examples.map((example) => example.id)])
const patterns = [...siblings].map(
  (id) => [id, new RegExp(`(?<![A-Za-z0-9-])${id}(?![A-Za-z0-9-])`)] as const,
)
const testPaths = [
  ...(await walk(resolve(root, 'backend')).catch(() => [])),
  ...(await walk(resolve(root, 'frontend')).catch(() => [])),
].filter((path) => /(?:_test\.go|\.(?:test|spec)\.tsx?)$/.test(path))

console.log('## Tests already naming an id of this rule')
let found = 0
for (const path of testPaths) {
  const source = await readFile(path, 'utf8')
  const hits = patterns.filter(([, pattern]) => pattern.test(source)).map(([id]) => id)
  if (hits.length === 0) continue
  found += 1
  console.log(`- ${relative(root, path)}: ${hits.sort().join(', ')}`)
}
if (found === 0) console.log('- (none; no test names this rule or any of its examples)')

console.log(
  '\nThese are places to look, derived from the contract and the test sources. ' +
    'They are not evidence that a test exists or that one is unnecessary: ' +
    'removing an id from the ledger still means reading the test.',
)

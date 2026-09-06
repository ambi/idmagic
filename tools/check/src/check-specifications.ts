#!/usr/bin/env bun

import type { Dirent } from 'node:fs'
import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'
import {
  checkNormativeCoverage,
  citedNormativeIds,
  type DebtEntry,
  type DeclaredId,
} from './normative-coverage.ts'
import { validateDocument } from './specification-doc.ts'

const args = process.argv.slice(2)
// A passing document is listed only on request; the closing count says the
// same thing in one line, and this check runs on every gate an agent reads.
const verbose = args.includes('--verbose')
const paths = args.filter((arg) => !arg.startsWith('--'))

const seen = new Map<string, string>()
const supersessions: Array<{ where: string; target: string }> = []
const scenarios: DeclaredId[] = []
const examples: DeclaredId[] = []
const standards: DeclaredId[] = []
let failed = false

for (const path of paths) {
  const rel = relative(process.cwd(), path)
  // The file name carries the grammar, so validation needs the path as the
  // repository sees it rather than as it looks from the tools directory.
  const canonical = relative(WORKSPACE_ROOT, path).replaceAll('\\', '/')
  const source = await readFile(path, 'utf8')
  const result = validateDocument(canonical, source)
  for (const finding of result.findings) {
    console.error(`${rel}:${finding.line}: ${finding.message}`)
    failed = true
  }
  for (const scenario of result.scenarioIds) {
    const previous = seen.get(scenario.id)
    if (previous) {
      console.error(
        `${rel}:${scenario.line}: duplicate ${scenario.id}; first declared in ${previous}`,
      )
      failed = true
    } else {
      seen.set(scenario.id, `${rel}:${scenario.line}`)
    }
    if (scenario.supersededBy) {
      supersessions.push({ where: `${rel}:${scenario.line}`, target: scenario.supersededBy })
      // A retired behavior has no steps left to exercise, so asking for a test
      // that names it would ask for a test of nothing.
      continue
    }
    scenarios.push({ id: scenario.id, path: `${canonical}:${scenario.line}` })
  }
  for (const example of result.exampleIds) {
    const previous = seen.get(example.id)
    if (previous) {
      console.error(
        `${rel}:${example.line}: duplicate ${example.id}; first declared in ${previous}`,
      )
      failed = true
    } else {
      seen.set(example.id, `${rel}:${example.line}`)
    }
    examples.push({ id: example.id, path: `${canonical}:${example.line}` })
  }
  for (const standard of result.standardIds) {
    standards.push({ id: standard.id, path: `${canonical}:${standard.line}` })
  }
  if (verbose) console.log(`ok  ${rel} (${result.scenarioIds.length} normative scenario id(s))`)
}
if (!verbose) console.log(`ok  ${paths.length} canonical document(s)`)

for (const supersession of supersessions) {
  if (!seen.has(supersession.target)) {
    console.error(`${supersession.where}: superseding ${supersession.target} does not exist`)
    failed = true
  }
}

/**
 * The trees a normative id may be named from. Only tests count: a mention in
 * implementation code would let a comment carrying the id satisfy the check.
 * Only the product's trees count either, because a tooling fixture is free to
 * write a real id as sample data and would then read as coverage.
 */
const PRODUCT_TREES = ['backend', 'frontend']
const TEST_FILE = /(?:_test\.go|\.(?:test|spec)\.tsx?)$/
const EXCLUDED_DIRECTORIES = new Set(['node_modules', 'vendor', 'dist', 'build', 'generated'])

async function testSources(directory: string, found: string[] = []): Promise<string[]> {
  let entries: Dirent[] = []
  try {
    entries = await readdir(directory, { withFileTypes: true })
  } catch {
    return found
  }
  for (const entry of entries) {
    const path = resolve(directory, entry.name)
    if (entry.isDirectory()) {
      if (EXCLUDED_DIRECTORIES.has(entry.name) || entry.name.startsWith('.')) continue
      await testSources(path, found)
    } else if (entry.isFile() && TEST_FILE.test(entry.name)) {
      found.push(await readFile(path, 'utf8'))
    }
  }
  return found
}

type DebtFile = { comment?: string[]; untested: DebtEntry[] }
type DebtBaselineFile = { comment?: string[]; ids: string[] }

/**
 * A debt list, or none. An absent file is the empty list rather than an error:
 * a workspace that has never needed one should not have to carry an empty file
 * to be checkable, and reading absence as "no debt is allowed" is the strict
 * direction. Losing the list by accident is loud, not silent -- every id it
 * held is reported the moment it is gone.
 */
async function readDebt(path: string): Promise<DebtEntry[]> {
  const source = await readFile(resolve(WORKSPACE_ROOT, path), 'utf8').catch(() => undefined)
  if (source === undefined) return []
  const parsed = JSON.parse(source) as DebtFile
  if (!Array.isArray(parsed.untested)) throw new Error(`${path}: untested must be an array`)
  return parsed.untested.map((entry) => {
    if (typeof entry?.id !== 'string' || typeof entry?.reason !== 'string') {
      throw new Error(`${path}: every untested entry needs an id and a reason`)
    }
    return entry
  })
}

async function readDebtBaseline(path: string): Promise<Set<string>> {
  const source = await readFile(resolve(WORKSPACE_ROOT, path), 'utf8').catch(() => undefined)
  if (source === undefined) return new Set()
  const parsed = JSON.parse(source) as DebtBaselineFile
  if (!Array.isArray(parsed.ids) || parsed.ids.some((id) => typeof id !== 'string')) {
    throw new Error(`${path}: ids must be an array of strings`)
  }
  return new Set(parsed.ids)
}

const EXAMPLE_DEBT = 'tools/check/example-coverage-debt.json'
const EXAMPLE_DEBT_BASELINE = 'tools/check/example-coverage-debt-baseline.json'
const STANDARDS_DEBT = 'tools/check/standards-coverage-debt.json'

const sources: string[] = []
for (const tree of PRODUCT_TREES)
  sources.push(...(await testSources(resolve(WORKSPACE_ROOT, tree))))
const cited = citedNormativeIds(
  sources,
  [...standards, ...examples].map((declaration) => declaration.id),
)

const coverage = [
  ...checkNormativeCoverage({
    declared: standards,
    cited,
    debt: await readDebt(STANDARDS_DEBT),
    debtPath: STANDARDS_DEBT,
  }),
  ...checkNormativeCoverage({
    declared: examples,
    cited,
    debt: await readDebt(EXAMPLE_DEBT),
    debtPath: EXAMPLE_DEBT,
    debtBaseline: await readDebtBaseline(EXAMPLE_DEBT_BASELINE),
  }),
]
for (const finding of coverage) {
  console.error(`${finding.path}: ${finding.message}`)
  failed = true
}
if (coverage.length === 0) {
  console.log(
    `ok  normative coverage (${standards.length} standard(s), ${scenarios.length} rule(s), ${examples.length} example(s), ` +
      `${cited.size} id(s) named by a test)`,
  )
}

if (failed) process.exit(1)

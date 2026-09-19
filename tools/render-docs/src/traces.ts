/**
 * Collect what names a normative identifier outside the specification.
 *
 * The search runs over the working tree rather than an authored index, so the
 * answer can never drift from what the repository actually says. Two readers
 * ask for it: the Traceability page, which draws it, and the work-item brief,
 * which answers "what already exists for this requirement" at a terminal. They
 * share this one walk so that the browsable view and the terminal answer are
 * the same answer.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import type { ScenarioTrace } from './render.ts'

const SCENARIO_IDENTIFIER = /(?:REQ-[A-Z0-9]+-[0-9]+|EX-[A-Z0-9]+-[0-9]+-[0-9]+)/g
const TEST_FILE = /(?:_test\.go|\.(?:test|spec)\.tsx?)$/
const SKIPPED_DIRECTORIES = new Set([
  '.git',
  'node_modules',
  'vendor',
  'dist',
  'build',
  'generated',
  'coverage',
])
const TEXT_EXTENSIONS = /\.(?:go|ts|tsx|js|jsx|sql|sh|py|rb|java|kt|rs|md|yaml|yml|json|tsp|toml)$/
const DEBT_LEDGER = 'tools/check/example-coverage-debt.json'

/** Whether a repository-relative path names a test rather than implementation. */
export function isTestPath(path: string): boolean {
  return TEST_FILE.test(path)
}

export async function collectTraces(root: string): Promise<ScenarioTrace[]> {
  const sources = new Map<string, Set<string>>()
  const workItems = new Map<string, Set<string>>()
  const debtSource = JSON.parse(await readFile(resolve(root, DEBT_LEDGER), 'utf8')) as {
    untested: Array<{ id: string; reason: string }>
  }
  const debt = new Map(debtSource.untested.map((entry) => [entry.id, entry.reason]))
  const walk = async (directory: string): Promise<void> => {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      if (entry.name.startsWith('.') || SKIPPED_DIRECTORIES.has(entry.name)) continue
      const absolute = resolve(directory, entry.name)
      const path = relative(root, absolute)
      if (entry.isDirectory()) {
        await walk(absolute)
        continue
      }
      // The specification declares the scenarios; only what points at them counts.
      if (path.startsWith('docs/') || path.startsWith('spec/') || !TEXT_EXTENSIONS.test(path))
        continue
      if (path === DEBT_LEDGER) continue
      const target = path.startsWith('work-items/') ? workItems : sources
      for (const match of (await readFile(absolute, 'utf8')).matchAll(SCENARIO_IDENTIFIER)) {
        if (match[0].startsWith('EX-') && target === sources && !isTestPath(path)) continue
        const paths = target.get(match[0]) ?? new Set<string>()
        paths.add(path)
        target.set(match[0], paths)
      }
    }
  }
  await walk(root)
  const ids = new Set([...sources.keys(), ...workItems.keys(), ...debt.keys()])
  return [...ids].map((id) => ({
    id,
    sources: [...(sources.get(id) ?? [])].sort(),
    workItems: [...(workItems.get(id) ?? [])].sort(),
    debt: debt.get(id),
  }))
}

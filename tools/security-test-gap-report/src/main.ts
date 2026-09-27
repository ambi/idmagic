#!/usr/bin/env bun

/**
 * List the refused state changes no test proves were left untouched.
 *
 * This reports; it does not fail. What a test must assert to prove the absence
 * of an effect depends on the operation, and a check that demanded "some read
 * happens somewhere in the body" would be satisfied by one meaningless call —
 * manufacturing exactly the hollow test a coverage threshold produces. The
 * number is worth watching, so it is printed rather than enforced.
 *
 * Only refusals of state-changing operations are counted. A refused read that
 * runs anyway leaves nothing behind.
 */

import { readdir, readFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'

import { renderReport, reportSecurityTestGaps, testFunctions, type GoTest } from './report.ts'

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

const paths = (await walk(resolve(root, 'backend'))).filter((path) => path.endsWith('_test.go'))
const tests: GoTest[] = []
for (const path of paths) {
  const source = await readFile(path, 'utf8')
  for (const fn of testFunctions(source)) {
    tests.push({ path: relative(root, path), ...fn })
  }
}
const report = reportSecurityTestGaps(tests)
console.log(renderReport(report, process.argv.includes('--list')))

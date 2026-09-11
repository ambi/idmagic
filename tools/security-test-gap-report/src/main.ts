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

const root = resolve(import.meta.dir, '../../..')
const excluded = new Set(['.git', 'node_modules', 'vendor', 'dist', 'build', 'generated'])

const MUTATING =
  /http\.Method(Post|Put|Patch|Delete)|\b(Create|Update|Delete|Cancel|Revoke|Disable|Enable|Kill|Rotate|Add|Remove|Write|Save|Issue|Approve|Deny|Bind|Unbind|Import|Apply)[A-Z]\w*\(/
const REFUSAL =
  /StatusForbidden|StatusUnauthorized|StatusConflict|StatusNotFound|StatusBadRequest|StatusUnprocessableEntity|Err\w*(Denied|Forbidden|Unauthorized|NotFound|AlreadyTerminal|LeaseLost|Unscoped|Mismatch)/
const READBACK = /\b(Get|Find|List|Load|Count|Lookup|Resolve)[A-Za-z]*\(|\.(Get|Find|List|Count)\(/

async function walk(dir: string, result: string[] = []): Promise<string[]> {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.isDirectory() && excluded.has(entry.name)) continue
    const path = resolve(dir, entry.name)
    if (entry.isDirectory()) await walk(path, result)
    else if (entry.isFile()) result.push(path)
  }
  return result
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

const paths = (await walk(resolve(root, 'backend'))).filter((path) => path.endsWith('_test.go'))
const gaps: Array<{ path: string; name: string }> = []
let refused = 0
for (const path of paths) {
  const source = await readFile(path, 'utf8')
  for (const fn of testFunctions(source)) {
    if (!REFUSAL.test(fn.body) || !MUTATING.test(fn.body)) continue
    refused += 1
    if (!READBACK.test(fn.body)) gaps.push({ path: relative(root, path), name: fn.name })
  }
}

const byArea = new Map<string, number>()
for (const gap of gaps) {
  const area = gap.path.split('/').slice(0, 3).join('/')
  byArea.set(area, (byArea.get(area) ?? 0) + 1)
}

console.log(`refused state changes covered by a test : ${refused}`)
console.log(`... with no read-back proving no effect : ${gaps.length}`)
console.log('')
for (const [area, count] of [...byArea.entries()].sort((a, b) => b[1] - a[1])) {
  console.log(`${String(count).padStart(4)}  ${area}`)
}
if (process.argv.includes('--list')) {
  console.log('')
  for (const gap of gaps) console.log(`${gap.path}  ${gap.name}`)
}

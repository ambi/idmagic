#!/usr/bin/env bun
/**
 * Derive what a change did to the normative specification.
 *
 *   spec-diff [ref]     # default: main
 *
 * Reviewers and agents need the normative delta of a change on its own, apart
 * from the code diff: which scenarios appeared, disappeared, or changed, which
 * state-transition rows moved, and which TypeSpec declarations came and went.
 * Writing that delta by hand means keeping a second copy of the specification,
 * so it is computed from git instead.
 *
 * This is a reading aid, not a gate. Nothing here fails a build.
 */

import { readdir } from 'node:fs/promises'
import { relative, resolve } from 'node:path'

/** Repository-relative path to file contents. */
export type Snapshot = Map<string, string>

export type SpecificationFacts = {
  /** Normative scenario id to its normalized body. */
  scenarios: Map<string, string>
  /** `<document>#<machine>` to its transition rows. */
  transitions: Map<string, Set<string>>
  /** `<path>:<declaration>` for every TypeSpec declaration. */
  declarations: Set<string>
}

export type SpecificationDiff = {
  addedScenarios: string[]
  removedScenarios: string[]
  changedScenarios: string[]
  changedTransitions: string[]
  addedDeclarations: string[]
  removedDeclarations: string[]
}

const DECLARATION = /\b(?:alias|enum|model|op|scalar|union)\s+([A-Za-z_][A-Za-z0-9_]*)/g

/**
 * Per-operation transport wrappers follow the operation they belong to and say
 * nothing on their own. Listing them buries the declarations a reader came for.
 */
const TRANSPORT_WRAPPER = /(?:Error\d{3}(?:Body)?|Http(?:Request|Response)|Success_\d{3})$/

function section(source: string, name: string): string {
  const start = source.match(new RegExp(`^## ${name}\\s*$`, 'm'))
  if (!start || start.index === undefined) return ''
  const rest = source.slice(start.index + start[0].length)
  const end = rest.match(/^## /m)
  return end?.index === undefined ? rest : rest.slice(0, end.index)
}

function normalize(block: string): string {
  return block
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .join('\n')
}

export function extractFacts(snapshot: Snapshot): SpecificationFacts {
  const facts: SpecificationFacts = {
    scenarios: new Map(),
    transitions: new Map(),
    declarations: new Set(),
  }

  for (const [path, source] of snapshot) {
    if (path.endsWith('.tsp')) {
      for (const match of source.matchAll(DECLARATION)) {
        const name = match[1] ?? ''
        if (!TRANSPORT_WRAPPER.test(name)) facts.declarations.add(`${path}:${name}`)
      }
      continue
    }

    const scenarios = section(source, 'Scenarios')
    const starts = [...scenarios.matchAll(/^### (REQ-[A-Z0-9-]+):/gm)]
    for (const [index, start] of starts.entries()) {
      const from = start.index ?? 0
      const to = starts[index + 1]?.index ?? scenarios.length
      facts.scenarios.set(start[1] ?? '', normalize(scenarios.slice(from, to)))
    }

    const transitions = section(source, 'State Transitions')
    let machine = ''
    for (const line of transitions.split('\n')) {
      const heading = line.match(/^### (.+)$/)
      if (heading) {
        machine = `${path}#${heading[1]}`
        continue
      }
      const row = line.trim()
      if (!machine || !row.startsWith('|') || /^\|[\s|:-]+\|$/.test(row)) continue
      if (row.includes('| From | Event |')) continue
      const rows = facts.transitions.get(machine) ?? new Set<string>()
      rows.add(row)
      facts.transitions.set(machine, rows)
    }
  }

  return facts
}

export function diffSpecifications(base: Snapshot, head: Snapshot): SpecificationDiff {
  const before = extractFacts(base)
  const after = extractFacts(head)

  const addedScenarios: string[] = []
  const changedScenarios: string[] = []
  for (const [id, body] of after.scenarios) {
    const previous = before.scenarios.get(id)
    if (previous === undefined) addedScenarios.push(id)
    else if (previous !== body) changedScenarios.push(id)
  }
  const removedScenarios = [...before.scenarios.keys()].filter((id) => !after.scenarios.has(id))

  const machines = new Set([...before.transitions.keys(), ...after.transitions.keys()])
  const changedTransitions: string[] = []
  for (const machine of machines) {
    const previous = before.transitions.get(machine) ?? new Set<string>()
    const current = after.transitions.get(machine) ?? new Set<string>()
    const same = previous.size === current.size && [...current].every((row) => previous.has(row))
    if (!same) changedTransitions.push(machine)
  }

  return {
    addedScenarios: addedScenarios.sort(),
    removedScenarios: removedScenarios.sort(),
    changedScenarios: changedScenarios.sort(),
    changedTransitions: changedTransitions.sort(),
    addedDeclarations: [...after.declarations]
      .filter((one) => !before.declarations.has(one))
      .sort(),
    removedDeclarations: [...before.declarations]
      .filter((one) => !after.declarations.has(one))
      .sort(),
  }
}

export function formatSpecificationDiff(diff: SpecificationDiff, ref: string): string {
  const groups: Array<[string, string[]]> = [
    ['added scenarios', diff.addedScenarios],
    ['removed scenarios', diff.removedScenarios],
    ['changed scenarios', diff.changedScenarios],
    ['changed state transitions', diff.changedTransitions],
    ['added TypeSpec declarations', diff.addedDeclarations],
    ['removed TypeSpec declarations', diff.removedDeclarations],
  ]
  const lines = groups
    .filter(([, entries]) => entries.length > 0)
    .map(([label, entries]) => `${label}:\n${entries.map((entry) => `  ${entry}`).join('\n')}`)
  return lines.length === 0
    ? `no normative specification change against ${ref}`
    : `normative specification change against ${ref}\n\n${lines.join('\n\n')}`
}

function isSpecificationSource(path: string): boolean {
  if (path.startsWith('spec/generated/')) return false
  return (
    path.endsWith('/SPECIFICATION.md') || path === 'spec/SPECIFICATION.md' || path.endsWith('.tsp')
  )
}

async function readWorkingTree(root: string): Promise<Snapshot> {
  const snapshot: Snapshot = new Map()
  const walk = async (directory: string): Promise<void> => {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      const absolute = resolve(directory, entry.name)
      const path = relative(root, absolute)
      if (entry.isDirectory()) {
        if (entry.name === 'generated' || entry.name === 'node_modules') continue
        await walk(absolute)
      } else if (isSpecificationSource(path)) {
        snapshot.set(path, await Bun.file(absolute).text())
      }
    }
  }
  await walk(resolve(root, 'spec'))
  return snapshot
}

function git(root: string, args: string[]): string {
  const result = Bun.spawnSync(['git', ...args], { cwd: root })
  if (result.exitCode !== 0) {
    throw new Error(`git ${args.join(' ')} failed: ${result.stderr.toString().trim()}`)
  }
  return result.stdout.toString()
}

function readRevision(root: string, ref: string): Snapshot {
  const snapshot: Snapshot = new Map()
  for (const path of git(root, ['ls-tree', '-r', '--name-only', ref, '--', 'spec']).split('\n')) {
    if (isSpecificationSource(path)) snapshot.set(path, git(root, ['show', `${ref}:${path}`]))
  }
  return snapshot
}

if (import.meta.main) {
  const root = resolve(import.meta.dir, '../../..')
  const ref = process.argv[2] ?? 'main'
  const diff = diffSpecifications(readRevision(root, ref), await readWorkingTree(root))
  console.log(formatSpecificationDiff(diff, ref))
}

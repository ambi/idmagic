#!/usr/bin/env bun
/**
 * Derive what a change did to the normative specification.
 *
 *   spec-diff [ref]     # default: main
 *
 * Reviewers and agents need the normative delta of a change on its own, apart
 * from the code diff: which scenarios appeared, disappeared, or changed, which
 * state-transition rows moved, and which TypeSpec declarations came, went, or
 * changed. Writing that delta by hand means keeping a second copy of the
 * specification, so it is computed from git instead.
 *
 * The `spec-diff` task is a reading aid and fails nothing. The `spec-impact`
 * check reads the same delta to compare it with what work items declare.
 */

import { readdir } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import MarkdownIt from 'markdown-it'
import { specificationRules } from './feature-specification.ts'
import { type ScenarioExample, parseScenarioDocument, ruleBodies } from './gherkin-scenarios.ts'
import { documentKind } from './specification-doc.ts'
import { typeSpecDeclarations } from './typespec-declarations.ts'

/** Repository-relative path to file contents. */
export type Snapshot = Map<string, string>

export type SpecificationFacts = {
  /** Normative scenario id to its normalized body. */
  scenarios: Map<string, string>
  /** `<owning directory>#<machine>` to its transition rows. */
  transitions: Map<string, Set<string>>
  /** `<path>#<normative id>` to the normalized standards row. */
  standards: Map<string, string>
  /** `<path>:<declaration>` to the normalized text of every TypeSpec declaration. */
  declarations: Map<string, string>
  /** `<path>:<declaration>` for every declaration carrying `@deprecated`. */
  deprecatedDeclarations: Set<string>
}

export type SpecificationDiff = {
  addedScenarios: string[]
  removedScenarios: string[]
  changedScenarios: string[]
  changedTransitions: string[]
  addedStandards: string[]
  removedStandards: string[]
  changedStandards: string[]
  addedDeclarations: string[]
  removedDeclarations: string[]
  changedDeclarations: string[]
  addedDeprecations: string[]
  removedDeprecations: string[]
}

/**
 * Per-operation transport wrappers follow the operation they belong to and say
 * nothing on their own. Listing them buries the declarations a reader came for.
 */
const TRANSPORT_WRAPPER = /(?:Error\d{3}(?:Body)?|Http(?:Request|Response)|Success_\d{3})$/
const markdown = new MarkdownIt({ html: false, linkify: false, typographer: false })

/**
 * 一つのファイルの宣言を、名前から本文への対応にする。
 *
 * 操作ごとの転送用の宣言は一覧に出さないが、その変更は操作の契約を変える。そこで本文を、
 * 所有する操作の本文へ連ねる。所有する操作がこのファイルにない転送用の宣言は読まない。
 */
function declarationTexts(source: string): Map<string, string> {
  const texts = new Map<string, string>()
  const wrappers: Array<{ owner: string; text: string }> = []
  for (const { name, text } of typeSpecDeclarations(source)) {
    if (TRANSPORT_WRAPPER.test(name)) {
      wrappers.push({ owner: name.replace(TRANSPORT_WRAPPER, ''), text })
    } else {
      texts.set(name, texts.has(name) ? `${texts.get(name)}\n${text}` : text)
    }
  }
  for (const { owner, text } of wrappers) {
    if (texts.has(owner)) texts.set(owner, `${texts.get(owner)}\n${text}`)
  }
  return texts
}

function deprecatedDeclarations(path: string, source: string): Set<string> {
  const declarations = new Set<string>()
  let pending = false
  for (const line of source.split('\n')) {
    const trimmed = line.trim()
    if (trimmed.startsWith('@deprecated')) {
      pending = true
      continue
    }
    if (trimmed.length === 0 || trimmed.startsWith('@') || trimmed.startsWith('//')) continue
    const declaration = line.match(
      /^[ \t]*(?:alias|enum|model|op|scalar|union)\s+([A-Za-z_][A-Za-z0-9_]*)/,
    )
    if (declaration?.[1]) {
      if (pending && !TRANSPORT_WRAPPER.test(declaration[1])) {
        declarations.add(`${path}:${declaration[1]}`)
      }
      pending = false
      continue
    }
    pending = false
  }
  return declarations
}

/**
 * 文書を移しただけの改名を、規範の変更として数えない。この道具は履歴のリビジョンを読むので、
 * 当時のパスを現在の配置へ写してから同定する。
 */
const RELOCATED_DOCUMENTS = new Map([
  ['docs/glossary.md', 'docs/requirements/glossary.md'],
  ['docs/standards.md', 'docs/requirements/standards.md'],
  ['docs/structure.md', 'docs/design/application/backend.md'],
  ['docs/scenarios.feature.md', 'docs/requirements/scenarios.feature.md'],
  ['docs/product-overview.md', 'docs/requirements/product-overview.md'],
  ['docs/design/product-overview.md', 'docs/requirements/product-overview.md'],
  ['docs/domain/glossary.md', 'docs/requirements/glossary.md'],
  ['docs/domain/standards.md', 'docs/requirements/standards.md'],
  ['docs/domain/structure.md', 'docs/design/application/backend.md'],
  ['docs/domain/scenarios.feature.md', 'docs/requirements/scenarios.feature.md'],
])

function currentPath(path: string): string {
  return (
    RELOCATED_DOCUMENTS.get(path) ??
    path
      .replace(/^docs\/(?:contexts|domain)\//, 'docs/modules/')
      .replace(/^spec\/contexts\//, 'spec/modules/')
  )
}

/** 標準仕様の行は、それを所有する文書と ID で同定する。 */
function standardRows(path: string, source: string): Map<string, string> {
  const owner = currentPath(path)
  const rows = new Map<string, string>()
  let cells: string[] | undefined
  for (const token of markdown.parse(source, {})) {
    if (token.type === 'tr_open') {
      cells = []
    } else if (token.type === 'inline' && cells) {
      cells.push(token.content.trim().replaceAll(/\s+/g, ' '))
    } else if (token.type === 'tr_close' && cells) {
      const id = cells[0]
      if (id && /^[A-Z][A-Z0-9-]+$/.test(id)) rows.set(`${owner}#${id}`, cells.join(' | '))
      cells = undefined
    }
  }
  return rows
}

function section(source: string, name: string): string {
  const start = source.match(new RegExp(`^## ${name}\\s*$`, 'm'))
  if (!start || start.index === undefined) return ''
  const rest = source.slice(start.index + start[0].length)
  const end = rest.match(/^## /m)
  return end?.index === undefined ? rest : rest.slice(0, end.index)
}

function legacyScenarioFacts(source: string): Map<string, string> {
  const facts = new Map<string, string>()
  const starts = [...source.matchAll(/^### (REQ-[A-Z0-9-]+): (.+)$/gm)]
  for (const [index, start] of starts.entries()) {
    const from = start.index ?? 0
    const to = starts[index + 1]?.index ?? source.length
    const fragments = new Set<string>()
    for (const line of source.slice(from, to).split('\n')) {
      const step = line.match(/^- (?:GIVEN|WHEN|THEN) (.+)$/)?.[1]
      if (step) fragments.add(step)
      const alternative = line.match(/^ {2}- ALT (.+)$/)?.[1]
      if (alternative) for (const fragment of alternative.split(' → ')) fragments.add(fragment)
    }
    facts.set(start[1] ?? '', ['legacy', start[2] ?? '', ...[...fragments].sort()].join('\n'))
  }
  return facts
}

/**
 * 規則の本文の一行を、書式の揺れを除いた形にする。表の区切り行は内容を持たないので落とし、
 * 表のセルは前後の空白を除く。リンクはラベルだけを比べる。リンク先の相対パスは文書を
 * 移すだけで変わり、規則の内容を表さない。
 */
function normalizedBodyLine(text: string): string | undefined {
  const line = text
    .trim()
    .replaceAll(/\s+/g, ' ')
    .replaceAll(/\[([^\]]*)\]\([^)]*\)/g, '[$1]')
  if (/^\|[\s|:-]+\|$/.test(line)) return undefined
  if (!line.startsWith('|')) return line
  return line
    .split('|')
    .map((cell) => cell.trim())
    .join('|')
}

/** 入力と期待結果の対応と操作順を保ち、例の掲載順と行番号は比較しない。 */
function exampleFact(example: ScenarioExample): string {
  return JSON.stringify([
    example.id,
    example.steps.map(({ kind, text, argument }) => ({ kind, text, argument })),
    example.parameters &&
      Object.entries(example.parameters).sort(([left], [right]) => left.localeCompare(right)),
  ])
}

function gherkinScenarioFacts(source: string): Map<string, string> {
  const facts = new Map<string, string>()
  const bodies = new Map(ruleBodies(source).map((body) => [body.id, body.lines]))
  for (const rule of parseScenarioDocument(source).rules) {
    const fragments = new Set(rule.examples.map(exampleFact))
    const title = rule.name.replace(new RegExp(`^${rule.id}(?::)?\\s*`), '')
    // 規則の本文と例の内部は順序を保ち、独立した例の掲載順だけを比較から外す。
    const body = (bodies.get(rule.id) ?? []).flatMap(({ text }) => normalizedBodyLine(text) ?? [])
    facts.set(rule.id, ['gherkin', title, ...[...fragments].sort(), '--', ...body].join('\n'))
  }
  return facts
}

export function extractFacts(snapshot: Snapshot): SpecificationFacts {
  const facts: SpecificationFacts = {
    scenarios: new Map(),
    transitions: new Map(),
    standards: new Map(),
    declarations: new Map(),
    deprecatedDeclarations: new Set(),
  }

  // 機能仕様の規則は、宣言と本文が仕様本文に、例が付録にある。両方を読んでから一つの事実にする。
  const declared = new Map<string, { title: string; body: string[] }>()
  const steps = new Map<string, Set<string>>()

  for (const [path, source] of snapshot) {
    const kind = documentKind(path)
    if (kind === 'specification') {
      const bodies = new Map(ruleBodies(source).map((body) => [body.id, body.lines]))
      for (const rule of specificationRules(source)) {
        const body = (bodies.get(rule.id) ?? []).flatMap(
          ({ text }) => normalizedBodyLine(text) ?? [],
        )
        declared.set(rule.id, { title: rule.title, body })
      }
    } else if (kind === 'examples') {
      for (const rule of parseScenarioDocument(source).rules) {
        const fragments = steps.get(rule.id) ?? new Set<string>()
        for (const example of rule.examples) fragments.add(exampleFact(example))
        steps.set(rule.id, fragments)
      }
    }
    if (path.endsWith('.tsp')) {
      for (const [name, text] of declarationTexts(source)) {
        facts.declarations.set(`${currentPath(path)}:${name}`, text)
      }
      for (const declaration of deprecatedDeclarations(currentPath(path), source)) {
        facts.deprecatedDeclarations.add(declaration)
      }
      continue
    }

    // In the split layout the file name says what the file holds, so the whole
    // file is the section; the single canonical document names its sections.
    const name = path.split('/').at(-1) ?? ''
    if (name === 'standards.md') {
      for (const [id, row] of standardRows(path, source)) facts.standards.set(id, row)
    }
    const scenarioFacts =
      name === 'scenarios.feature.md'
        ? gherkinScenarioFacts(source)
        : name === 'scenarios.md'
          ? legacyScenarioFacts(source)
          : legacyScenarioFacts(section(source, 'Scenarios'))
    for (const [id, body] of scenarioFacts) {
      facts.scenarios.set(id, body)
    }

    // A machine belongs to the context that owns it, not to the file that
    // happens to hold it, so moving it between files is not a change. 機能スライスへ
    // 移すことも同じで、機能スライスの段を落としてモジュールで同定する。
    const owner = currentPath(
      path.slice(0, Math.max(0, path.length - name.length - 1)) || path,
    ).replace(/^(docs\/modules\/[^/]+)(?:\/[^/]+)+$/, '$1')
    const split = name === 'states.md'
    const transitions = split
      ? source
      : kind === 'specification'
        ? section(source, '状態遷移')
        : section(source, 'State Transitions')
    const machineHeading = split ? /^## (?!#)(.+)$/ : /^### (.+)$/
    let machine = ''
    // Only the rows under the transition header are transitions. A states.md
    // also carries the state table, whose rows say nothing about a transition.
    let inTransitions = false
    for (const line of transitions.split('\n')) {
      const heading = line.match(machineHeading)
      if (heading) {
        machine = `${owner}#${heading[1]}`
        inTransitions = false
        continue
      }
      const row = line.trim()
      if (row.includes('| From | Event |')) {
        inTransitions = true
        continue
      }
      if (!row.startsWith('|')) {
        if (row.length > 0) inTransitions = false
        continue
      }
      if (!machine || !inTransitions || /^\|[\s|:-]+\|$/.test(row)) continue
      const rows = facts.transitions.get(machine) ?? new Set<string>()
      rows.add(row)
      facts.transitions.set(machine, rows)
    }
  }

  // 形式の印を旧形式の `gherkin` と分けるので、移行のコミットでは題名だけが比べられる。
  for (const [id, { title, body }] of declared) {
    const fragments = [...(steps.get(id) ?? [])].sort()
    facts.scenarios.set(id, ['spec', title, ...fragments, '--', ...body].join('\n'))
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
    else {
      const [previousFormat, previousTitle] = previous.split('\n')
      const [currentFormat, currentTitle] = body.split('\n')
      const formatMigration = previousFormat !== currentFormat
      if (
        (formatMigration && previousTitle !== currentTitle) ||
        (!formatMigration && previous !== body)
      ) {
        changedScenarios.push(id)
      }
    }
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

  const addedStandards: string[] = []
  const changedStandards: string[] = []
  for (const [id, row] of after.standards) {
    const previous = before.standards.get(id)
    if (previous === undefined) addedStandards.push(id)
    else if (previous !== row) changedStandards.push(id)
  }
  const removedStandards = [...before.standards.keys()].filter((id) => !after.standards.has(id))

  return {
    addedScenarios: addedScenarios.sort(),
    removedScenarios: removedScenarios.sort(),
    changedScenarios: changedScenarios.sort(),
    changedTransitions: changedTransitions.sort(),
    addedStandards: addedStandards.sort(),
    removedStandards: removedStandards.sort(),
    changedStandards: changedStandards.sort(),
    addedDeclarations: [...after.declarations.keys()]
      .filter((one) => !before.declarations.has(one))
      .sort(),
    removedDeclarations: [...before.declarations.keys()]
      .filter((one) => !after.declarations.has(one))
      .sort(),
    changedDeclarations: [...after.declarations]
      .filter(
        ([one, text]) => before.declarations.has(one) && before.declarations.get(one) !== text,
      )
      .map(([one]) => one)
      .sort(),
    addedDeprecations: [...after.deprecatedDeclarations]
      .filter((one) => !before.deprecatedDeclarations.has(one))
      .sort(),
    removedDeprecations: [...before.deprecatedDeclarations]
      .filter((one) => !after.deprecatedDeclarations.has(one))
      .sort(),
  }
}

export function formatSpecificationDiff(diff: SpecificationDiff, ref: string): string {
  const groups: Array<[string, string[]]> = [
    ['added scenarios', diff.addedScenarios],
    ['removed scenarios', diff.removedScenarios],
    ['changed scenarios', diff.changedScenarios],
    ['changed state transitions', diff.changedTransitions],
    ['added standards requirements', diff.addedStandards],
    ['removed standards requirements', diff.removedStandards],
    ['changed standards requirements', diff.changedStandards],
    ['added TypeSpec declarations', diff.addedDeclarations],
    ['removed TypeSpec declarations', diff.removedDeclarations],
    ['changed TypeSpec declarations', diff.changedDeclarations],
    ['added TypeSpec deprecations', diff.addedDeprecations],
    ['removed TypeSpec deprecations', diff.removedDeprecations],
  ]
  const lines = groups
    .filter(([, entries]) => entries.length > 0)
    .map(([label, entries]) => `${label}:\n${entries.map((entry) => `  ${entry}`).join('\n')}`)
  return lines.length === 0
    ? `no normative specification change against ${ref}`
    : `normative specification change against ${ref}\n\n${lines.join('\n\n')}`
}

/**
 * The trees a specification can live in. A revision from before the prose moved
 * holds both under `spec/`, so both names stay listed and history keeps reading.
 */
const SPECIFICATION_TREES = ['docs', 'spec'] as const

export function isSpecificationSource(path: string): boolean {
  if (path.startsWith('spec/generated/')) return false
  // A revision from before the per-kind split still holds SPECIFICATION.md.
  // This tool reads history, so it keeps understanding that shape even though
  // nothing writes it any more.
  if (path.endsWith('/SPECIFICATION.md')) return true
  if (path.endsWith('/scenarios.md') || path === 'docs/scenarios.md') return true
  // 全体の文書は、`docs/` 直下、次に `docs/domain/` 直下に置いていた。
  if (
    [
      'docs/glossary.md',
      'docs/standards.md',
      'docs/structure.md',
      'docs/scenarios.feature.md',
      'docs/product-overview.md',
      'docs/domain/glossary.md',
      'docs/domain/standards.md',
      'docs/domain/structure.md',
      'docs/domain/scenarios.feature.md',
    ].includes(path)
  )
    return true
  return path.endsWith('.tsp') || documentKind(path) !== undefined
}

export async function readWorkingTree(root: string): Promise<Snapshot> {
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
  // Prose lives under docs/, the TypeSpec contract under spec/. Walking only
  // one of them reads as "nothing changed" rather than as an error, which is
  // how this tool silently stopped seeing Markdown when the prose moved.
  for (const tree of SPECIFICATION_TREES) await walk(resolve(root, tree))
  return snapshot
}

export async function diffWorkspaceSpecifications(
  root: string,
  ref = 'main',
): Promise<SpecificationDiff> {
  return diffSpecifications(readRevision(root, ref), await readWorkingTree(root))
}

function git(root: string, args: string[]): string {
  const result = Bun.spawnSync(['git', ...args], { cwd: root })
  if (result.exitCode !== 0) {
    throw new Error(`git ${args.join(' ')} failed: ${result.stderr.toString().trim()}`)
  }
  return result.stdout.toString()
}

export function readRevision(root: string, ref: string): Snapshot {
  const snapshot: Snapshot = new Map()
  const paths = git(root, ['ls-tree', '-r', '-z', '--name-only', ref, '--', ...SPECIFICATION_TREES])
    .split('\0')
    .filter(isSpecificationSource)
  if (paths.length === 0) return snapshot
  // ファイルごとに git を起動すると、一つのリビジョンを読むだけで数百回の起動になる。
  const batch = Bun.spawnSync(['git', 'cat-file', '--batch'], {
    cwd: root,
    stdin: Buffer.from(paths.map((path) => `${ref}:${path}\n`).join('')),
  })
  if (batch.exitCode !== 0) {
    throw new Error(`git cat-file --batch failed: ${batch.stderr.toString().trim()}`)
  }
  const output = batch.stdout
  let offset = 0
  for (const path of paths) {
    // 各項目は `<object> <type> <size>` の行、本文、改行の順に並ぶ。
    const headerEnd = output.indexOf(0x0a, offset)
    const size = Number(output.subarray(offset, headerEnd).toString().split(' ')[2])
    if (headerEnd < 0 || !Number.isInteger(size)) {
      throw new Error(`git cat-file --batch returned no object for ${ref}:${path}`)
    }
    snapshot.set(path, output.subarray(headerEnd + 1, headerEnd + 1 + size).toString())
    offset = headerEnd + 1 + size + 1
  }
  return snapshot
}

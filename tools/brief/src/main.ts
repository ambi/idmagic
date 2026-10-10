#!/usr/bin/env bun

/**
 * 規範参照と、一次情報が持つ作業条件から着手時の候補を組み立てる。
 * 名前の言及と配置から導く候補は、実装や検証の完了を証明しない。
 */

import { readdir, readFile, realpath } from 'node:fs/promises'
import { basename, relative, resolve } from 'node:path'
import { parseFrontmatterAndMarkdown } from '../../check/src/work-item-markdown.ts'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { deriveContext, neighborCandidates, renderContext } from './context.ts'
import { collectTraces } from './traces.ts'
import {
  COVERAGE_NOTE,
  coverageLine,
  coverageOf,
  extractDeclaration,
  initialContextDraft,
  partitionSources,
  sourceDirectories,
} from './brief.ts'

const root = resolve(process.env.SPEC_WORKSPACE_ROOT ?? resolve(import.meta.dir, '../../..'))
const argument = process.argv[2]
const usage =
  'Usage: brief <work-item> [--kind <kind>] [--target <path>] [--concern boundary|effects|contract|security|performance] [--include history|neighbors]\n'
if (argument === undefined || argument === '' || argument === '--help') {
  process.stderr.write(usage)
  process.exit(argument === '--help' ? 0 : 2)
}

async function findWorkItem(name: string): Promise<string> {
  const stem = basename(name).replace(/\.md$/, '')
  for (const directory of ['work-items/active', 'work-items/done']) {
    for (const entry of await readdir(resolve(root, directory))) {
      if (!entry.endsWith('.md')) continue
      if (entry === `${stem}.md` || entry.startsWith(`${stem}-`)) return `${directory}/${entry}`
    }
  }
  throw new Error(`no work item matches ${name}`)
}

function git(...args: string[]): string {
  const result = Bun.spawnSync(['git', ...args], { cwd: root })
  return result.exitCode === 0 ? result.stdout.toString().trimEnd() : ''
}

/**
 * Files naming an identifier the traceability walk does not index.
 *
 * That walk collects `REQ-*` and `EX-*`, which is what the Traceability page
 * draws. A standards requirement is a normative reference too, and a work item
 * may name one instead — so when the index has nothing, the working tree is
 * asked directly rather than reporting that nothing exists.
 */
function namingFiles(id: string, word = false): { sources: string[]; workItems: string[] } {
  const found = git(
    'grep',
    '-l',
    '--fixed-strings',
    ...(word ? ['-w'] : []),
    id,
    '--',
    ':!docs',
    ':!spec',
  )
  const paths = found === '' ? [] : found.split('\n')
  return {
    sources: paths.filter((one) => !one.startsWith('work-items/')),
    workItems: paths.filter((one) => one.startsWith('work-items/')),
  }
}

/** Where TypeSpec declares a symbol, as `<path>:<line>`. */
async function declarationOf(
  symbol: string,
  files: readonly string[],
): Promise<string | undefined> {
  const name = symbol.split('.').at(-1) ?? symbol
  const pattern = new RegExp(
    `^\\s*(?:@\\w+\\s+)*(?:model|op|interface|enum|union|namespace|scalar|alias)\\s+${name}\\b`,
  )
  for (const file of files) {
    const lines = (await readFile(resolve(root, file), 'utf8')).split('\n')
    const index = lines.findIndex((line) => pattern.test(line))
    if (index >= 0) return `${file}:${index + 1}`
  }
  return undefined
}

async function typespecFiles(directory: string, found: string[] = []): Promise<string[]> {
  for (const entry of await readdir(resolve(root, directory), { withFileTypes: true })) {
    if (entry.name === 'generated') continue
    const path = `${directory}/${entry.name}`
    if (entry.isDirectory()) await typespecFiles(path, found)
    else if (entry.name.endsWith('.tsp')) found.push(path)
  }
  return found
}

const path = await findWorkItem(argument)
const record = parseFrontmatterAndMarkdown(path, await readFile(resolve(root, path), 'utf8')) as {
  affected_spec?: Array<{ path?: string; requirement?: string; symbol?: string }>
  spec_impact?: { kind?: string; reason?: string }
  change_kind?: string
}
const affected = record.affected_spec ?? []
const seed = {
  kind: record.change_kind ?? 'feature',
  targets: [] as string[],
  concerns: [] as string[],
}
const include = new Set<string>()
try {
  const args = process.argv.slice(3)
  for (let index = 0; index < args.length; index += 2) {
    const option = args[index]
    const value = args[index + 1]
    if (!value || value.startsWith('--')) throw new Error(`Missing value for ${option}`)
    if (option === '--kind') {
      if (
        !['feature', 'bugfix', 'refactor', 'docs', 'tooling', 'maintenance', 'operations'].includes(
          value,
        )
      )
        throw new Error(`Unknown kind: ${value}`)
      seed.kind = value
    } else if (option === '--target') {
      const target = relative(root, resolve(root, value))
      const actual = relative(await realpath(root), await realpath(resolve(root, target)))
      if (
        target.startsWith('../') ||
        actual.startsWith('../') ||
        target === '..' ||
        actual === '..' ||
        target === ''
      )
        throw new Error(`Target must stay within the repository: ${value}`)
      seed.targets.push(target)
    } else if (option === '--concern') {
      if (!['boundary', 'effects', 'contract', 'security', 'performance'].includes(value))
        throw new Error(`Unknown concern: ${value}`)
      seed.concerns.push(value)
    } else if (option === '--include') {
      if (!['history', 'neighbors'].includes(value)) throw new Error(`Unknown expansion: ${value}`)
      include.add(value)
    } else throw new Error(`Unknown option: ${option}`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n${usage}`)
  process.exit(2)
}
seed.targets = [
  ...new Set([
    ...seed.targets,
    ...affected.map((reference) => reference.path ?? '').filter(Boolean),
  ]),
]
const workspace = createWorkspaceSnapshot(root)
const context = await deriveContext(seed, workspace)

const lines: string[] = [`# ${basename(path, '.md')}`, '', `Record: ${path}`, '']
lines.push(
  `Kind: ${seed.kind}; targets: ${seed.targets.join(', ') || 'none (supply --target)'}; concerns: ${seed.concerns.join(', ') || 'none'}`,
  '',
  ...renderContext(context),
)
if (affected.length === 0) {
  const impact = record.spec_impact
  lines.push(
    `No \`affected_spec\` references${impact?.kind ? ` (spec_impact: ${impact.kind})` : ''}.`,
    'Task context is derived from the kind and explicit targets above.',
    '',
  )
} else {
  // 判定の意味は 1 回だけ言う。記号ごとに繰り返せば、出力を読む時間が増える。
  lines.push(COVERAGE_NOTE, '')
}

const traces = new Map(
  (affected.some((one) => one.requirement) ? await collectTraces(root) : []).map((trace) => [
    trace.id,
    trace,
  ]),
)
const tspFiles = affected.some((one) => one.symbol) ? await typespecFiles('spec') : []

const specification: string[] = []
const typespec: string[] = []
const implementation = new Set<string>()
const tests = new Set<string>()
const modules = new Set<string>()

for (const reference of affected) {
  const documentPath = reference.path ?? ''
  modules.add(documentPath.match(/^docs\/modules\/([^/]+)\//)?.[1] ?? '')

  if (reference.requirement) {
    const id = reference.requirement
    specification.push(`${documentPath}#${id}`)
    lines.push(`## ${id}`, '', `Declared in ${documentPath}`, '')
    const source = await readFile(resolve(root, documentPath), 'utf8').catch(() => '')
    const declaration = extractDeclaration(source, id)
    lines.push(
      declaration ? '```markdown' : '',
      declaration ?? `The document declares no ${id}.`,
      declaration ? '```' : '',
      '',
    )
    const trace = traces.get(id) ?? namingFiles(id)
    const partition = partitionSources(trace.sources)
    for (const one of partition.implementation) implementation.add(one)
    for (const one of partition.tests) tests.add(one)
    lines.push(
      `- Named by tests: ${partition.tests.join(', ') || 'none'}`,
      `- Named by implementation: ${partition.implementation.join(', ') || 'none'}`,
      `- Named by work items: ${trace.workItems.join(', ') || 'none'}`,
      `- Coverage: ${coverageLine(coverageOf(declaration, trace.sources))}`,
      '',
    )
  }

  if (reference.symbol) {
    typespec.push(reference.symbol)
    const declaration = await declarationOf(reference.symbol, tspFiles)
    // 完全修飾名は Go にも TypeScript にも現れないので、宣言された名前で問う。
    // `declarationOf` が宣言を探すときと同じ名前であり、答えの主語が揃う。
    const name = reference.symbol.split('.').at(-1) ?? reference.symbol
    const naming = namingFiles(name, true).sources
    lines.push(
      `## ${reference.symbol}`,
      '',
      `- Declared at: ${declaration ?? `not found under spec/ (record names ${documentPath})`}`,
      `- Coverage: ${coverageLine(coverageOf(declaration, naming))}`,
      '',
    )
  }
}

const paths = seed.targets
if (include.has('history') && paths.length > 0) {
  lines.push(
    '## Recent history',
    '',
    '```',
    git('log', '-5', '--oneline', '--', ...paths),
    '```',
    '',
  )
}

// A work item that touched the same module is the nearest precedent, and the
// most recent ones are the ones whose conventions still hold.
const named = new Set<string>()
for (const module of include.has('history') ? modules : []) {
  if (module === '') continue
  const result = Bun.spawnSync(
    ['git', 'log', '-5', '--format=%h %s', '--', `docs/modules/${module}`],
    {
      cwd: root,
    },
  )
  for (const line of result.stdout.toString().trimEnd().split('\n')) {
    if (line !== '') named.add(line)
  }
}
if (named.size > 0) {
  lines.push('## Recent work on the same module', '', ...[...named].map((one) => `- ${one}`), '')
}

if (include.has('neighbors')) {
  lines.push(
    '## Neighbor candidates',
    '',
    'Inferred from public contract text; these are not required reading or verified dependencies.',
    '',
  )
  for (const candidate of await neighborCandidates(context, workspace))
    lines.push(
      `- ${candidate.location} [inferred]; origin: ${candidate.origin}; reason: ${candidate.reason}`,
    )
  lines.push('')
}

// The draft points at packages on the source side and at the naming files on
// the test side: the code to read is a package, and the test to re-run is a file.
lines.push(
  '## Draft reading list',
  '',
  '```yaml',
  initialContextDraft({
    specification: [
      ...new Set([
        ...specification,
        ...context.reading
          .filter((one) => one.location.startsWith('docs/'))
          .map((one) => one.location.split('#')[0]!),
      ]),
    ],
    typespec,
    implementation: [
      ...new Set([
        ...sourceDirectories([...implementation, ...tests]),
        ...seed.targets.filter((one) => !one.startsWith('docs/') && !one.startsWith('spec/')),
        ...context.reading
          .filter((one) => !one.location.startsWith('docs/') && !one.location.startsWith('spec/'))
          .map((one) => one.location),
      ]),
    ],
    tests: [...tests].sort(),
  }).trimEnd(),
  '```',
  '',
)

process.stdout.write(
  `${lines.filter((line, index, all) => line !== '' || all[index - 1] !== '').join('\n')}\n`,
)

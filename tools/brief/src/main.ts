#!/usr/bin/env bun

/**
 * Print what a work item should be read from, in one command.
 *
 * Everything here is derived: the normative references come from the record's
 * own `affected_spec`, the declarations from the canonical documents, the
 * files that already name each requirement from the traceability walk the
 * generated site uses, and the recent history from Git. Nothing is authored,
 * so nothing can go stale.
 */

import { readdir, readFile } from 'node:fs/promises'
import { basename, resolve } from 'node:path'
import { parseFrontmatterAndMarkdown } from '../../check/src/main.ts'
import { collectTraces } from '../../render-spec-docs/src/traces.ts'
import {
  COVERAGE_NOTE,
  coverageLine,
  coverageOf,
  extractDeclaration,
  initialContextDraft,
  partitionSources,
  sourceDirectories,
} from './brief.ts'

const root = resolve(import.meta.dir, '../../..')
const argument = process.argv[2]
if (argument === undefined || argument === '' || argument === '--help') {
  process.stderr.write('Usage: brief <work-item>\n')
  process.exit(argument === '--help' ? 0 : 2)
}

async function findWorkItem(name: string): Promise<string> {
  const stem = basename(name).replace(/\.md$/, '')
  for (const directory of ['work-items', 'work-items/done']) {
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
}
const affected = record.affected_spec ?? []

const lines: string[] = [`# ${basename(path, '.md')}`, '', `Record: ${path}`, '']
if (affected.length === 0) {
  const impact = record.spec_impact
  lines.push(
    `No \`affected_spec\` references${impact?.kind ? ` (spec_impact: ${impact.kind})` : ''}.`,
    'The brief derives its reading list from those references, so there is nothing to resolve.',
    '',
  )
} else {
  // 判定の意味は 1 回だけ言う。記号ごとに繰り返せば、出力を読む時間が増える。
  lines.push(COVERAGE_NOTE, '')
}

const traces = new Map((await collectTraces(root)).map((trace) => [trace.id, trace]))
const tspFiles = await typespecFiles('spec')

const specification: string[] = []
const typespec: string[] = []
const implementation = new Set<string>()
const tests = new Set<string>()
const contexts = new Set<string>()

for (const reference of affected) {
  const documentPath = reference.path ?? ''
  contexts.add(documentPath.match(/^docs\/contexts\/([^/]+)\//)?.[1] ?? '')

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

const paths = [...new Set(affected.map((reference) => reference.path ?? '').filter(Boolean))]
if (paths.length > 0) {
  lines.push(
    '## Recent history',
    '',
    '```',
    git('log', '-5', '--oneline', '--', ...paths),
    '```',
    '',
  )
}

// A work item that touched the same context is the nearest precedent, and the
// most recent ones are the ones whose conventions still hold.
const named = new Set<string>()
for (const context of contexts) {
  if (context === '') continue
  const result = Bun.spawnSync(
    ['git', 'log', '-5', '--format=%h %s', '--', `docs/contexts/${context}`],
    {
      cwd: root,
    },
  )
  for (const line of result.stdout.toString().trimEnd().split('\n')) {
    if (line !== '') named.add(line)
  }
}
if (named.size > 0) {
  lines.push('## Recent work on the same context', '', ...[...named].map((one) => `- ${one}`), '')
}

// The draft points at packages on the source side and at the naming files on
// the test side: the code to read is a package, and the test to re-run is a file.
lines.push(
  '## Draft reading list',
  '',
  '```yaml',
  initialContextDraft({
    specification,
    typespec,
    implementation: sourceDirectories([...implementation, ...tests]),
    tests: [...tests].sort(),
  }).trimEnd(),
  '```',
  '',
)

process.stdout.write(
  `${lines.filter((line, index, all) => line !== '' || all[index - 1] !== '').join('\n')}\n`,
)

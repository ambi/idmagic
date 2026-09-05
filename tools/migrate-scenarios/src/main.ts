#!/usr/bin/env bun

import { Glob } from 'bun'
import { readFile, unlink, writeFile } from 'node:fs/promises'
import { relative, resolve } from 'node:path'
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'

type LegacyStep = {
  keyword: 'GIVEN' | 'WHEN' | 'THEN'
  text: string
  alternatives: string[]
}

type LegacyRule = {
  id: string
  title: string
  actor?: string
  description: string[]
  steps: LegacyStep[]
  supersededBy?: string
}

function parseRule(block: string, path: string): LegacyRule {
  const lines = block.trimEnd().split('\n')
  const heading = lines[0]?.match(/^### (REQ-[A-Z0-9-]+): (.+)$/)
  if (!heading?.[1] || !heading[2]) throw new Error(`${path}: invalid legacy rule heading`)
  const supersededBy = heading[2].match(/\(superseded by (REQ-[A-Z0-9-]+)\)$/)?.[1]
  const description: string[] = []
  const steps: LegacyStep[] = []
  let actor: string | undefined
  for (const [index, line] of lines.slice(1).entries()) {
    const lineNumber = index + 2
    const actorMatch = line.match(/^- ACTOR (.+)$/)
    if (actorMatch?.[1]) {
      actor = actorMatch[1]
      continue
    }
    const stepMatch = line.match(/^- (GIVEN|WHEN|THEN) (.+)$/)
    if (stepMatch?.[1] && stepMatch[2]) {
      steps.push({
        keyword: stepMatch[1] as LegacyStep['keyword'],
        text: stepMatch[2],
        alternatives: [],
      })
      continue
    }
    const alternative = line.match(/^  - ALT (.+)$/)?.[1]
    if (alternative) {
      const parent = steps.at(-1)
      if (!parent || parent.keyword === 'GIVEN') {
        throw new Error(`${path}:${lineNumber}: ALT has no WHEN or THEN parent`)
      }
      if (!alternative.includes(' → ')) {
        throw new Error(`${path}:${lineNumber}: ALT has no result separator`)
      }
      parent.alternatives.push(alternative)
      continue
    }
    if (line.trim()) description.push(line)
  }
  if (!supersededBy && (!actor || steps.length === 0)) {
    throw new Error(`${path}: ${heading[1]} is live but has no actor or steps`)
  }
  return { id: heading[1], title: heading[2], actor, description, steps, supersededBy }
}

function stepLine(step: LegacyStep, firstOfKind: boolean): string {
  const keyword =
    step.keyword === 'GIVEN'
      ? firstOfKind
        ? 'Given'
        : 'And'
      : step.keyword === 'WHEN'
        ? 'When'
        : 'Then'
  return `- ${keyword} ${step.text}`
}

function renderSteps(steps: readonly LegacyStep[]): string[] {
  const seen = new Set<LegacyStep['keyword']>()
  return steps.map((step) => {
    const line = stepLine(step, !seen.has(step.keyword))
    seen.add(step.keyword)
    return line
  })
}

function renderRule(rule: LegacyRule): { source: string; examples: number; alternatives: number } {
  const output = [`## Rule: ${rule.id} ${rule.title}`, '']
  if (rule.description.length > 0) output.push(...rule.description, '')
  if (rule.actor) output.push(`Primary actor: \`${rule.actor}\``, '')
  if (rule.supersededBy) return { source: output.join('\n'), examples: 0, alternatives: 0 }

  let sequence = 1
  output.push(
    `### Example: ${rule.id.replace(/^REQ-/, 'EX-')}-${String(sequence).padStart(2, '0')} 通常経路`,
    '',
  )
  output.push(...renderSteps(rule.steps), '')

  let alternatives = 0
  for (const [parentIndex, parent] of rule.steps.entries()) {
    for (const alternative of parent.alternatives) {
      alternatives += 1
      sequence += 1
      const [condition = '', ...results] = alternative.split(' → ')
      const id = `${rule.id.replace(/^REQ-/, 'EX-')}-${String(sequence).padStart(2, '0')}`
      output.push(`### Example: ${id} ${condition}`, '')
      const prefix =
        parent.keyword === 'WHEN'
          ? rule.steps.slice(0, parentIndex + 1)
          : rule.steps.slice(0, parentIndex)
      output.push(...renderSteps(prefix))
      output.push(`- ${parent.keyword === 'WHEN' ? 'But' : 'Then'} ${condition}`)
      for (const [index, result] of results.entries()) {
        output.push(`- ${index === 0 ? 'Then' : 'And'} ${result}`)
      }
      output.push('')
    }
  }
  return { source: output.join('\n'), examples: sequence, alternatives }
}

function semanticFragments(rule: LegacyRule): string[] {
  return [
    ...rule.steps.map((step) => step.text),
    ...rule.steps.flatMap((step) =>
      step.alternatives.flatMap((alternative) => alternative.split(' → ')),
    ),
  ]
}

const glob = new Glob('docs/**/scenarios.feature.md')
const paths = [...glob.scanSync({ cwd: WORKSPACE_ROOT, onlyFiles: true })].sort()
let rules = 0
let liveRules = 0
let examples = 0
let alternatives = 0
for (const relativePath of paths) {
  const path = resolve(WORKSPACE_ROOT, relativePath)
  const source = await readFile(path, 'utf8')
  const starts = [...source.matchAll(/^### REQ-[A-Z0-9-]+: .+$/gm)].map((match) => match.index ?? 0)
  const legacyRules = starts.map((start, index) =>
    parseRule(source.slice(start, starts[index + 1] ?? source.length), relativePath),
  )
  const firstHeading = starts[0] ?? source.length
  const preamble = source.slice(0, firstHeading).trimEnd().split('\n')
  const title = preamble.shift()?.replace(/^# (?:Feature: )?/, '') ?? relativePath
  const body = [`# Feature: ${title}`, '', ...preamble.filter((line) => line.trim())]
  if (body.at(-1) !== '') body.push('')
  for (const rule of legacyRules) {
    const rendered = renderRule(rule)
    body.push(rendered.source)
    rules += 1
    if (!rule.supersededBy) liveRules += 1
    examples += rendered.examples
    alternatives += rendered.alternatives
  }
  const migrated = `${body.join('\n').trimEnd()}\n`
  for (const fragment of legacyRules.flatMap(semanticFragments)) {
    if (!migrated.includes(fragment)) {
      throw new Error(`${relativePath}: lost semantic fragment: ${fragment}`)
    }
  }
  const target = resolve(path, '..', 'scenarios.feature.md')
  await writeFile(target, migrated)
  await unlink(path)
  console.log(`migrated ${relative(WORKSPACE_ROOT, path)} -> ${relative(WORKSPACE_ROOT, target)}`)
}
console.log(
  `migrated ${paths.length} files, ${rules} rules (${liveRules} live), ${alternatives} alternatives, ${examples} examples`,
)

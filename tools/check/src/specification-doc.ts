export type SpecificationFinding = {
  line: number
  message: string
}

export type SpecificationValidation = {
  findings: SpecificationFinding[]
  scenarioIds: Array<{ id: string; line: number; supersededBy?: string }>
}

/** A retired scenario keeps its heading and names its successor instead of carrying steps. */
const SUPERSEDED_HEADING = /^### REQ-[A-Z0-9-]+: .+ \(superseded by (REQ-[A-Z0-9-]+)\)$/

const SECTION_ORDER = [
  'Overview',
  'Glossary',
  'Standards',
  'State Transitions',
  'Design',
  'Scenarios',
] as const

type SectionName = (typeof SECTION_ORDER)[number]

const STANDARDS_HEADER = '| ID | Adoption | Strength | Statement |'

/** Whether the product takes the standard's capability at all. */
const ADOPTION_VALUES = new Set(['required', 'optional', 'partial', 'excluded'])

/** How firmly the product holds the rule once taken. */
const STRENGTH_VALUES = new Set(['MUST', 'MUST NOT', 'SHOULD', 'MAY'])

/** An excluded capability carries no obligation, so these strengths cannot describe one. */
const OBLIGATION_STRENGTHS = new Set(['MUST', 'SHOULD'])

/** Splits a Markdown table row into trimmed cells, dropping the empty edges. */
function tableRowCells(line: string): string[] {
  return line
    .split('|')
    .slice(1, -1)
    .map((cell) => cell.trim())
}

function isSeparatorRow(line: string): boolean {
  return /^\|(?:\s*:?-+:?\s*\|)+$/.test(line)
}

function lineAt(source: string, offset: number): number {
  return source.slice(0, offset).split('\n').length
}

function sectionBody(
  source: string,
  sections: Array<{ name: string; index: number; bodyStart: number }>,
  name: string,
): { body: string; offset: number } | undefined {
  const index = sections.findIndex((section) => section.name === name)
  if (index < 0) return undefined
  const section = sections[index]
  if (!section) return undefined
  return {
    body: source.slice(section.bodyStart, sections[index + 1]?.index ?? source.length),
    offset: section.bodyStart,
  }
}

function validateScenario(
  block: string,
  offset: number,
  source: string,
  findings: SpecificationFinding[],
): void {
  const heading = block.match(/^### (REQ-[A-Z0-9-]+): .+$/m)
  if (!heading) {
    findings.push({
      line: lineAt(source, offset),
      message: 'scenario must start with a REQ heading',
    })
    return
  }

  for (const match of block.matchAll(/^- (Actor|Given|When|Then|Alternative|Alt)(?::| \()/gm)) {
    findings.push({
      line: lineAt(source, offset + (match.index ?? 0)),
      message: `scenario keyword ${match[1]} must be uppercase and must not use a colon`,
    })
  }

  const actors = [...block.matchAll(/^- ACTOR .+$/gm)]
  if (actors.length !== 1) {
    findings.push({
      line: lineAt(source, offset),
      message: 'scenario must contain exactly one ACTOR line',
    })
  }

  const whenMatches = [...block.matchAll(/^- WHEN .+$/gm)]
  if (whenMatches.length === 0) {
    findings.push({
      line: lineAt(source, offset),
      message: 'scenario must contain at least one WHEN line',
    })
  }

  const thenMatches = [...block.matchAll(/^- THEN .+$/gm)]
  if (thenMatches.length === 0) {
    findings.push({
      line: lineAt(source, offset),
      message: 'scenario must contain at least one THEN line',
    })
  }
  const stepMatches = [...block.matchAll(/^- (WHEN|THEN) .+$/gm)]
  if (stepMatches[0]?.[1] !== 'WHEN') {
    findings.push({
      line: lineAt(source, offset),
      message: 'the first behavior step must be WHEN',
    })
  }

  const topLevelClauses = [...block.matchAll(/^- (ACTOR|GIVEN|WHEN|THEN) .+$/gm)]
  let behaviorStarted = false
  for (const [index, clause] of topLevelClauses.entries()) {
    const keyword = clause[1]
    if (keyword === 'ACTOR' && index !== 0) {
      findings.push({
        line: lineAt(source, offset + (clause.index ?? 0)),
        message: 'ACTOR must be the first scenario clause',
      })
    }
    if (keyword === 'GIVEN' && behaviorStarted) {
      findings.push({
        line: lineAt(source, offset + (clause.index ?? 0)),
        message: 'GIVEN must appear before WHEN and THEN clauses',
      })
    }
    if (keyword === 'WHEN' || keyword === 'THEN') behaviorStarted = true
  }

  for (const match of block.matchAll(/^  - ALT (.+)$/gm)) {
    if (!match[1]?.includes(' → ')) {
      findings.push({
        line: lineAt(source, offset + (match.index ?? 0)),
        message: 'ALT must separate its condition and result with →',
      })
    }
    const before = block
      .slice(0, match.index ?? 0)
      .trimEnd()
      .split('\n')
    const parent = [...before].reverse().find((line) => !line.startsWith('  - ALT '))
    if (!parent || !/^- (?:WHEN|THEN) .+$/.test(parent)) {
      findings.push({
        line: lineAt(source, offset + (match.index ?? 0)),
        message: 'ALT must be nested immediately below a WHEN or THEN step',
      })
    }
  }

  for (const match of block.matchAll(/^(\s*)- ALT .+$/gm)) {
    if (match[1]?.length !== 2) {
      findings.push({
        line: lineAt(source, offset + (match.index ?? 0)),
        message: 'ALT must be a two-space-indented child of a WHEN or THEN step',
      })
    }
  }

  for (const match of block.matchAll(/^- (?:WHEN|THEN) \(\d+\) .+$/gm)) {
    findings.push({
      line: lineAt(source, offset + (match.index ?? 0)),
      message: 'WHEN and THEN must not use local step numbers',
    })
  }

  for (const match of block.matchAll(/^  - ALT \(\d+\) .+$/gm)) {
    findings.push({
      line: lineAt(source, offset + (match.index ?? 0)),
      message: 'ALT must not use local step numbers',
    })
  }
}

/**
 * Standards rows are contract data: two closed vocabularies and an ID other documents cite.
 * Adoption and Strength are independent axes, so only the pairing that cannot mean anything —
 * an obligation attached to a capability the product does not provide — is rejected.
 */
function validateStandards(
  body: string,
  offset: number,
  source: string,
  findings: SpecificationFinding[],
): void {
  const standards = [...body.matchAll(/^### .+$/gm)]
  const seenIds = new Map<string, number>()
  for (const [index, standard] of standards.entries()) {
    const start = (standard.index ?? 0) + standard[0].length
    const end = standards[index + 1]?.index ?? body.length
    const block = body.slice(start, end)
    if (!block.includes(STANDARDS_HEADER)) {
      findings.push({
        line: lineAt(source, offset + (standard.index ?? 0)),
        message: `standard must use ${STANDARDS_HEADER}`,
      })
      continue
    }
    for (const match of block.matchAll(/^\|.*\|$/gm)) {
      const line = match[0]
      if (line === STANDARDS_HEADER || isSeparatorRow(line)) continue
      const at = lineAt(source, offset + start + (match.index ?? 0))
      const [id, adoption, strength] = tableRowCells(line)
      if (!id || !adoption || !strength) continue
      const previous = seenIds.get(id)
      if (previous !== undefined) {
        findings.push({
          line: at,
          message: `duplicate standard id ${id} (first seen on line ${previous})`,
        })
      } else {
        seenIds.set(id, at)
      }
      if (!ADOPTION_VALUES.has(adoption)) {
        findings.push({
          line: at,
          message: `${id} has Adoption "${adoption}"; use one of ${[...ADOPTION_VALUES].join(', ')}`,
        })
      }
      if (!STRENGTH_VALUES.has(strength)) {
        findings.push({
          line: at,
          message: `${id} has Strength "${strength}"; use one of ${[...STRENGTH_VALUES].join(', ')}`,
        })
      }
      if (adoption === 'excluded' && OBLIGATION_STRENGTHS.has(strength)) {
        findings.push({
          line: at,
          message: `${id} is excluded, so it cannot carry the obligation "${strength}"`,
        })
      }
    }
  }
}

export function validateSpecification(source: string): SpecificationValidation {
  const findings: SpecificationFinding[] = []
  const frontmatter = source.match(/^---\n([\s\S]*?)\n---\n/)
  if (!frontmatter) {
    findings.push({ line: 1, message: 'SPECIFICATION.md requires YAML frontmatter' })
  } else {
    const parsed = Bun.YAML.parse(frontmatter[1] ?? '') as Record<string, unknown>
    if (typeof parsed.context !== 'string' || !/^[a-z0-9][a-z0-9-]*$/.test(parsed.context)) {
      findings.push({ line: 2, message: 'frontmatter context must be a lowercase slug' })
    }
    if (typeof parsed.updated_at !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(parsed.updated_at)) {
      findings.push({ line: 3, message: 'frontmatter updated_at must be YYYY-MM-DD' })
    }
  }

  const titles = [...source.matchAll(/^# (?!#).+$/gm)]
  if (titles.length !== 1)
    findings.push({ line: 1, message: 'document must contain exactly one H1' })

  const sections = [...source.matchAll(/^## (.+)$/gm)].map((match) => ({
    name: match[1]?.trim() ?? '',
    index: match.index ?? 0,
    bodyStart: (match.index ?? 0) + match[0].length,
  }))
  const seenSections = new Set<string>()
  let previousOrder = -1
  for (const section of sections) {
    const order = SECTION_ORDER.indexOf(section.name as SectionName)
    if (order < 0) {
      findings.push({
        line: lineAt(source, section.index),
        message: `unknown top-level section ${section.name}`,
      })
      continue
    }
    if (seenSections.has(section.name)) {
      findings.push({
        line: lineAt(source, section.index),
        message: `duplicate top-level section ${section.name}`,
      })
    }
    if (order < previousOrder) {
      findings.push({
        line: lineAt(source, section.index),
        message: `${section.name} is out of canonical section order`,
      })
    }
    seenSections.add(section.name)
    previousOrder = Math.max(previousOrder, order)
  }
  if (!seenSections.has('Overview'))
    findings.push({ line: 1, message: 'Overview section is required' })

  for (const match of source.matchAll(/\[[^\]]+\]\([^\n)]*decisions\/[^\n)]*\)/g)) {
    findings.push({
      line: lineAt(source, match.index ?? 0),
      message: 'current specification must be self-contained and must not link to decisions/',
    })
  }

  const scenarioIds = [...source.matchAll(/^### (REQ-[A-Z0-9-]+): .+$/gm)].map((match) => ({
    id: match[1] ?? '',
    line: lineAt(source, match.index ?? 0),
    supersededBy: match[0].match(SUPERSEDED_HEADING)?.[1],
  }))
  const localIds = new Set<string>()
  for (const scenario of scenarioIds) {
    if (localIds.has(scenario.id)) {
      findings.push({ line: scenario.line, message: `duplicate scenario id ${scenario.id}` })
    }
    localIds.add(scenario.id)
  }

  const states = sectionBody(source, sections, 'State Transitions')
  if (states) {
    const machines = [...states.body.matchAll(/^### .+$/gm)]
    for (const [index, machine] of machines.entries()) {
      const start = (machine.index ?? 0) + machine[0].length
      const end = machines[index + 1]?.index ?? states.body.length
      const body = states.body.slice(start, end)
      if (!body.includes('| From | Event | Guard | To | Effects |')) {
        findings.push({
          line: lineAt(source, states.offset + (machine.index ?? 0)),
          message: 'state transition must use From | Event | Guard | To | Effects',
        })
      }
      for (const match of body.matchAll(/^\|[^|\n]*\|[^|\n]*\|\s*""\s*\|/gm)) {
        findings.push({
          line: lineAt(source, states.offset + start + (match.index ?? 0)),
          message: 'unconditional state transition guard must use — instead of an empty string',
        })
      }
    }
  }

  const standards = sectionBody(source, sections, 'Standards')
  if (standards) validateStandards(standards.body, standards.offset, source, findings)

  const scenarioSection = sectionBody(source, sections, 'Scenarios')
  if (scenarioSection) {
    const starts = [...scenarioSection.body.matchAll(/^### /gm)].map((match) => match.index ?? 0)
    for (const [index, start] of starts.entries()) {
      const block = scenarioSection.body.slice(
        start,
        starts[index + 1] ?? scenarioSection.body.length,
      )
      if (SUPERSEDED_HEADING.test(block.split('\n')[0] ?? '')) continue
      validateScenario(block, scenarioSection.offset + start, source, findings)
    }
  }

  return { findings, scenarioIds }
}

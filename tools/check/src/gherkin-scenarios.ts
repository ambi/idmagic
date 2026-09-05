import { AstBuilder, compile, GherkinInMarkdownTokenMatcher, Parser } from '@cucumber/gherkin'
import {
  IdGenerator,
  type GherkinDocument,
  type Scenario,
  StepKeywordType,
} from '@cucumber/messages'
import type { SpecificationFinding } from './specification-doc.ts'

export type ScenarioStep = {
  keyword: string
  kind: 'context' | 'action' | 'outcome'
  text: string
  line: number
}

export type ScenarioExample = {
  id: string
  name: string
  line: number
  steps: ScenarioStep[]
  outline: boolean
  decisionTable: boolean
}

export type ScenarioRule = {
  id: string
  name: string
  line: number
  supersededBy?: string
  examples: ScenarioExample[]
}

export type ParsedScenarioDocument = {
  feature?: string
  rules: ScenarioRule[]
  findings: SpecificationFinding[]
  pickleCount: number
}

const RULE_ID = /^(REQ-[A-Z0-9-]+)(?::)?(?:\s+|$)/
const EXAMPLE_ID = /^(EX-[A-Z0-9-]+-\d+)(?:\s+|$)/
const SUPERSEDED = /\(superseded by (REQ-[A-Z0-9-]+)\)$/

function expectedExamplePrefix(ruleId: string): string {
  return `${ruleId.replace(/^REQ-/, 'EX-')}-`
}

function stepsOf(scenario: Scenario): ScenarioStep[] {
  let prior: ScenarioStep['kind'] = 'context'
  return scenario.steps.map((step) => {
    const kind =
      step.keywordType === StepKeywordType.ACTION
        ? 'action'
        : step.keywordType === StepKeywordType.OUTCOME
          ? 'outcome'
          : step.keywordType === StepKeywordType.CONTEXT
            ? 'context'
            : prior
    prior = kind
    return {
      keyword: step.keyword.trim(),
      kind,
      text: step.text,
      line: step.location.line,
    }
  })
}

function validatePathSteps(
  steps: readonly ScenarioStep[],
  line: number,
  findings: SpecificationFinding[],
): void {
  if (!steps.some((step) => step.kind === 'action')) {
    findings.push({ line, message: 'example must contain at least one When step' })
  }
  if (!steps.some((step) => step.kind === 'outcome')) {
    findings.push({ line, message: 'example must contain at least one Then step' })
  }
}

function rowValues(scenario: Scenario, examplesIndex: number): string[][] {
  const examples = scenario.examples[examplesIndex]
  if (!examples?.tableHeader) return []
  const headers = examples.tableHeader.cells.map((cell) => cell.value)
  return examples.tableBody.map((row) =>
    headers.map((_, index) => row.cells[index]?.value.trim() ?? ''),
  )
}

function decisionRowsOverlap(left: readonly string[], right: readonly string[]): boolean {
  return left.every(
    (value, index) => value === 'any' || right[index] === 'any' || value === right[index],
  )
}

/** Parse and validate the repository contract layered on official Markdown with Gherkin. */
export function parseScenarioDocument(source: string): ParsedScenarioDocument {
  const findings: SpecificationFinding[] = []
  const rules: ScenarioRule[] = []
  let document: GherkinDocument
  try {
    const parser = new Parser(
      new AstBuilder(IdGenerator.incrementing()),
      new GherkinInMarkdownTokenMatcher('en'),
    )
    document = parser.parse(source)
  } catch (error) {
    const located = error as Error & { location?: { line?: number } }
    return {
      rules,
      findings: [
        {
          line: located.location?.line ?? 1,
          message: `invalid Markdown with Gherkin: ${located.message}`,
        },
      ],
      pickleCount: 0,
    }
  }

  const feature = document.feature
  if (!feature) {
    return {
      rules,
      findings: [{ line: 1, message: 'scenario document must contain exactly one Feature' }],
      pickleCount: 0,
    }
  }

  const seenExampleIds = new Map<string, number>()
  for (const child of feature.children) {
    if (child.scenario) {
      findings.push({
        line: child.scenario.location.line,
        message: 'examples must belong to a Rule',
      })
      continue
    }
    if (child.background) {
      findings.push({
        line: child.background.location.line,
        message: 'Feature-level Background is not supported; keep setup inside each Rule',
      })
      continue
    }
    const rule = child.rule
    if (!rule) continue
    const match = rule.name.match(RULE_ID)
    if (!match?.[1]) {
      findings.push({ line: rule.location.line, message: 'Rule name must start with a REQ id' })
      continue
    }
    const ruleId = match[1]
    const supersededBy = rule.name.match(SUPERSEDED)?.[1]
    const parsedRule: ScenarioRule = {
      id: ruleId,
      name: rule.name,
      line: rule.location.line,
      supersededBy,
      examples: [],
    }

    for (const ruleChild of rule.children) {
      const scenario = ruleChild.scenario
      if (!scenario) continue
      const steps = stepsOf(scenario)
      validatePathSteps(steps, scenario.location.line, findings)
      const isOutline = scenario.keyword.trim() === 'Scenario Outline'
      if (!isOutline) {
        if (scenario.keyword.trim() !== 'Example') {
          findings.push({
            line: scenario.location.line,
            message: 'use Example for a concrete path or Scenario Outline for a table',
          })
        }
        const id = scenario.name.match(EXAMPLE_ID)?.[1]
        if (!id) {
          findings.push({
            line: scenario.location.line,
            message: 'Example name must start with an EX id',
          })
          continue
        }
        parsedRule.examples.push({
          id,
          name: scenario.name,
          line: scenario.location.line,
          steps,
          outline: false,
          decisionTable: false,
        })
      } else {
        if (scenario.examples.length === 0) {
          findings.push({
            line: scenario.location.line,
            message: 'Scenario Outline must contain at least one Examples table',
          })
        }
        for (const [examplesIndex, examples] of scenario.examples.entries()) {
          const headers = examples.tableHeader?.cells.map((cell) => cell.value.trim()) ?? []
          const idColumn = headers.indexOf('example_id')
          if (idColumn < 0) {
            findings.push({
              line: examples.location.line,
              message: 'Examples table must contain an example_id column',
            })
            continue
          }
          const decisionTable = examples.name === 'Decision table (Unique)'
          const outcomeParameters = new Set(
            steps
              .filter((step) => step.kind === 'outcome')
              .flatMap((step) =>
                [...step.text.matchAll(/<([^>]+)>/g)].map((item) => item[1] ?? ''),
              ),
          )
          const outcomeColumns = headers
            .map((header, index) => (outcomeParameters.has(header) ? index : -1))
            .filter((index) => index >= 0)
          const conditionColumns = headers
            .map((header, index) =>
              header !== 'example_id' && !outcomeParameters.has(header) ? index : -1,
            )
            .filter((index) => index >= 0)
          const seenConditions: string[][] = []
          for (const [rowIndex, values] of rowValues(scenario, examplesIndex).entries()) {
            const row = examples.tableBody[rowIndex]
            const id = values[idColumn] ?? ''
            if (!EXAMPLE_ID.test(`${id} `)) {
              findings.push({
                line: row?.location.line ?? examples.location.line,
                message: 'Examples row must contain an EX id in example_id',
              })
              continue
            }
            if (
              decisionTable &&
              (outcomeColumns.length === 0 || outcomeColumns.every((index) => !values[index]))
            ) {
              findings.push({
                line: row?.location.line ?? examples.location.line,
                message: `decision row ${id} must have a nonempty outcome`,
              })
            }
            if (decisionTable) {
              const conditions = conditionColumns.map((index) => values[index] ?? '')
              if (seenConditions.some((prior) => decisionRowsOverlap(prior, conditions))) {
                findings.push({
                  line: row?.location.line ?? examples.location.line,
                  message: 'Decision table (Unique) has overlapping condition values',
                })
              }
              seenConditions.push(conditions)
            }
            parsedRule.examples.push({
              id,
              name: scenario.name,
              line: row?.location.line ?? examples.location.line,
              steps,
              outline: true,
              decisionTable,
            })
          }
        }
      }
    }

    if (!supersededBy && parsedRule.examples.length === 0) {
      findings.push({
        line: rule.location.line,
        message: `${ruleId} must contain at least one example`,
      })
    }
    for (const example of parsedRule.examples) {
      if (!example.id.startsWith(expectedExamplePrefix(ruleId))) {
        findings.push({
          line: example.line,
          message: `${example.id} must belong to ${ruleId}`,
        })
      }
      const previous = seenExampleIds.get(example.id)
      if (previous !== undefined) {
        findings.push({
          line: example.line,
          message: `duplicate example id ${example.id}`,
        })
      } else {
        seenExampleIds.set(example.id, example.line)
      }
    }
    rules.push(parsedRule)
  }

  return {
    feature: feature.name,
    rules,
    findings,
    pickleCount: compile(document, 'scenarios.feature.md', IdGenerator.incrementing()).length,
  }
}

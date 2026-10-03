import { specificationRules, validateSpecificationDeclarations } from './feature-specification.ts'
import { parseScenarioDocument } from './gherkin-scenarios.ts'
export {
  canonicalDocumentNames,
  CONTEXT_DOCUMENTS,
  ROOT_DOCUMENTS,
  SYSTEM_DOCUMENT_DIRECTORIES,
  SYSTEM_DOCUMENT_PATHS,
} from '../../workspace/src/document-layout.ts'
import { canonicalDocumentNames, documentNames } from '../../workspace/src/document-layout.ts'

export type SpecificationFinding = {
  line: number
  message: string
}

export type SpecificationValidation = {
  findings: SpecificationFinding[]
  scenarioIds: Array<{ id: string; line: number; supersededBy?: string; title?: string }>
  exampleIds: Array<{ id: string; line: number; parentId: string }>
  standardIds: Array<{ id: string; line: number }>
  /** 付録の `Rule` が参照する規則。宣言ではないので、ID の重複には数えない。 */
  ruleReferences: Array<{ id: string; line: number; title: string }>
}

/**
 * The split layout names each file after the kind of content it holds, so the
 * file name replaces the section-set and section-order checks the single
 * canonical document needed. A directory takes the split layout as soon as it
 * holds a README.md.
 *
 * 新しい形式の機能ノードは、機能仕様（`README.md` と任意の名前の章）、内部設計
 * （`design.md`）、例の付録（`examples.feature.md`）を持つ。Context の `design/` は内部設計で、
 * そのうち `decisions.md` だけが判断の記録の骨格を持つ。
 */
/** What a file's name says about the grammar its body must follow. */
export type DocumentKind =
  | 'standards'
  | 'states'
  | 'scenarios'
  | 'specification'
  | 'examples'
  | 'decision-records'
  | 'prose'

const KIND_BY_NAME = new Map<string, DocumentKind>([
  ['standards.md', 'standards'],
  ['states.md', 'states'],
  ['scenarios.feature.md', 'scenarios'],
])

/** Context より下の段で、任意の名前の章として読む名前。 */
const CHAPTER_NAME = /^[a-z0-9]+(?:-[a-z0-9]+)*\.md$/

/**
 * The kind of a canonical document, or undefined when the path is not one.
 * `path` is repository-relative and uses forward slashes.
 *
 * `docs/contexts/` は `docs/domain/` へ改名する前の名前で、履歴を読む道具（`spec-diff`）が
 * その時点のリビジョンを規範文書として認識し続けるために読み替える。
 *
 * Context より下の段の種別は、Context の形式を問わずパスだけから決める。履歴や一つの
 * ファイルだけを読む道具が、隣のファイルを見ずに同じ答えを得られるようにするためである。
 * その段にそのファイルを置いてよいかは、段の集合を見る `verifyCanonicalDocumentSet` が決める。
 */
export function documentKind(path: string): DocumentKind | undefined {
  const name = path.split('/').at(-1) ?? ''
  const directory = path
    .slice(0, path.lastIndexOf('/'))
    .replace(/^docs\/contexts\//, 'docs/domain/')
  const below = directory.match(/^docs\/domain\/[^/]+\/(.+)$/)?.[1]?.split('/')
  if (below) {
    if (below[0] === 'design') {
      if (below.length !== 1 || !(name === 'README.md' || CHAPTER_NAME.test(name))) return undefined
      return name === 'decisions.md' ? 'decision-records' : 'prose'
    }
    // 共有語彙と採用した標準は機能をまたいで使うので、Context の直下にだけ置く。
    if (below.length > 2 || name === 'glossary.md' || name === 'standards.md') return undefined
    const named = KIND_BY_NAME.get(name)
    if (named) return named
    if (name === 'examples.feature.md') return 'examples'
    if (['decisions.md', 'internals.md', 'design.md'].includes(name)) return 'prose'
    return name === 'README.md' || CHAPTER_NAME.test(name) ? 'specification' : undefined
  }
  const allowed = directory.startsWith('docs/domain/')
    ? documentNames(directory)
    : canonicalDocumentNames(directory)
  if (!allowed?.includes(name)) return undefined
  return KIND_BY_NAME.get(name) ?? 'prose'
}

const STATE_HEADER = '| State | Kind | Meaning |'
const TRANSITION_HEADER = '| From | Event | Guard | To | Effects |'

/** Whether a state starts the machine, ends it, or does neither. */
const STATE_KINDS = new Set(['initial', 'terminal', '—'])

const STANDARDS_HEADER = '| ID | Adoption | Strength | Statement |'

/** Whether the product takes the standard's capability at all. */
const ADOPTION_VALUES = new Set(['required', 'optional', 'partial', 'excluded'])

/** How firmly the product holds the rule once taken. */
const STRENGTH_VALUES = new Set(['MUST', 'MUST NOT', 'SHOULD', 'MAY'])

/** An excluded capability carries no obligation, so these strengths cannot describe one. */
const OBLIGATION_STRENGTHS = new Set(['MUST', 'SHOULD'])

/**
 * Splits a Markdown table row into trimmed cells, dropping the empty edges.
 * A cell may contain an escaped pipe — a CEL guard writes disjunction as
 * `\|\|` — which does not end the cell.
 */
function tableRowCells(line: string): string[] {
  return line
    .replaceAll('\\|', '\u0000')
    .split('|')
    .slice(1, -1)
    .map((cell) => cell.trim().replaceAll('\u0000', '\\|'))
}

function isSeparatorRow(line: string): boolean {
  return /^\|(?:\s*:?-+:?\s*\|)+$/.test(line)
}

function lineAt(source: string, offset: number): number {
  return source.slice(0, offset).split('\n').length
}

/**
 * Reads the rows of one Markdown table, located by its header line. Returns the
 * cells of each row along with the offset of the row inside `block`.
 */
function tableRows(block: string, header: string): Array<{ cells: string[]; index: number }> {
  const start = block.indexOf(header)
  if (start < 0) return []
  const rows: Array<{ cells: string[]; index: number }> = []
  let cursor = start
  for (const line of block.slice(start).split('\n')) {
    const index = cursor
    cursor += line.length + 1
    const row = line.trim()
    if (!row.startsWith('|')) {
      if (rows.length > 0 || row.length > 0) break
      continue
    }
    if (row === header || isSeparatorRow(row)) continue
    rows.push({ cells: tableRowCells(row), index })
  }
  return rows
}

/** Table cells carry Markdown emphasis; the value is what is left without it. */
function cellValue(cell: string | undefined): string {
  return (cell ?? '').replace(/[`*_]/g, '').trim()
}

/**
 * A state machine declares its states and then its transitions. The state table
 * is what makes the set of states explicit and gives each one a meaning: derived
 * from the From and To columns alone, a state nothing transitions into vanishes.
 *
 */
function validateStateMachines(
  body: string,
  offset: number,
  source: string,
  heading: RegExp,
  findings: SpecificationFinding[],
): void {
  const machines = [...body.matchAll(heading)]
  for (const [index, machine] of machines.entries()) {
    const start = (machine.index ?? 0) + machine[0].length
    const end = machines[index + 1]?.index ?? body.length
    const block = body.slice(start, end)
    const at = (position: number) => lineAt(source, offset + start + position)
    if (!block.includes(TRANSITION_HEADER)) {
      findings.push({
        line: lineAt(source, offset + (machine.index ?? 0)),
        message: 'state transition must use From | Event | Guard | To | Effects',
      })
    }
    for (const match of block.matchAll(/^\|[^|\n]*\|[^|\n]*\|\s*""\s*\|/gm)) {
      findings.push({
        line: at(match.index ?? 0),
        message: 'unconditional state transition guard must use — instead of an empty string',
      })
    }
    if (!block.includes(STATE_HEADER)) {
      findings.push({
        line: lineAt(source, offset + (machine.index ?? 0)),
        message: `state machine must declare its states with ${STATE_HEADER}`,
      })
      continue
    }
    const states = new Set<string>()
    let initial = 0
    for (const row of tableRows(block, STATE_HEADER)) {
      const name = cellValue(row.cells[0])
      const kind = cellValue(row.cells[1])
      if (!name) continue
      if (states.has(name)) {
        findings.push({ line: at(row.index), message: `duplicate state ${name}` })
      }
      states.add(name)
      if (!STATE_KINDS.has(kind)) {
        findings.push({
          line: at(row.index),
          message: `state ${name} has Kind "${kind}"; use one of ${[...STATE_KINDS].join(', ')}`,
        })
      }
      if (kind === 'initial') initial += 1
    }
    if (initial !== 1) {
      findings.push({
        line: lineAt(source, offset + (machine.index ?? 0)),
        message: `state machine must declare exactly one initial state, found ${initial}`,
      })
    }
    for (const row of tableRows(block, TRANSITION_HEADER)) {
      for (const column of [0, 3]) {
        const name = cellValue(row.cells[column])
        if (name && !states.has(name)) {
          findings.push({
            line: at(row.index),
            message: `transition names ${name}, which the state table does not declare`,
          })
        }
      }
    }
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
  heading: RegExp,
  findings: SpecificationFinding[],
): SpecificationValidation['standardIds'] {
  const standards = [...body.matchAll(heading)]
  const seenIds = new Map<string, number>()
  const ids: SpecificationValidation['standardIds'] = []
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
      ids.push({ id, line: at })
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
  return ids
}

/** Every canonical document names itself once, whatever kind it is. */
function validateShared(source: string, findings: SpecificationFinding[]): void {
  const titles = [...source.matchAll(/^# (?!#).+$/gm)]
  if (titles.length !== 1)
    findings.push({ line: 1, message: 'document must contain exactly one H1' })
  for (const match of source.matchAll(/\[[^\]]+\]\([^\n)]*decisions\/[^\n)]*\)/g)) {
    findings.push({
      line: lineAt(source, match.index ?? 0),
      message: 'current specification must be self-contained and must not link to decisions/',
    })
  }
}

/**
 * Validate one canonical document. The file name says which grammar applies,
 * so each file is checked against that grammar alone.
 */
export function validateDocument(path: string, source: string): SpecificationValidation {
  const kind = documentKind(path)
  if (kind === undefined) {
    return {
      findings: [{ line: 1, message: 'not a canonical specification document' }],
      scenarioIds: [],
      exampleIds: [],
      standardIds: [],
      ruleReferences: [],
    }
  }

  const findings: SpecificationFinding[] = []
  validateShared(source, findings)

  const standardIds =
    kind === 'standards' ? validateStandards(source, 0, source, /^## .+$/gm, findings) : []
  if (kind === 'states') validateStateMachines(source, 0, source, /^## .+$/gm, findings)
  if (kind === 'decision-records') validateDecisionRecords(source, findings)

  let scenarioIds: SpecificationValidation['scenarioIds'] = []
  let exampleIds: SpecificationValidation['exampleIds'] = []
  let ruleReferences: SpecificationValidation['ruleReferences'] = []
  if (kind === 'specification') {
    findings.push(...validateSpecificationDeclarations(source))
    scenarioIds = specificationRules(source).map((rule) => ({
      id: rule.id,
      line: rule.line,
      supersededBy: rule.supersededBy,
      title: rule.title,
    }))
    const transitions = source.match(/^## 状態遷移[ \t]*$/m)
    if (transitions?.index !== undefined) {
      const start = transitions.index + transitions[0].length
      const rest = source.slice(start)
      const end = rest.match(/^## /m)?.index ?? rest.length
      validateStateMachines(rest.slice(0, end), start, source, /^### .+$/gm, findings)
    }
  } else if (kind === 'examples') {
    const parsed = parseScenarioDocument(source)
    findings.push(...parsed.findings)
    for (const rule of parsed.rules) {
      if (rule.supersededBy) {
        findings.push({
          line: rule.line,
          message: `${rule.id} is superseded and must leave the examples appendix`,
        })
      }
    }
    ruleReferences = parsed.rules.map((rule) => ({
      id: rule.id,
      line: rule.line,
      title: rule.name.replace(new RegExp(`^${rule.id}(?::)?\\s*`), ''),
    }))
    exampleIds = parsed.rules.flatMap((rule) =>
      rule.examples.map((example) => ({ id: example.id, line: example.line, parentId: rule.id })),
    )
  } else if (kind === 'scenarios') {
    const parsed = parseScenarioDocument(source)
    findings.push(...parsed.findings)
    scenarioIds = parsed.rules.map((rule) => ({
      id: rule.id,
      line: rule.line,
      supersededBy: rule.supersededBy,
    }))
    exampleIds = parsed.rules.flatMap((rule) =>
      rule.examples.map((example) => ({
        id: example.id,
        line: example.line,
        parentId: rule.id,
      })),
    )
    const local = new Set<string>()
    for (const scenario of scenarioIds) {
      if (local.has(scenario.id)) {
        findings.push({ line: scenario.line, message: `duplicate scenario id ${scenario.id}` })
      }
      local.add(scenario.id)
    }
  } else {
    for (const match of source.matchAll(/^#{2,6} (?:Rule: )?(REQ-[A-Z0-9-]+)(?:\s+|$)/gm)) {
      findings.push({
        line: lineAt(source, match.index ?? 0),
        message: `${match[1]} must be declared in scenarios.feature.md or in a feature specification`,
      })
    }
  }

  return { findings, scenarioIds, exampleIds, standardIds, ruleReferences }
}

/**
 * 判断の記録の各部。背景と代替案のない判断は規則の言い換えと区別できず、再検討の条件の
 * ない判断は、いつ見直すべきかを読み手に残さない。
 */
const DECISION_RECORD_SECTIONS = [
  '背景',
  '決定',
  '検討した代替案',
  '結果と再検討の条件',
  '関連する規則',
] as const

/** `design/decisions.md` の各判断（H2）が、記録の各部（H3）をこの順で一度ずつ持つことを確かめる。 */
function validateDecisionRecords(source: string, findings: SpecificationFinding[]): void {
  const decisions = [...source.matchAll(/^## (?!#)(.+)$/gm)]
  for (const [index, decision] of decisions.entries()) {
    const title = decision[1]?.trim() ?? ''
    const start = (decision.index ?? 0) + decision[0].length
    const end = decisions[index + 1]?.index ?? source.length
    const sections = [...source.slice(start, end).matchAll(/^### (.+)$/gm)].map(
      (match) => match[1]?.trim() ?? '',
    )
    const line = lineAt(source, decision.index ?? 0)
    for (const expected of DECISION_RECORD_SECTIONS) {
      if (!sections.includes(expected)) {
        findings.push({ line, message: `${title} must have the section ${expected}` })
      }
    }
    const known = sections.filter((section) =>
      (DECISION_RECORD_SECTIONS as readonly string[]).includes(section),
    )
    const ordered = DECISION_RECORD_SECTIONS.filter((section) => known.includes(section))
    if (known.join('\n') !== ordered.join('\n')) {
      findings.push({
        line,
        message: `${title} must order its sections as ${DECISION_RECORD_SECTIONS.join(', ')}`,
      })
    }
  }
}

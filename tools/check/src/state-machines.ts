import MarkdownIt from 'markdown-it'

type State = { name: string; kind: string; meaning: string }
type Transition = { from: string; event: string; guard: string; to: string; effects: string }
type Outcome =
  | { kind: 'transition'; to: string; condition?: string }
  | { kind: 'noop'; condition?: string }
  | { kind: 'refusal'; response: string; condition?: string }
type StateMachine = {
  name: string
  states: State[]
  transitions: Transition[]
  operations: string[]
  matrix: Record<string, Record<string, Outcome[]>>
}

const markdown = new MarkdownIt({ html: true })
const START = /^<!-- spec:state-machine (.+) -->$/
const END = '<!-- /spec:state-machine -->'
const SOURCE_PATH = /^backend\/[A-Za-z0-9_./-]+\/testdata\/[A-Za-z0-9]+\.statemachine\.json$/
const REGENERATE = 'run mise run generate-spec-from-testdata'

function object(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/** 表の一つのセルに入る、改行のない文字列だけを受け付ける。 */
function text(value: unknown, field: string, allowEmpty = false): string {
  if (typeof value !== 'string' || /[\r\n]/.test(value) || (!allowEmpty && !value.trim()))
    throw new Error(`${field} must be ${allowEmpty ? 'a' : 'a nonempty'} one-line string`)
  return value
}

function fields(value: unknown, allowed: readonly string[], what: string): Record<string, unknown> {
  if (!object(value)) throw new Error(`${what} must be an object`)
  for (const key of Object.keys(value))
    if (!allowed.includes(key)) throw new Error(`unknown ${what} field ${key}`)
  return value
}

function list(value: unknown, what: string): unknown[] {
  if (!Array.isArray(value)) throw new Error(`${what} must be an array`)
  return value
}

function outcome(value: unknown, at: string): Outcome {
  const item = fields(value, ['kind', 'to', 'response', 'condition'], 'state matrix outcome')
  const condition =
    item.condition === undefined ? {} : { condition: text(item.condition, `${at} condition`) }
  if (item.kind === 'transition' && item.response === undefined)
    return { kind: 'transition', to: text(item.to, `${at} to`), ...condition }
  if (item.kind === 'noop' && item.to === undefined && item.response === undefined)
    return { kind: 'noop', ...condition }
  if (item.kind === 'refusal' && item.to === undefined)
    return { kind: 'refusal', response: text(item.response, `${at} response`), ...condition }
  throw new Error(`state matrix outcome for ${at} must be transition, noop, or refusal`)
}

/**
 * 一次情報の形だけを確かめる。初期状態の数、遷移とマトリクスの一致などの意味の規則は、生成した表を
 * 読む `check-spec` の文書の検査が、手書きだった頃と同じ規則で確かめる。
 */
function parseStateMachine(value: unknown): StateMachine {
  const machine = fields(
    value,
    ['name', 'states', 'transitions', 'operations', 'matrix'],
    'state machine',
  )
  const states = list(machine.states, 'states').map((item) => {
    const state = fields(item, ['name', 'kind', 'meaning'], 'state')
    return {
      name: text(state.name, 'state name'),
      kind: text(state.kind, 'state kind'),
      meaning: text(state.meaning, 'state meaning'),
    }
  })
  if (states.length === 0) throw new Error('states must not be empty')
  const transitions = list(machine.transitions, 'transitions').map((item) => {
    const row = fields(item, ['from', 'event', 'guard', 'to', 'effects'], 'transition')
    return {
      from: text(row.from, 'transition from'),
      event: text(row.event, 'transition event'),
      guard: text(row.guard, 'transition guard'),
      to: text(row.to, 'transition to'),
      effects: text(row.effects, 'transition effects', true),
    }
  })
  const operations = list(machine.operations, 'operations').map((item) => text(item, 'operation'))
  if (operations.length === 0) throw new Error('operations must not be empty')
  if (!object(machine.matrix)) throw new Error('matrix must be an object')
  const stateNames = new Set(states.map((state) => state.name))
  for (const name of Object.keys(machine.matrix))
    if (!stateNames.has(name))
      throw new Error(`state matrix names ${name}, which the state table does not declare`)
  for (const { name } of states)
    if (machine.matrix[name] === undefined)
      throw new Error(`state matrix has no row for state ${name}`)
  // マトリクスの行は、状態の表と違う順序で読みやすく並べることがあるので、一次情報の順序を保つ。
  const matrix: StateMachine['matrix'] = {}
  for (const [name, row] of Object.entries(machine.matrix)) {
    const cells = fields(row, operations, `state matrix row ${name}`)
    matrix[name] = {}
    for (const operation of operations) {
      const at = `${name} × ${operation}`
      const outcomes = cells[operation] === undefined ? [] : list(cells[operation], at)
      if (outcomes.length === 0) throw new Error(`state matrix gives no outcome for ${at}`)
      matrix[name][operation] = outcomes.map((item) => outcome(item, at))
    }
  }
  return { name: text(machine.name, 'name'), states, transitions, operations, matrix }
}

/** ガードの CEL の `||` などがセルの区切りにならないよう、`|` を `\|` へ戻す。 */
function row(cells: readonly string[]): string {
  return `| ${cells.map((cell) => cell.replaceAll('|', '\\|')).join(' | ')} |`
}

function table(header: readonly string[], rows: readonly (readonly string[])[]): string[] {
  return [row(header), row(header.map(() => '---')), ...rows.map(row)]
}

function cell(outcomes: readonly Outcome[]): string {
  return outcomes
    .map((item) => {
      const condition = item.condition === undefined ? '' : `（${item.condition}）`
      if (item.kind === 'transition') return `→ ${item.to}${condition}`
      if (item.kind === 'noop') return `何もしない${condition}`
      return `拒否：${item.response}${condition}`
    })
    .join('<br>')
}

/** 機能仕様の状態遷移の節の三つの表を、`### <heading>` の状態機械の一次情報から生成する。 */
export function renderStateMachine(value: unknown, heading: string): string {
  const machine = parseStateMachine(value)
  if (machine.name !== heading)
    throw new Error(`state machine source names ${machine.name} under ### ${heading}`)
  return [
    ...table(
      ['State', 'Kind', 'Meaning'],
      machine.states.map((state) => [state.name, state.kind, state.meaning]),
    ),
    '',
    ...table(
      ['From', 'Event', 'Guard', 'To', 'Effects'],
      machine.transitions.map((item) => [item.from, item.event, item.guard, item.to, item.effects]),
    ),
    '',
    ...table(
      ['State', ...machine.operations],
      Object.entries(machine.matrix).map(([state, cells]) => [
        state,
        ...machine.operations.map((operation) => cell(cells[operation] ?? [])),
      ]),
    ),
  ].join('\n')
}

/** コード例を区切りと誤認しないよう、Markdown のコメント token の行だけを区切りとして読む。 */
function commentLines(source: string): Set<number> {
  return new Set(
    markdown
      .parse(source, {})
      .filter((token) => token.type === 'html_block')
      .flatMap((token) => {
        const [start, end] = token.map ?? [0, 0]
        return Array.from({ length: end - start }, (_, index) => start + index)
      }),
  )
}

export function updateStateMachineBlocks(
  source: string,
  read: (path: string) => string | undefined,
): string {
  const lines = source.split('\n')
  const comments = commentLines(source)
  const output: string[] = []
  let heading = ''
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index] ?? ''
    heading = line.match(/^### (.+)$/)?.[1]?.trim() ?? heading
    const match = comments.has(index) ? line.match(START) : undefined
    if (!match) {
      if (comments.has(index) && line === END)
        throw new Error('state machine block has no opening marker')
      if (comments.has(index) && /^<!--\s*\/?spec:state-machine/.test(line))
        throw new Error('invalid state machine marker')
      output.push(line)
      continue
    }
    const path = match[1] ?? ''
    if (!SOURCE_PATH.test(path) || path.split('/').some((part) => part === '..' || part === '.'))
      throw new Error(`invalid state machine source path: ${path}`)
    const begin = index
    while (++index < lines.length && !(comments.has(index) && lines[index] === END)) {
      if (comments.has(index) && START.test(lines[index] ?? ''))
        throw new Error('state machine blocks cannot be nested')
    }
    if (index === lines.length)
      throw new Error(`state machine block at line ${begin + 1} is not closed`)
    const data = read(path)
    if (data === undefined) throw new Error(`state machine source does not exist: ${path}`)
    output.push(line, '', renderStateMachine(JSON.parse(data), heading), '', END)
  }
  return output.join('\n')
}

/**
 * 状態遷移の節のすべての状態機械が一次情報から生成され、一次情報とずれていないことを確かめる。
 * 手書きの表はテストが読めない一次情報になるので、生成の区切りのない状態機械を拒否する。
 */
export function stateMachineFindings(
  source: string,
  read: (path: string) => string | undefined,
): string[] {
  const findings: string[] = []
  const section = source.match(/^## 状態遷移[ \t]*$/m)
  if (section?.index !== undefined) {
    const rest = source.slice(section.index + section[0].length)
    const body = rest.slice(0, rest.match(/^## /m)?.index ?? rest.length)
    const machines = [...body.matchAll(/^### (.+)$/gm)]
    for (const [index, machine] of machines.entries()) {
      const block = body.slice(machine.index, machines[index + 1]?.index ?? body.length)
      if (!/^<!-- spec:state-machine /m.test(block))
        findings.push(
          `state machine ${machine[1]?.trim()} is not generated; move its tables to testdata and ${REGENERATE}`,
        )
    }
  }
  try {
    if (updateStateMachineBlocks(source, read) !== source)
      findings.push(`state machine tables drift from their testdata; ${REGENERATE}`)
  } catch (error) {
    findings.push(error instanceof Error ? error.message : String(error))
  }
  return findings
}

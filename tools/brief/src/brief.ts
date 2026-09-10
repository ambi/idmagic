/**
 * Turn one work item's normative references into the reading list it starts from.
 *
 * `initial_context` is what keeps an agent from reading 13,000 lines of
 * canonical documentation for a change that touches one rule — but it is
 * written when the work starts, so before that point every first pass is made
 * by hand. What that pass needs already exists: the canonical documents
 * declare the rules, TypeSpec declares the symbols, and the traceability walk
 * knows every file that names an identifier. This assembles those into the
 * block a work item can take as written.
 *
 * The output is deliberately locations and one declaration each, never whole
 * files. Handing back a document would reintroduce exactly the cost the
 * reading list exists to avoid.
 */

import { isTestPath } from '../../render-spec-docs/src/traces.ts'

export type SourcePartition = {
  implementation: string[]
  tests: string[]
}

export function partitionSources(paths: readonly string[]): SourcePartition {
  const implementation: string[] = []
  const tests: string[] = []
  for (const path of paths) (isTestPath(path) ? tests : implementation).push(path)
  return { implementation, tests }
}

/**
 * The block of a canonical document that declares one normative identifier.
 *
 * A scenario is declared by a `## Rule:` heading and runs to the next heading
 * of the same level. A standards requirement is declared by a table row, which
 * means nothing on its own, so it comes back with the section heading and the
 * column names that say what its cells are.
 */
export function extractDeclaration(source: string, id: string): string | undefined {
  const lines = source.split('\n')

  const ruleStart = lines.findIndex((line) => new RegExp(`^##\\s+Rule:\\s+${id}\\b`).test(line))
  if (ruleStart >= 0) {
    let end = lines.length
    for (let index = ruleStart + 1; index < lines.length; index++) {
      if (/^##\s/.test(lines[index] ?? '')) {
        end = index
        break
      }
    }
    return `${lines.slice(ruleStart, end).join('\n').trimEnd()}\n`
  }

  const rowIndex = lines.findIndex((line) => new RegExp(`^\\|\\s*${id}\\s*\\|`).test(line))
  if (rowIndex < 0) return undefined
  let heading = ''
  let header = ''
  for (let index = rowIndex - 1; index >= 0; index--) {
    const line = lines[index] ?? ''
    if (header === '' && /^\|\s*ID\s*\|/.test(line)) header = line
    if (/^##\s/.test(line)) {
      heading = line
      break
    }
  }
  return [heading, '', header, lines[rowIndex] ?? '']
    .filter((line, index) => line !== '' || index === 1)
    .join('\n')
    .concat('\n')
}

/**
 * 識別子について、着手前に判断を変える 3 つの答え。
 *
 * `declared` は宣言が解決できたか、残りの 2 つはその識別子を名指すファイルが
 * あるかである。3 つとも「存在するか」であって、内容の一致ではない。
 */
export type Coverage = {
  declared: boolean
  implementation: boolean
  tests: boolean
}

export function coverageOf(declaration: string | undefined, naming: readonly string[]): Coverage {
  const partition = partitionSources(naming)
  return {
    declared: declaration !== undefined,
    implementation: partition.implementation.length > 0,
    tests: partition.tests.length > 0,
  }
}

/**
 * 判定が読み違えられないための一言。
 *
 * 索引は宣言された名前を literal に含むファイルを集めるので、コメントで触れて
 * いるだけのファイルも「名指している」に入る。到達可能性の保証ではない。
 */
export const COVERAGE_NOTE =
  'Coverage answers whether an identifier is declared and whether anything outside `docs/` and `spec/` names it. Naming is a literal mention of the declared name, not proof that the behavior is reachable.'

/**
 * 充足状況の 1 行。
 *
 * 前半は 3 つの答えをそのまま並べ、後半はそこから決まる着手位置を言う。
 * 着手位置を分けるのは、宣言があるかと、それを名指すファイルが 1 つでもあるかの
 * 2 点である。`REQ-*` を名指すのはテストだという慣行があるため、実装だけを見て
 * 未実装と言えば、実装済みの振る舞いを未着手として読ませることになる。
 */
export function coverageLine(coverage: Coverage): string {
  const state = [
    coverage.declared ? 'declared' : 'not declared',
    coverage.implementation ? 'named by implementation' : 'named by no implementation',
    coverage.tests ? 'named by tests' : 'named by no tests',
  ].join(', ')
  const reading = !coverage.declared
    ? 'nothing declares it yet, so this starts from a specification change'
    : coverage.implementation || coverage.tests
      ? 'something already names it, so read those files before changing either side'
      : 'the declaration is already there, so what is missing is code and not a specification edit'
  return `${state} — ${reading}`
}

/**
 * The directories the naming files sit in.
 *
 * A reading list points at packages, not at the individual file that happened
 * to cite an identifier: what has to be read is the code beside the test that
 * names the rule. Collapsing to the directory is what turns the trace result
 * into that list.
 */
export function sourceDirectories(paths: readonly string[]): string[] {
  const directories = new Set<string>()
  for (const path of paths) {
    const cut = path.lastIndexOf('/')
    directories.add(cut < 0 ? path : path.slice(0, cut))
  }
  return [...directories].sort()
}

export type InitialContext = {
  specification: readonly string[]
  typespec: readonly string[]
  implementation: readonly string[]
  tests: readonly string[]
}

/** One `initial_context` key, inline when short and a list when it is not. */
function field(name: string, values: readonly string[]): string[] {
  if (values.length === 0) return [`  ${name}: []`]
  if (values.length === 1) return [`  ${name}: [${values[0]}]`]
  return [`  ${name}:`, ...values.map((value) => `    - ${value}`)]
}

export function initialContextDraft(context: InitialContext): string {
  return [
    'initial_context:',
    ...field('specification', context.specification),
    ...field('typespec', context.typespec),
    ...field('source', context.implementation),
    ...field('tests', context.tests),
    // What to leave unread is a judgement about this change, so the draft
    // names the key and leaves the answer to whoever starts the work.
    '  stop_before_reading: []',
    '',
  ].join('\n')
}

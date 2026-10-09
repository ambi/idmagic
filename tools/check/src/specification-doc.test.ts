import { describe, expect, it } from 'bun:test'
import { documentKind, validateDocument } from './specification-doc.ts'

const SCENARIOS = 'docs/domain/demo/scenarios.feature.md'
const STATES = 'docs/domain/demo/states.md'
const STANDARDS = 'docs/domain/demo/standards.md'

const scenarios = `# Feature: Demo

## Rule: REQ-DEMO-001 A valid request succeeds

### Example: EX-DEMO-001-01 A valid request

- Given a valid request exists
- When the user submits the request
- Then the request succeeds
`

const messages = (path: string, source: string) =>
  validateDocument(path, source).findings.map((finding) => finding.message)

describe('documentKind', () => {
  it('names the grammar of each canonical document', () => {
    expect(documentKind('docs/domain/demo/states.md')).toBe('states')
    expect(documentKind('docs/domain/demo/scenarios.feature.md')).toBe('scenarios')
    expect(documentKind('docs/domain/demo/decisions.md')).toBe('prose')
    expect(documentKind('docs/domain/standards.md')).toBe('standards')
    expect(documentKind('docs/design/security/authorization.md')).toBe('prose')
    expect(documentKind('docs/design/security/threat-model.md')).toBe('prose')
    expect(documentKind('docs/design/application/design-guidelines.md')).toBe('prose')
  })

  it('names the grammar of the top-down system document tree', () => {
    expect(documentKind('docs/requirements/quality.md')).toBe('prose')
    expect(documentKind('docs/design/architecture/deployment.md')).toBe('prose')
    expect(documentKind('docs/design/security/threat-model.md')).toBe('prose')
    expect(documentKind('docs/design/observability/logging.md')).toBe('prose')
    expect(documentKind('docs/design/verification/system-acceptance.md')).toBe('prose')
    expect(documentKind('docs/operations/service-management.md')).toBe('prose')
  })

  it('rejects a name the layout does not define, and a context-only name at the root', () => {
    expect(documentKind('docs/domain/demo/notes.md')).toBeUndefined()
    expect(documentKind('docs/states.md')).toBeUndefined()
    expect(documentKind('docs/authorization.md')).toBeUndefined()
    expect(documentKind('frontend/README.md')).toBeUndefined()
    expect(documentKind('docs/design/security/network.md')).toBeUndefined()
  })

  it('reads a feature slice one level below its context with the grammar its name gives', () => {
    expect(documentKind('docs/domain/demo/user/scenarios.feature.md')).toBe('scenarios')
    expect(documentKind('docs/domain/demo/user/states.md')).toBe('states')
    expect(documentKind('docs/domain/demo/user/README.md')).toBe('specification')
    expect(documentKind('docs/domain/demo/user/internals.md')).toBe('prose')
  })

  // 基準のリビジョンには改名前の付録が残るので、spec-diff はそれも付録として読む。
  it('reads the appendix under its former name examples.feature.md', () => {
    expect(documentKind('docs/domain/demo/people/user/examples.feature.md')).toBe('examples')
  })

  it('reads the feature layout: specifications, chapters, appendix, and design', () => {
    expect(documentKind('docs/domain/demo/people/user/README.md')).toBe('specification')
    expect(documentKind('docs/domain/demo/people/user/lifecycle.md')).toBe('specification')
    expect(documentKind('docs/domain/demo/people/user/design.md')).toBe('prose')
    expect(documentKind('docs/domain/demo/people/user/acceptance.feature.md')).toBe('examples')
    expect(documentKind('docs/domain/demo/design/README.md')).toBe('design-index')
    expect(documentKind('docs/design/README.md')).toBe('design-index')
    expect(documentKind('docs/domain/demo/quality.md')).toBe('prose')
    expect(documentKind('docs/domain/demo/design/csv-transfer.md')).toBe('prose')
    expect(documentKind('docs/domain/demo/design/decisions.md')).toBe('decision-records')
    expect(documentKind('docs/domain/demo/design/csv/notes.md')).toBeUndefined()
  })

  it('keeps shared vocabulary and adopted standards at the context, not in a feature slice', () => {
    expect(documentKind('docs/domain/demo/user/glossary.md')).toBeUndefined()
    expect(documentKind('docs/domain/demo/user/standards.md')).toBeUndefined()
  })

  // 機能群の一段下まで機能スライスを置けるので、その下で木が止まる。
  it('stops the tree at the feature slice', () => {
    expect(documentKind('docs/domain/demo/people/user/profile/README.md')).toBeUndefined()
  })

  it('no longer recognizes the single canonical document', () => {
    expect(documentKind('docs/SPECIFICATION.md')).toBeUndefined()
    expect(documentKind('docs/domain/demo/SPECIFICATION.md')).toBeUndefined()
  })

  it('rejects a path the canonical layout does not define', () => {
    expect(messages('docs/domain/demo/notes.md', '# Notes\n')).toEqual([
      'not a canonical specification document',
    ])
  })
})

describe('scenarios.feature.md', () => {
  it('accepts a Gherkin example and reports its rule and example ids', () => {
    const result = validateDocument(SCENARIOS, scenarios)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.map((scenario) => scenario.id)).toEqual(['REQ-DEMO-001'])
    expect(result.exampleIds).toEqual([{ id: 'EX-DEMO-001-01', line: 5, parentId: 'REQ-DEMO-001' }])
  })

  it('accepts a retired rule without examples and reports its successor', () => {
    const source = `${scenarios}
## Rule: REQ-DEMO-002 An old behavior (superseded by REQ-DEMO-001)

Replaced by the valid request scenario.
`
    const result = validateDocument(SCENARIOS, source)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.at(-1)).toMatchObject({
      id: 'REQ-DEMO-002',
      supersededBy: 'REQ-DEMO-001',
    })
  })

  it('still requires examples in a rule that is not retired', () => {
    const source = `${scenarios}
## Rule: REQ-DEMO-002 An old behavior

Replaced by the valid request scenario.
`
    expect(messages(SCENARIOS, source)).toContain('REQ-DEMO-002 must contain at least one example')
  })

  it('rejects an example without a trigger', () => {
    const source = scenarios.replace(
      '- When the user submits the request',
      '- Given the user submits the request',
    )
    expect(messages(SCENARIOS, source)).toContain('example must contain at least one When step')
  })

  it('accepts multiple triggers in a multi-operation flow', () => {
    const source = scenarios.replace(
      '- Then the request succeeds',
      '- Then the request succeeds\n- When the user retrieves the result\n- Then the result is returned',
    )
    expect(validateDocument(SCENARIOS, source).findings).toEqual([])
  })

  it('rejects links from a canonical document to decisions/', () => {
    const source = scenarios.replace(
      '# Feature: Demo',
      '# Feature: Demo\n\nSee [old choice](../decisions/old-choice.md).',
    )
    expect(messages(SCENARIOS, source)).toContain(
      'current specification must be self-contained and must not link to decisions/',
    )
  })

  it('requires exactly one H1', () => {
    expect(messages(SCENARIOS, scenarios.replace('# Feature: Demo\n', ''))).toContain(
      'document must contain exactly one H1',
    )
  })

  it('形式文書のコードブロックを見出しや規範宣言として数えない', () => {
    const source = [
      '# 仕様フォーマット',
      '',
      '````markdown',
      '# 機能仕様のテンプレート',
      '```text',
      '#### REQ-DEMO-001 コード内の例',
      '```',
      '````',
      '',
      '~~~markdown',
      '# 作業項目のテンプレート',
      '#### REQ-DEMO-002 別の例',
      '~~~',
      '',
    ].join('\n')
    const path = 'docs/formats/specification-format.md'

    expect(validateDocument(path, source).findings).toEqual([])
    expect(validateDocument(path, source).scenarioIds).toEqual([])
    expect(validateDocument(path, `${source}#### REQ-DEMO-003 本文の宣言\n`).findings).toEqual([
      {
        line: 14,
        message:
          'REQ-DEMO-003 must be declared in scenarios.feature.md or in a feature specification',
      },
    ])
    expect(messages(path, source.replace('# 仕様フォーマット', '見出しなし'))).toContain(
      'document must contain exactly one H1',
    )
  })

  it('keeps normative rules out of the other documents', () => {
    const source = `# Demo Decisions

## Rule: REQ-DEMO-002 A behavior
`
    const result = validateDocument('docs/domain/demo/decisions.md', source)
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'REQ-DEMO-002 must be declared in scenarios.feature.md or in a feature specification',
    ])
    expect(result.scenarioIds).toEqual([])
  })
})

describe('feature specification', () => {
  const path = 'docs/domain/demo/work/task/README.md'

  it('declares rules with headings and reads their titles and supersession', () => {
    const result = validateDocument(
      path,
      [
        '# Task',
        '',
        '## 操作',
        '',
        '#### REQ-DEMO-002 開いたタスクだけを一覧する',
        '',
        '#### REQ-DEMO-001 旧い一覧 (superseded by REQ-DEMO-002)',
        '',
      ].join('\n'),
    )
    expect(result.findings).toEqual([])
    expect(result.scenarioIds).toEqual([
      { id: 'REQ-DEMO-002', line: 5, supersededBy: undefined, title: '開いたタスクだけを一覧する' },
      {
        id: 'REQ-DEMO-001',
        line: 7,
        supersededBy: 'REQ-DEMO-002',
        title: '旧い一覧 (superseded by REQ-DEMO-002)',
      },
    ])
  })

  it('rejects the Gherkin keyword and declarations at the wrong heading level', () => {
    const result = validateDocument(
      path,
      '# Task\n\n### Rule: REQ-DEMO-002 一覧\n\n## REQ-DEMO-003 数える\n',
    )
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'REQ-DEMO-002 must be declared as a "#### REQ-DEMO-002 <title>" heading, without "Rule:"',
      'REQ-DEMO-003 must be declared at heading level 3 or 4',
    ])
  })

  it('checks the state machines under its state transition section', () => {
    const result = validateDocument(
      path,
      [
        '# Task',
        '',
        '## 状態遷移',
        '',
        '### TaskLifecycle',
        '',
        '| From | Event | Guard | To | Effects |',
        '|---|---|---|---|---|',
        '| open | TaskClosed | — | closed |  |',
        '',
      ].join('\n'),
    )
    expect(result.findings.map((finding) => finding.message)).toContain(
      'state machine must declare its states with | State | Kind | Meaning |',
    )
  })
})

describe('examples appendix', () => {
  it('reads rule references and examples without declaring the rules', () => {
    const result = validateDocument(
      'docs/domain/demo/work/task/acceptance.feature.md',
      [
        '# Feature: タスクの例',
        '',
        '## Rule: REQ-DEMO-002 開いたタスクだけを一覧する',
        '',
        '### Example: EX-DEMO-002-01 開いたタスク',
        '',
        '- When 一覧を要求する',
        '- Then 開いたタスクを返す',
      ].join('\n'),
    )
    expect(result.findings).toEqual([])
    expect(result.scenarioIds).toEqual([])
    expect(result.ruleReferences).toEqual([
      { id: 'REQ-DEMO-002', line: 3, title: '開いたタスクだけを一覧する' },
    ])
    expect(result.exampleIds).toEqual([{ id: 'EX-DEMO-002-01', line: 5, parentId: 'REQ-DEMO-002' }])
  })
})

const states = `# Demo State Transitions

## Lifecycle

| State | Kind | Meaning |
|---|---|---|
| Ready | initial | 受理直後 |
| Done | terminal | 完了 |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Ready | Run | — | Done | Completed |

| State | 実行 |
|---|---|
| Ready | → Done |
| Done | 何もしない |
`

describe('states.md', () => {
  it('accepts a machine that declares its states before its transitions', () => {
    expect(validateDocument(STATES, states).findings).toEqual([])
  })

  it('requires the state table', () => {
    const source = states.replace(
      '| State | Kind | Meaning |\n|---|---|---|\n| Ready | initial | 受理直後 |\n| Done | terminal | 完了 |\n\n',
      '',
    )
    expect(messages(STATES, source)).toEqual([
      'state machine must declare its states with | State | Kind | Meaning |',
    ])
  })

  it('requires the transition table', () => {
    const source = states.slice(0, states.indexOf('| From | Event |'))
    expect(messages(STATES, source)).toContain(
      'state transition must use From | Event | Guard | To | Effects',
    )
  })

  it('rejects a transition into a state the state table does not declare', () => {
    const source = states.replace('| Ready | Run | — | Done |', '| Ready | Run | — | Gone |')
    expect(messages(STATES, source)).toEqual([
      'transition names Gone, which the state table does not declare',
      'state matrix moves Ready to Done, which the transition table does not list',
      'transition Ready to Gone has no operation in the state matrix',
    ])
  })

  it('rejects a Kind outside the vocabulary and a machine without one initial state', () => {
    const source = states.replace('| Ready | initial |', '| Ready | 初期 |')
    expect(messages(STATES, source)).toEqual([
      'state Ready has Kind "初期"; use one of initial, terminal, —',
      'state machine must declare exactly one initial state, found 0',
    ])
  })

  it('rejects an empty-string guard', () => {
    const source = states.replace('| Ready | Run | — |', '| Ready | Run | "" |')
    expect(messages(STATES, source)).toContain(
      'unconditional state transition guard must use — instead of an empty string',
    )
  })

  it('keeps an escaped pipe inside a guard rather than ending the cell', () => {
    const source = states.replace(
      '| Ready | Run | — | Done | Completed |',
      '| Ready | Run | input.purge \\|\\| expired | Done | Completed |',
    )
    expect(validateDocument(STATES, source).findings).toEqual([])
  })
})

const matrixMachine = (matrix: string) => `# Demo State Transitions

## Lifecycle

| State | Kind | Meaning |
|---|---|---|
| Ready | initial | 受理直後 |
| Done | terminal | 完了 |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Ready | Run | — | Done | Completed |

${matrix}
`

const completeMatrix = [
  '| State | 実行 | 取り消し |',
  '|---|---|---|',
  '| Ready | → Done | 何もしない（期限内）<br>拒否：409 expired（期限後） |',
  '| Done | 何もしない | 拒否：404 not_found |',
].join('\n')

describe('state matrix', () => {
  it('accepts a matrix that gives every state and operation an outcome', () => {
    expect(validateDocument(STATES, matrixMachine(completeMatrix)).findings).toEqual([])
  })

  it('reads the condition after a target state apart from the state name', () => {
    const source = matrixMachine(
      completeMatrix.replace(
        '| Ready | → Done |',
        '| Ready | → Done（期限内）<br>何もしない（期限後） |',
      ),
    )
    expect(validateDocument(STATES, source).findings).toEqual([])
  })

  it('reads a target state with an underscore the same way as the state table', () => {
    const source = `# Demo State Transitions

## Lifecycle

| State | Kind | Meaning |
|---|---|---|
| pending | initial | 待ち |
| dead_letter | terminal | 配送をあきらめた |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | DeadLettered | — | dead_letter |  |

| State | 配送 |
|---|---|
| pending | → dead_letter（上限） |
| dead_letter | 何もしない |
`
    expect(validateDocument(STATES, source).findings).toEqual([])
  })

  it('rejects a machine without a matrix', () => {
    const source = states.slice(0, states.indexOf('| State | 実行 |'))
    expect(messages(STATES, source)).toEqual([
      'state machine must give its state matrix as | State | <operation> | rows',
    ])
  })

  it('rejects an empty cell and a row with fewer cells than operations', () => {
    const source = matrixMachine(
      completeMatrix.replace('| Done | 何もしない | 拒否：404 not_found |', '| Done |  |'),
    )
    expect(messages(STATES, source)).toEqual([
      'state matrix gives no outcome for Done × 実行',
      'state matrix gives no outcome for Done × 取り消し',
    ])
  })

  it('rejects an outcome outside the vocabulary', () => {
    const source = matrixMachine(completeMatrix.replace('| Done | 何もしない |', '| Done | 未定 |'))
    expect(messages(STATES, source)).toEqual([
      'state matrix outcome "未定" for Done × 実行 must be → <State>, 何もしない, or 拒否：<response>',
    ])
  })

  it('rejects a row or a target that the state table does not declare', () => {
    const source = matrixMachine(
      `${completeMatrix}\n| Gone | → Ready | 何もしない |`.replace(
        '| Done | 何もしない |',
        '| Done | → Lost |',
      ),
    )
    expect(messages(STATES, source)).toEqual([
      'state matrix names Lost, which the state table does not declare',
      'state matrix names Gone, which the state table does not declare',
    ])
  })

  it('rejects a declared state the matrix has no row for', () => {
    const source = matrixMachine(
      completeMatrix.replace('\n| Done | 何もしない | 拒否：404 not_found |', ''),
    )
    expect(messages(STATES, source)).toEqual(['state matrix has no row for state Done'])
  })

  it('rejects a matrix transition that the transition table does not list', () => {
    const source = matrixMachine(
      completeMatrix.replace('| Done | 何もしない |', '| Done | → Ready |'),
    )
    expect(messages(STATES, source)).toEqual([
      'state matrix moves Done to Ready, which the transition table does not list',
    ])
  })

  it('rejects a listed transition that no matrix cell reaches', () => {
    const source = matrixMachine(
      completeMatrix.replace('| Ready | → Done |', '| Ready | 何もしない |'),
    )
    expect(messages(STATES, source)).toEqual([
      'transition Ready to Done has no operation in the state matrix',
    ])
  })
})

const withStandards = (rows: string) => `# Demo Standards

## Demo Protocol

https://example.invalid/demo

| ID | Adoption | Strength | Statement |
|---|---|---|---|
${rows}
`

describe('standards.md', () => {
  it('accepts closed vocabularies on both axes, including optional with MUST', () => {
    const source = withStandards(
      [
        '| DEMO-CORE | required | MUST | The product answers a demo request. |',
        '| DEMO-EXTRA | optional | MUST | When the extension is offered, its rules are honored. |',
        '| DEMO-LEGACY | excluded | MAY | The legacy transport is not offered. |',
      ].join('\n'),
    )
    expect(validateDocument(STANDARDS, source).findings).toEqual([])
  })

  it('rejects a standard without the canonical table', () => {
    const source = '# Demo Standards\n\n## Demo Protocol\n\nAdopted in full.\n'
    expect(messages(STANDARDS, source)).toContain(
      'standard must use | ID | Adoption | Strength | Statement |',
    )
  })

  it('rejects an adoption outside the vocabulary', () => {
    const source = withStandards('| DEMO-CORE | planned | MUST | Someday. |')
    expect(messages(STANDARDS, source)).toContain(
      'DEMO-CORE has Adoption "planned"; use one of required, optional, partial, excluded',
    )
  })

  it('rejects a strength outside the vocabulary', () => {
    const source = withStandards('| DEMO-CORE | required | SHALL | The product answers. |')
    expect(messages(STANDARDS, source)).toContain(
      'DEMO-CORE has Strength "SHALL"; use one of MUST, MUST NOT, SHOULD, MAY',
    )
  })

  it('rejects an obligation on an excluded capability', () => {
    const source = withStandards(
      '| DEMO-LEGACY | excluded | MUST | The legacy transport is required. |',
    )
    expect(messages(STANDARDS, source)).toContain(
      'DEMO-LEGACY is excluded, so it cannot carry the obligation "MUST"',
    )
  })

  it('rejects a duplicate standard id across two standards', () => {
    const source = `${withStandards('| DEMO-CORE | required | MUST | The product answers. |')}
## Other Protocol

https://example.invalid/other

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| DEMO-CORE | optional | MAY | A second row claiming the same id. |
`
    expect(messages(STANDARDS, source)).toContain(
      'duplicate standard id DEMO-CORE (first seen on line 9)',
    )
  })

  it('collects the id of every row so the coverage check can ask for a test', () => {
    const source = withStandards(
      [
        '| DEMO-CORE | required | MUST | The product answers a demo request. |',
        '| DEMO-LEGACY | excluded | MAY | The legacy transport is not offered. |',
      ].join('\n'),
    )
    expect(validateDocument(STANDARDS, source).standardIds).toEqual([
      { id: 'DEMO-CORE', line: 9 },
      { id: 'DEMO-LEGACY', line: 10 },
    ])
  })

  it('collects no standard id from a document of another kind', () => {
    expect(validateDocument(SCENARIOS, scenarios).standardIds).toEqual([])
  })
})

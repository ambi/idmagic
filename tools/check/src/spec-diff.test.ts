import { describe, expect, it } from 'bun:test'
import { type Snapshot, diffSpecifications, formatSpecificationDiff } from './spec-diff.ts'

const document = (scenarios: string, transitions = ''): string =>
  [
    '# Demo Specification',
    '',
    '## Overview',
    '',
    'Demo behavior.',
    '',
    '## State Transitions',
    '',
    transitions,
    '',
    '## Scenarios',
    '',
    scenarios,
  ].join('\n')

const machine = (effect: string): string =>
  [
    '### Lifecycle',
    '',
    '| From | Event | Guard | To | Effects |',
    '|---|---|---|---|---|',
    `| Ready | Run | — | Done | ${effect} |`,
  ].join('\n')

const scenario = (id: string, result: string): string =>
  [
    `### ${id}: A request`,
    '- ACTOR User',
    '- WHEN the request is submitted',
    `- THEN ${result}`,
  ].join('\n')

const gherkinScenario = (id: string, result: string): string =>
  [
    '# Feature: Demo Scenarios',
    '',
    `## Rule: ${id} A request`,
    '',
    `### Example: ${id.replace('REQ-', 'EX-')}-01 request succeeds`,
    '',
    '- When the request is submitted',
    `- Then ${result}`,
  ].join('\n')

const snapshot = (documentSource: string, tsp = 'op StartTask(): void;'): Snapshot =>
  new Map([
    ['docs/domain/demo/SPECIFICATION.md', documentSource],
    ['spec/contexts/demo/main.tsp', tsp],
  ])

describe('diffSpecifications', () => {
  it('DocString の本文だけの変更を検出する', () => {
    const document = (content: string): Snapshot =>
      new Map([
        [
          'docs/domain/demo/scenarios.feature.md',
          `${gherkinScenario('REQ-DEMO-001', '次の本文になる')}\n\n  \`\`\`json\n  ${content}\n  \`\`\`\n`,
        ],
      ])
    expect(
      diffSpecifications(document('{"name":"before"}'), document('{"name":"after"}'))
        .changedScenarios,
    ).toEqual(['REQ-DEMO-001'])
    expect(
      diffSpecifications(document('{"name":"before"}'), document('{"name":"before"}'))
        .changedScenarios,
    ).toEqual([])
  })

  it('同じ手順の集合でも操作順の変更を検出する', () => {
    const document = (actions: string): Snapshot =>
      new Map([
        [
          'docs/domain/demo/scenarios.feature.md',
          gherkinScenario('REQ-DEMO-001', '結果を返す').replace(
            '- When the request is submitted',
            actions,
          ),
        ],
      ])
    expect(
      diffSpecifications(
        document('- When 最初の操作を実行する\n- And 次の操作を実行する'),
        document('- When 次の操作を実行する\n- And 最初の操作を実行する'),
      ).changedScenarios,
    ).toEqual(['REQ-DEMO-001'])
  })

  it('決定表の値だけの変更を検出し、列の掲載順は比較しない', () => {
    const path = 'docs/domain/demo/scenarios.feature.md'
    const document = (columns: string[], values: string[]): Snapshot =>
      new Map([
        [
          path,
          [
            '# Feature: Demo',
            '',
            '## Rule: REQ-DEMO-001 条件から結果を決める',
            '',
            '### Scenario Outline: 条件ごとの結果',
            '',
            '- Given 条件は <condition> である',
            '- When 利用者が要求を送る',
            '- Then 結果は <outcome> である',
            '',
            '#### Examples: Decision table (Unique)',
            '',
            `  | ${columns.join(' | ')} |`,
            `  | ${columns.map(() => '---').join(' | ')} |`,
            `  | ${values.join(' | ')} |`,
          ].join('\n'),
        ],
      ])
    const before = document(
      ['example_id', 'condition', 'outcome'],
      ['EX-DEMO-001-01', 'allowed', 'success'],
    )
    expect(
      diffSpecifications(
        before,
        document(['example_id', 'condition', 'outcome'], ['EX-DEMO-001-01', 'allowed', 'refusal']),
      ).changedScenarios,
    ).toEqual(['REQ-DEMO-001'])
    expect(
      diffSpecifications(
        before,
        document(['outcome', 'example_id', 'condition'], ['success', 'EX-DEMO-001-01', 'allowed']),
      ).changedScenarios,
    ).toEqual([])
  })

  it('reports nothing when the normative content is unchanged', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Done')))
    const diff = diffSpecifications(
      base,
      snapshot(document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Done'))),
    )
    expect(diff).toEqual({
      addedScenarios: [],
      removedScenarios: [],
      changedScenarios: [],
      changedTransitions: [],
      addedStandards: [],
      removedStandards: [],
      changedStandards: [],
      addedDeclarations: [],
      removedDeclarations: [],
      changedDeclarations: [],
      addedDeprecations: [],
      removedDeprecations: [],
    })
    expect(formatSpecificationDiff(diff, 'main')).toBe(
      'no normative specification change against main',
    )
  })

  it('separates added, removed, and changed scenarios', () => {
    const base = snapshot(
      document(
        [scenario('REQ-DEMO-001', 'it succeeds'), scenario('REQ-DEMO-002', 'it stops')].join(
          '\n\n',
        ),
      ),
    )
    const head = snapshot(
      document(
        [scenario('REQ-DEMO-001', 'it is accepted'), scenario('REQ-DEMO-003', 'it retries')].join(
          '\n\n',
        ),
      ),
    )
    const diff = diffSpecifications(base, head)
    expect(diff.addedScenarios).toEqual(['REQ-DEMO-003'])
    expect(diff.removedScenarios).toEqual(['REQ-DEMO-002'])
    expect(diff.changedScenarios).toEqual(['REQ-DEMO-001'])
  })

  it('detects a changed transition row and ignores reformatting', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Done')))
    const reformatted = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds'), `${machine('emit Done')}   `),
    )
    expect(diffSpecifications(base, reformatted).changedTransitions).toEqual([])

    const changed = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Completed')),
    )
    expect(diffSpecifications(base, changed).changedTransitions).toEqual([
      'docs/domain/demo#Lifecycle',
    ])
  })

  it('reports nothing when a context moves to the split layout', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Done')))
    const head: Snapshot = new Map([
      [
        'docs/domain/demo/scenarios.feature.md',
        `${gherkinScenario('REQ-DEMO-001', 'it succeeds')}\n`,
      ],
      [
        'docs/domain/demo/states.md',
        [
          '# Demo State Transitions',
          '',
          '## Lifecycle',
          '',
          '| State | Kind | Meaning |',
          '|---|---|---|',
          '| Ready | initial | 受理直後 |',
          '| Done | terminal | 完了 |',
          '',
          machine('emit Done').split('\n').slice(1).join('\n'),
        ].join('\n'),
      ],
      ['spec/contexts/demo/main.tsp', 'op StartTask(): void;'],
    ])
    const diff = diffSpecifications(base, head)
    expect(diff).toEqual({
      addedScenarios: [],
      removedScenarios: [],
      changedScenarios: [],
      changedTransitions: [],
      addedStandards: [],
      removedStandards: [],
      changedStandards: [],
      addedDeclarations: [],
      removedDeclarations: [],
      changedDeclarations: [],
      addedDeprecations: [],
      removedDeprecations: [],
    })
  })

  it('reports a rule whose body changed even when its title and steps did not', () => {
    const rule = (limit: string): Snapshot =>
      new Map([
        [
          'docs/domain/demo/scenarios.feature.md',
          [
            '# Feature: Demo Scenarios',
            '',
            '## Rule: REQ-DEMO-001 A request',
            '',
            '- 担保手段：`demo.Limit`',
            '',
            '| 入力 | 結果 |',
            '| --- | --- |',
            `| limit なし | ${limit} 件 |`,
            '',
            '### Example: EX-DEMO-001-01 request succeeds',
            '',
            '- When the request is submitted',
            '- Then it succeeds',
          ].join('\n'),
        ],
      ])
    expect(diffSpecifications(rule('50'), rule('50')).changedScenarios).toEqual([])
    expect(diffSpecifications(rule('50'), rule('10')).changedScenarios).toEqual(['REQ-DEMO-001'])
  })

  it('compares a link in a rule body by its label, not by its relative target', () => {
    const rule = (path: string, link: string): Snapshot =>
      new Map([
        [
          path,
          [
            '# Feature: Demo Scenarios',
            '',
            '## Rule: REQ-DEMO-001 A request',
            '',
            `| 認可 | 規則は${link}が定める |`,
            '',
            '### Example: EX-DEMO-001-01 request succeeds',
            '',
            '- When the request is submitted',
            '- Then it succeeds',
          ].join('\n'),
        ],
      ])
    const base = rule(
      'docs/domain/demo/group/task/scenarios.feature.md',
      '[管理 API の認可](../../common/access/README.md#認可)',
    )
    const moved = rule(
      'docs/domain/demo/task/scenarios.feature.md',
      '[管理 API の認可](../access/README.md#認可)',
    )
    const relabeled = rule(
      'docs/domain/demo/task/scenarios.feature.md',
      '[ロール](../access/README.md#認可)',
    )
    expect(diffSpecifications(base, moved).changedScenarios).toEqual([])
    expect(diffSpecifications(base, relabeled).changedScenarios).toEqual(['REQ-DEMO-001'])
  })

  it('reports nothing when a rule and its state machine move into a feature slice', () => {
    const lifecycle = [
      '## Lifecycle',
      '',
      '| State | Kind | Meaning |',
      '|---|---|---|',
      '| Ready | initial | 受理直後 |',
      '| Done | terminal | 完了 |',
      '',
      machine('emit Done').split('\n').slice(1).join('\n'),
    ].join('\n')
    const base: Snapshot = new Map([
      [
        'docs/domain/demo/scenarios.feature.md',
        [
          '# Feature: Demo Scenarios',
          '',
          '## Rule: REQ-DEMO-001 A request',
          '',
          '- 規則文の一行',
          '',
          '### Example: EX-DEMO-001-01 request succeeds',
          '',
          '- When the request is submitted',
          '- Then it succeeds',
        ].join('\n'),
      ],
      ['docs/domain/demo/states.md', `# Demo State Transitions\n\n${lifecycle}\n`],
    ])
    const head: Snapshot = new Map([
      ['docs/domain/demo/scenarios.feature.md', '# Feature: Demo Scenarios\n'],
      [
        'docs/domain/demo/task/scenarios.feature.md',
        [
          '# Feature: Task',
          '',
          '## 生成',
          '',
          '### Rule: REQ-DEMO-001 A request',
          '',
          '- 規則文の一行',
          '',
          '#### Example: EX-DEMO-001-01 request succeeds',
          '',
          '- When the request is submitted',
          '- Then it succeeds',
        ].join('\n'),
      ],
      ['docs/domain/demo/task/states.md', `# Task State Transitions\n\n${lifecycle}\n`],
    ])
    const diff = diffSpecifications(base, head)
    expect(diff.addedScenarios).toEqual([])
    expect(diff.removedScenarios).toEqual([])
    expect(diff.changedScenarios).toEqual([])
    expect(diff.changedTransitions).toEqual([])
  })

  it('reports nothing when a rule moves into a feature specification with an appendix', () => {
    const base: Snapshot = new Map([
      [
        'docs/domain/demo/task/scenarios.feature.md',
        [
          '# Feature: Task',
          '',
          '## 生成',
          '',
          '### Rule: REQ-DEMO-001 A request',
          '',
          '- 規則文の一行',
          '',
          '#### Example: EX-DEMO-001-01 request succeeds',
          '',
          '- When the request is submitted',
          '- Then it succeeds',
        ].join('\n'),
      ],
    ])
    const specification = (statement: string) =>
      [
        '# Task',
        '',
        '## 状態遷移',
        '',
        machine('emit Done'),
        '',
        '## 操作',
        '',
        '### Submit',
        '',
        '#### REQ-DEMO-001 A request',
        '',
        `- ${statement}`,
        '- **担保手段**：`Task.Open`',
      ].join('\n')
    const appendix = (result: string) =>
      [
        '# Feature: Task の例',
        '',
        '## Rule: REQ-DEMO-001 A request',
        '',
        '### Example: EX-DEMO-001-01 request succeeds',
        '',
        '- When the request is submitted',
        `- Then ${result}`,
      ].join('\n')
    const head = (statement: string, result: string): Snapshot =>
      new Map([
        ['docs/domain/demo/work/task/README.md', specification(statement)],
        ['docs/domain/demo/work/task/acceptance.feature.md', appendix(result)],
      ])
    const migrated = diffSpecifications(base, head('規則文の一行', 'it succeeds'))
    expect(migrated.removedScenarios).toEqual([])
    expect(migrated.changedScenarios).toEqual([])
    expect(migrated.changedTransitions).toEqual(['docs/domain/demo#Lifecycle'])

    const settled = head('規則文の一行', 'it succeeds')
    expect(
      diffSpecifications(settled, head('規則文の二行', 'it succeeds')).changedScenarios,
    ).toEqual(['REQ-DEMO-001'])
    expect(diffSpecifications(settled, head('規則文の一行', 'it fails')).changedScenarios).toEqual([
      'REQ-DEMO-001',
    ])
    expect(diffSpecifications(settled, settled).changedTransitions).toEqual([])
  })

  it('tracks TypeSpec declarations coming and going', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')), 'model Task {}')
    const head = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      'op StartTask(): void;',
    )
    const diff = diffSpecifications(base, head)
    expect(diff.addedDeclarations).toEqual(['spec/contexts/demo/main.tsp:StartTask'])
    expect(diff.removedDeclarations).toEqual(['spec/contexts/demo/main.tsp:Task'])
  })

  it('tracks TypeSpec deprecations independently from declaration existence', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')), 'model LegacyDemo {}')
    const head = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      '@deprecated("Use CurrentDemo")\nmodel LegacyDemo {}',
    )
    const diff = diffSpecifications(base, head)
    expect(diff.addedDeclarations).toEqual([])
    expect(diff.addedDeprecations).toEqual(['spec/contexts/demo/main.tsp:LegacyDemo'])
    expect(diff.removedDeprecations).toEqual([])
    expect(formatSpecificationDiff(diff, 'main')).toContain(
      'added TypeSpec deprecations:\n  spec/contexts/demo/main.tsp:LegacyDemo',
    )
  })

  it('separates added, removed, and changed standards rows by owning path and id', () => {
    const standard = (rows: string[]): string =>
      [
        '# Standards',
        '',
        '| Normative ID | Adoption | Strength | Statement |',
        '| --- | --- | --- | --- |',
        ...rows,
      ].join('\n')
    const base: Snapshot = new Map([
      [
        'docs/domain/demo/standards.md',
        standard([
          '| RFC-DEMO-ONE | required | MUST | The first behavior. |',
          '| RFC-DEMO-TWO | partial | SHOULD | The old behavior. |',
        ]),
      ],
    ])
    const head: Snapshot = new Map([
      [
        'docs/domain/demo/standards.md',
        standard([
          '| RFC-DEMO-ONE | required | MUST | The changed behavior. |',
          '| RFC-DEMO-THREE | required | MUST | The new behavior. |',
        ]),
      ],
    ])

    const diff = diffSpecifications(base, head)
    expect(diff.addedStandards).toEqual(['docs/domain/demo/standards.md#RFC-DEMO-THREE'])
    expect(diff.removedStandards).toEqual(['docs/domain/demo/standards.md#RFC-DEMO-TWO'])
    expect(diff.changedStandards).toEqual(['docs/domain/demo/standards.md#RFC-DEMO-ONE'])
    expect(formatSpecificationDiff(diff, 'main')).toContain(
      'changed standards requirements:\n  docs/domain/demo/standards.md#RFC-DEMO-ONE',
    )
  })

  it('leaves per-operation transport wrappers out of the declaration list', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')), '')
    const head = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      [
        'op StartTask(): void;',
        'model StartTaskError403Body {}',
        'model StartTaskHttpRequest {}',
        'model StartTaskSuccess_200 {}',
      ].join('\n'),
    )
    expect(diffSpecifications(base, head).addedDeclarations).toEqual([
      'spec/contexts/demo/main.tsp:StartTask',
    ])
  })

  it('does not read a declaration out of English doc prose', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')), '')
    const head = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      [
        '@doc("The union of the direct roles and the group roles. The model was renamed.")',
        'model Task {}',
      ].join('\n'),
    )
    expect(diffSpecifications(base, head).addedDeclarations).toEqual([
      'spec/contexts/demo/main.tsp:Task',
    ])
  })

  it('formats only the groups that have entries', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')))
    const head = snapshot(
      document(
        [scenario('REQ-DEMO-001', 'it succeeds'), scenario('REQ-DEMO-002', 'it retries')].join(
          '\n\n',
        ),
      ),
    )
    const text = formatSpecificationDiff(diffSpecifications(base, head), 'HEAD')
    expect(text).toContain('added scenarios:\n  REQ-DEMO-002')
    expect(text).not.toContain('removed scenarios')
  })

  it('reports an existing TypeSpec declaration whose body changed', () => {
    const base = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      'model Task { id: string; }\nop StartTask(): Task;',
    )
    const head = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      'model Task {\n  id: string;\n  name: string;\n}\n\nop StartTask():   Task;',
    )
    const diff = diffSpecifications(base, head)
    expect(diff.changedDeclarations).toEqual(['spec/contexts/demo/main.tsp:Task'])
    expect(diff.addedDeclarations).toEqual([])
    expect(formatSpecificationDiff(diff, 'main')).toContain(
      'changed TypeSpec declarations:\n  spec/contexts/demo/main.tsp:Task',
    )
  })

  it('reports a changed transport wrapper as a change to the operation that owns it', () => {
    const operation = (members: string): string =>
      ['union StartTaskError400Body {', members, '}', 'op StartTask(): void;'].join('\n')
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')), operation('Invalid,'))
    const head = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      operation('Invalid,\nConflict,'),
    )
    expect(diffSpecifications(base, head).changedDeclarations).toEqual([
      'spec/contexts/demo/main.tsp:StartTask',
    ])
  })
})

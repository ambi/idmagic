import { describe, expect, it } from 'bun:test'
import { diffSpecifications, type SpecificationDiff } from './spec-diff.ts'
import {
  type CheckedCommit,
  type CheckedWorkItem,
  type SpecReference,
  type WorkRange,
  changedCharacterizations,
  parseSpecImpactTrailer,
  referencedElement,
  specImpactDeclaration,
  verifySpecImpact,
  workRangeStart,
} from './spec-impact.ts'

const SCENARIOS = 'docs/modules/demo/scenarios.feature.md'
const STANDARDS = 'docs/modules/demo/standards.md'
const TYPESPEC = 'spec/modules/demo/main.tsp'
const ITEM = 'work-items/wi-10001-demo.md'
const CONCRETE = 'トークンを検索する順序、返すエラー、発行するイベントを変えない。'

const diff = (changes: Partial<SpecificationDiff> = {}): SpecificationDiff => ({
  ...diffSpecifications(new Map(), new Map()),
  ...changes,
})

const range = (
  changes: Partial<Omit<WorkRange, 'diffs'>> & { diff?: SpecificationDiff } = {},
): WorkRange => {
  const { diff: single, ...rest } = changes
  return {
    diffs: single ? [single] : [],
    claimedByOthers: [],
    changedCharacterizations: [],
    ...rest,
  }
}

const requirement = (id: string): SpecReference => ({
  path: id.startsWith('REQ-') || id.startsWith('EX-') ? SCENARIOS : STANDARDS,
  requirement: id,
})

const symbol = (name: string): SpecReference => ({
  path: TYPESPEC,
  symbol: `Demo.Operations.${name}`,
})

const affected = (references: SpecReference[], workRange = range()): CheckedWorkItem => ({
  path: ITEM,
  declaration: { kind: 'affected', references },
  range: workRange,
})

const none = (workRange = range(), reason = CONCRETE): CheckedWorkItem => ({
  path: ITEM,
  declaration: { kind: 'none', reason },
  range: workRange,
})

const commit = (changes: Partial<CheckedCommit> = {}): CheckedCommit => ({
  sha: '0123456789abcdef',
  subject: 'refactor(demo): extract helper',
  productionPaths: ['backend/demo/service.go'],
  trailer: undefined,
  diff: diff(),
  changedCharacterizations: [],
  ...changes,
})

const verifyItems = (...items: CheckedWorkItem[]): string[] =>
  verifySpecImpact({ items, commits: [], diffs: [] })

const verifyCommits = (...commits: CheckedCommit[]): string[] =>
  verifySpecImpact({ items: [], commits, diffs: [] })

describe('specImpactDeclaration', () => {
  it('affected_spec の参照を読み、path のない廃止済みの形式を読み飛ばす', () => {
    expect(
      specImpactDeclaration({
        affected_spec: [
          { path: SCENARIOS, requirement: 'REQ-DEMO-001' },
          { path: TYPESPEC, symbol: 'Demo.Task' },
          { scl: 'legacy' },
        ],
      }),
    ).toEqual({
      kind: 'affected',
      references: [
        { path: SCENARIOS, requirement: 'REQ-DEMO-001', symbol: undefined },
        { path: TYPESPEC, requirement: undefined, symbol: 'Demo.Task' },
      ],
    })
  })

  it('spec_impact: none を理由とともに読み、宣言がなければ undefined を返す', () => {
    expect(specImpactDeclaration({ spec_impact: { kind: 'none', reason: CONCRETE } })).toEqual({
      kind: 'none',
      reason: CONCRETE,
    })
    expect(specImpactDeclaration({ status: 'in_progress' })).toBeUndefined()
  })
})

describe('referencedElement', () => {
  it.each([
    [requirement('REQ-DEMO-001'), 'REQ-DEMO-001'],
    [requirement('EX-DEMO-001-02'), 'REQ-DEMO-001'],
    [requirement('RFC-DEMO-ONE'), `${STANDARDS}#RFC-DEMO-ONE`],
    [symbol('StartTask'), `${TYPESPEC}:StartTask`],
  ])('%j → %s', (reference, element) => {
    expect(referencedElement(reference)).toBe(element)
  })
})

describe('parseSpecImpactTrailer', () => {
  it.each([
    'none — keeps the refusal',
    'none -- keeps the refusal',
    'none - keeps the refusal',
    'none: keeps the refusal',
    'none keeps the refusal',
    'none',
  ])('%p を受け入れる', (value) => {
    expect(parseSpecImpactTrailer(value)).toBeUndefined()
  })

  it.each([
    ['modifies — REQ-DEMO-001', 'modifies'],
    ['nonesuch', 'nonesuch'],
  ])('none 以外の値 %p を拒否する', (value, kind) => {
    expect(parseSpecImpactTrailer(value)).toEqual({
      error: expect.stringContaining(`accepts only "none", not "${kind}"`),
    })
  })
})

describe('changedCharacterizations', () => {
  const test = (name: string, body: string): string =>
    `func ${name}(t *testing.T) {\n\t${body}\n}\n`
  const before = [
    'package demo\n',
    test('TestCharacterizeStart', 'check(t, Start(), "started")'),
    test('TestCharacterizeStop', 'check(t, Stop(), "stopped")'),
    test('TestStart', 'check(t, Start(), "started")'),
  ].join('\n')

  it('本文が変わった関数と消えた関数だけを返す', () => {
    const after = [
      'package demo\n',
      test('TestCharacterizeStart', 'check(t, Start(), "running")'),
      test('TestStart', 'check(t, Start(), "running")'),
      test('TestCharacterizeRestart', 'check(t, Restart(), "started")'),
    ].join('\n')
    expect(changedCharacterizations(before, after)).toEqual([
      'TestCharacterizeStart',
      'TestCharacterizeStop',
    ])
  })

  it('関数の外の変更と、新しく足した関数は返さない', () => {
    const after = `${before.replace('package demo\n', 'package demo\n\nimport "testing"\n')}\n${test('TestCharacterizeRestart', 'Restart()')}`
    expect(changedCharacterizations(before, after)).toEqual([])
  })

  it('消したファイルではすべてを返し、新しいファイルでは何も返さない', () => {
    expect(changedCharacterizations(before, undefined)).toEqual([
      'TestCharacterizeStart',
      'TestCharacterizeStop',
    ])
    expect(changedCharacterizations(undefined, before)).toEqual([])
  })
})

describe('workRangeStart', () => {
  it('状態が初めて pending でなくなったコミットの親を起点にする', () => {
    expect(
      workRangeStart(
        [
          { sha: 'filed', status: 'pending' },
          { sha: 'started', status: 'in_progress' },
          { sha: 'finished', status: 'completed' },
        ],
        'completed',
        'head',
      ),
    ).toBe('started^')
  })

  it('checkpoint を畳んだ履歴では、completed になったコミットの親を起点にする', () => {
    expect(
      workRangeStart(
        [
          { sha: 'filed', status: 'pending' },
          { sha: 'finished', status: 'completed' },
        ],
        'completed',
        'head',
      ),
    ).toBe('finished^')
  })

  it('着手がまだコミットされていなければ HEAD を起点にする', () => {
    expect(workRangeStart([{ sha: 'filed', status: 'pending' }], 'pending', 'head')).toBe('head')
    expect(workRangeStart([], undefined, 'head')).toBe('head')
  })

  it('コミット済みの状態が着手後なのに遷移を見つけられなければ、起点を返さない', () => {
    expect(workRangeStart([{ sha: 'filed', status: 'pending' }], 'in_progress', 'head')).toBe(
      undefined,
    )
    expect(workRangeStart([{ sha: 'broken', status: undefined }], 'completed', 'head')).toBe(
      undefined,
    )
  })
})

describe('verifySpecImpact の work item', () => {
  it('affected_spec の作業項目は、挙げた要素を変えたかどうかを問わず通す', () => {
    expect(
      verifyItems(
        affected(
          [requirement('REQ-DEMO-001'), symbol('StartTask')],
          range({ diff: diff({ changedScenarios: ['REQ-DEMO-001'] }) }),
        ),
        affected([requirement('REQ-DEMO-001'), symbol('StartTask')]),
      ),
    ).toEqual([])
  })

  it('別の work item が挙げた差分は、none の矛盾にしない', () => {
    const others = range({
      diff: diff({ changedScenarios: ['REQ-DEMO-001'] }),
      claimedByOthers: [requirement('REQ-DEMO-001')],
    })
    expect(verifyItems(none(others))).toEqual([])
  })

  it('状態遷移の差分は、同じ範囲に仕様を変える別の work item があれば none の矛盾にしない', () => {
    const transitions = diff({ changedTransitions: ['docs/modules/demo#Lifecycle'] })
    expect(
      verifyItems(
        none(range({ diff: transitions, claimedByOthers: [requirement('REQ-DEMO-009')] })),
      ),
    ).toEqual([])
    expect(verifyItems(none(range({ diff: transitions })))).toEqual([
      expect.stringContaining('state transitions of docs/modules/demo#Lifecycle'),
    ])
  })

  it('削除した宣言は、同じ範囲に仕様を変える別の work item があれば none の矛盾にしない', () => {
    const removed = diff({ removedDeclarations: [`${TYPESPEC}:Legacy`] })
    expect(
      verifyItems(none(range({ diff: removed, claimedByOthers: [requirement('REQ-DEMO-009')] }))),
    ).toEqual([])
    expect(verifyItems(none(range({ diff: removed })))).toEqual([
      expect.stringContaining(`${TYPESPEC}:Legacy`),
    ])
  })

  it.each([
    ['シナリオ', { changedScenarios: ['REQ-DEMO-001'] }, 'REQ-DEMO-001'],
    ['標準の行', { addedStandards: [`${STANDARDS}#RFC-DEMO-ONE`] }, 'RFC-DEMO-ONE'],
    ['宣言の変更', { changedDeclarations: [`${TYPESPEC}:Task`] }, `${TYPESPEC}:Task`],
    ['宣言の削除', { removedDeclarations: [`${TYPESPEC}:Task`] }, `${TYPESPEC}:Task`],
    ['状態遷移', { changedTransitions: ['docs/modules/demo#Lifecycle'] }, 'Lifecycle'],
    ['非推奨の指定', { addedDeprecations: [`${TYPESPEC}:Task`] }, `${TYPESPEC}:Task`],
  ])('spec_impact: none の作業範囲に %s の差分があれば失敗する', (_kind, changes, named) => {
    const findings = verifyItems(none(range({ diff: diff(changes) })))
    expect(findings).toEqual([expect.stringContaining('spec_impact is none')])
    expect(findings[0]).toContain(named)
  })

  it('spec_impact: none の作業範囲の区間のどれかに差分があれば失敗する', () => {
    const findings = verifyItems(
      none({
        ...range(),
        diffs: [diff(), diff({ changedScenarios: ['REQ-DEMO-002'] })],
      }),
    )
    expect(findings).toEqual([expect.stringContaining('REQ-DEMO-002')])
  })

  it('spec_impact.reason の言い回しを判定しない', () => {
    expect(verifyItems(none(range(), 'リファクタリングのみ'))).toEqual([])
  })

  it('作業範囲の起点を求められなければ失敗として報告する', () => {
    expect(verifyItems({ ...none(), range: undefined })).toEqual([
      expect.stringContaining('its work range is unknown'),
    ])
  })
})

describe('verifySpecImpact のコミット', () => {
  it('本番コードを変更し、トレーラーのないコミットを失敗にする', () => {
    const findings = verifyCommits(
      commit({ productionPaths: ['backend/demo/a.go', 'backend/demo/b.go'] }),
    )
    expect(findings).toEqual([
      expect.stringContaining(
        '01234567 refactor(demo): extract helper: changes production code (backend/demo/a.go and 1 more)',
      ),
    ])
  })

  it('理由を問わず none のトレーラーのあるコミットと、本番コードを変えないコミットを通す', () => {
    expect(
      verifyCommits(
        commit({ trailer: `none — ${CONCRETE}` }),
        commit({ trailer: 'none' }),
        commit({ productionPaths: [] }),
      ),
    ).toEqual([])
  })

  it('none 以外のトレーラーを失敗にする', () => {
    expect(verifyCommits(commit({ trailer: 'conforms — REQ-DEMO-001' }))).toEqual([
      expect.stringContaining('accepts only "none"'),
    ])
  })

  it('トレーラーを付けたコミットが規範仕様を変更していれば失敗する', () => {
    const findings = verifyCommits(
      commit({
        productionPaths: [],
        trailer: `none — ${CONCRETE}`,
        diff: diff({ changedScenarios: ['REQ-DEMO-001'] }),
      }),
    )
    expect(findings).toEqual([
      expect.stringContaining(
        'declares Spec-Impact: none, but changes the normative specification: REQ-DEMO-001',
      ),
    ])
  })
})

describe('verifySpecImpact の申告漏れ', () => {
  const changed = diff({
    addedScenarios: ['REQ-DEMO-002'],
    changedScenarios: ['REQ-DEMO-001'],
    changedStandards: [`${STANDARDS}#RFC-DEMO-ONE`],
    changedDeclarations: [`${TYPESPEC}:Task`],
    removedScenarios: ['REQ-DEMO-009'],
    changedTransitions: ['docs/modules/demo#Lifecycle'],
  })

  it('差分の規範要素がどの work item の affected_spec にもなければ失敗する', () => {
    const findings = verifySpecImpact({ items: [], commits: [], diffs: [changed] })
    expect(findings.map((finding) => finding.split(':')[0])).toEqual([
      'REQ-DEMO-001',
      'REQ-DEMO-002',
      `${STANDARDS}#RFC-DEMO-ONE`,
      'spec/modules/demo/main.tsp',
    ])
  })

  it('区間ごとの差分を合わせて申告漏れを探す', () => {
    const findings = verifySpecImpact({
      items: [affected([requirement('REQ-DEMO-001')], range())],
      commits: [],
      diffs: [
        diff({ changedScenarios: ['REQ-DEMO-001'] }),
        diff({ changedScenarios: ['REQ-DEMO-002'] }),
      ],
    })
    expect(findings).toEqual([expect.stringMatching(/^REQ-DEMO-002: /)])
  })

  it('すべての追加と変更が affected_spec に挙がっていれば通す', () => {
    const references = [
      requirement('REQ-DEMO-001'),
      requirement('REQ-DEMO-002'),
      requirement('RFC-DEMO-ONE'),
      symbol('Task'),
    ]
    expect(
      verifySpecImpact({
        items: [affected(references, range({ diff: changed }))],
        commits: [],
        diffs: [changed],
      }),
    ).toEqual([])
  })
})

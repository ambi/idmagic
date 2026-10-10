import { describe, expect, it } from 'bun:test'
import { diffSpecifications, type SpecificationDiff } from './spec-diff.ts'
import {
  type CheckedCommit,
  type CheckedWorkItem,
  type SpecReference,
  type WorkRange,
  changedCharacterizations,
  isBoilerplateReason,
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

const range = (changes: Partial<WorkRange> = {}): WorkRange => ({
  diff: diff(),
  claimedByOthers: [],
  changedTests: [],
  changedCharacterizations: [],
  ...changes,
})

const requirement = (id: string, impact: SpecReference['impact'] = 'modifies'): SpecReference => ({
  path: id.startsWith('REQ-') || id.startsWith('EX-') ? SCENARIOS : STANDARDS,
  requirement: id,
  impact,
})

const symbol = (name: string, impact: SpecReference['impact'] = 'modifies'): SpecReference => ({
  path: TYPESPEC,
  symbol: `Demo.Operations.${name}`,
  impact,
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
  workItemDeclared: false,
  trailer: undefined,
  diff: diff(),
  changedCharacterizations: [],
  ...changes,
})

const verifyItems = (...items: CheckedWorkItem[]): string[] =>
  verifySpecImpact({ items, commits: [], diff: diff() })

const verifyCommits = (...commits: CheckedCommit[]): string[] =>
  verifySpecImpact({ items: [], commits, diff: diff() })

describe('specImpactDeclaration', () => {
  it('impact を省いた項目を modifies として読む', () => {
    expect(
      specImpactDeclaration({
        affected_spec: [
          { path: SCENARIOS, requirement: 'REQ-DEMO-001' },
          { path: SCENARIOS, requirement: 'REQ-DEMO-002', impact: 'conforms' },
          { path: TYPESPEC, symbol: 'Demo.Task', impact: 'modifies' },
          { scl: 'legacy' },
        ],
      }),
    ).toEqual({
      kind: 'affected',
      references: [
        { path: SCENARIOS, requirement: 'REQ-DEMO-001', symbol: undefined, impact: 'modifies' },
        { path: SCENARIOS, requirement: 'REQ-DEMO-002', symbol: undefined, impact: 'conforms' },
        { path: TYPESPEC, requirement: undefined, symbol: 'Demo.Task', impact: 'modifies' },
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

describe('isBoilerplateReason', () => {
  it.each([
    '',
    'リファクタリングのみ',
    'リファクタリングのみで、振る舞いは変えない。',
    '仕様に影響しない。',
    '内部の実装を整理するだけで、挙動は変わらない。',
    'refactoring only',
    'No functional change.',
    'Pure refactor, no behavior change',
    'n/a',
  ])('定型句だけの理由を拒否する: %p', (reason) => {
    expect(isBoilerplateReason(reason)).toBe(true)
  })

  it.each([
    CONCRETE,
    'RevokeFamily が失効させたトークンの一覧を返すだけで、失効の対象は変えない。',
    'The order of returned tokens, the error kinds, and the emitted events stay the same.',
    'Extracts the PKCE comparison; refusal and audit event are unchanged.',
  ])('維持するものを名指す理由を受け入れる: %p', (reason) => {
    expect(isBoilerplateReason(reason)).toBe(false)
  })
})

describe('parseSpecImpactTrailer', () => {
  it.each([
    ['none — keeps the refusal', 'keeps the refusal'],
    ['none -- keeps the refusal', 'keeps the refusal'],
    ['none - keeps the refusal', 'keeps the refusal'],
    ['none: keeps the refusal', 'keeps the refusal'],
    ['none keeps the refusal', 'keeps the refusal'],
    ['none', ''],
  ])('%p から理由を読む', (value, reason) => {
    expect(parseSpecImpactTrailer(value)).toEqual({ reason })
  })

  it('none 以外の値を拒否する', () => {
    expect(parseSpecImpactTrailer('modifies — REQ-DEMO-001')).toEqual({
      error: expect.stringContaining('accepts only "none", not "modifies"'),
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
  it('modifies の規範要素が差分にあれば通す', () => {
    expect(
      verifyItems(
        affected(
          [
            requirement('REQ-DEMO-001'),
            requirement('EX-DEMO-002-01'),
            requirement('RFC-DEMO-ONE'),
            symbol('StartTask'),
            symbol('Task'),
          ],
          range({
            diff: diff({
              changedScenarios: ['REQ-DEMO-001'],
              addedScenarios: ['REQ-DEMO-002'],
              removedStandards: [`${STANDARDS}#RFC-DEMO-ONE`],
              changedDeclarations: [`${TYPESPEC}:StartTask`],
              addedDeclarations: [`${TYPESPEC}:Task`],
            }),
          }),
        ),
      ),
    ).toEqual([])
  })

  it('modifies の規範要素が差分になければ失敗する', () => {
    const findings = verifyItems(
      affected(
        [requirement('REQ-DEMO-001'), symbol('StartTask')],
        range({ diff: diff({ changedScenarios: ['REQ-DEMO-009'] }) }),
      ),
    )
    expect(findings).toEqual([
      expect.stringContaining(
        `${ITEM}: affected_spec ${SCENARIOS} REQ-DEMO-001 is impact: modifies`,
      ),
      expect.stringContaining('Demo.Operations.StartTask is impact: modifies'),
    ])
  })

  it('conforms の規範要素が差分になく、変更したテストがそれを引いていれば通す', () => {
    expect(
      verifyItems(
        affected(
          [
            requirement('REQ-DEMO-001', 'conforms'),
            requirement('RFC-DEMO-ONE', 'conforms'),
            symbol('StartTask', 'conforms'),
          ],
          range({
            changedTests: [
              '//spec:covers EX-DEMO-001-02: 拒否した後に状態を変えない\nfunc TestRefusal(t *testing.T) {}',
              '//spec:covers RFC-DEMO-ONE: 署名を検証する\nfunc TestStart(t *testing.T) { usecases.StartTask(ctx) }',
            ],
          }),
        ),
      ),
    ).toEqual([])
  })

  it('conforms の規範要素を引くテストが作業範囲になければ失敗する', () => {
    const findings = verifyItems(
      affected(
        [requirement('REQ-DEMO-001', 'conforms'), symbol('StartTask', 'conforms')],
        range({
          changedTests: [
            // 別の規則の例と、散文での言及は、引用に数えない。
            '//spec:covers EX-DEMO-0011-01: 別の規則\n// REQ-DEMO-001 は別のテストが担当する\nfunc TestStartTaskRetry(t *testing.T) {}',
          ],
        }),
      ),
    )
    expect(findings).toEqual([
      expect.stringContaining('REQ-DEMO-001 is impact: conforms, but no test'),
      expect.stringContaining('StartTask is impact: conforms, but no test'),
    ])
  })

  it('conforms の規範要素が差分にあれば失敗する', () => {
    const findings = verifyItems(
      affected(
        [requirement('REQ-DEMO-001', 'conforms')],
        range({
          diff: diff({ changedScenarios: ['REQ-DEMO-001'] }),
          changedTests: ['//spec:covers REQ-DEMO-001: 結果\nfunc TestDemo(t *testing.T) {}'],
        }),
      ),
    )
    expect(findings).toEqual([
      expect.stringContaining('is impact: conforms, but the work range changes it'),
    ])
  })

  it('別の work item が modifies として挙げた差分は、conforms と none の矛盾にしない', () => {
    const others = range({
      diff: diff({ changedScenarios: ['REQ-DEMO-001'] }),
      claimedByOthers: [requirement('REQ-DEMO-001')],
      changedTests: ['//spec:covers REQ-DEMO-001: 結果\nfunc TestDemo(t *testing.T) {}'],
    })
    expect(verifyItems(affected([requirement('REQ-DEMO-001', 'conforms')], others))).toEqual([])
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

  it('別の work item が conforms として挙げただけの差分は、none の矛盾として残す', () => {
    const findings = verifyItems(
      none(
        range({
          diff: diff({ changedScenarios: ['REQ-DEMO-001'] }),
          claimedByOthers: [requirement('REQ-DEMO-001', 'conforms')],
        }),
      ),
    )
    expect(findings).toEqual([expect.stringContaining('spec_impact is none')])
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

  it('spec_impact.reason が定型句だけなら失敗する', () => {
    expect(verifyItems(none(range(), 'リファクタリングのみ'))).toEqual([
      expect.stringContaining('spec_impact.reason names nothing'),
    ])
  })

  it('作業範囲の起点を求められなければ失敗として報告する', () => {
    expect(verifyItems({ ...none(), range: undefined })).toEqual([
      expect.stringContaining('its work range is unknown'),
    ])
  })
})

describe('verifySpecImpact のコミット', () => {
  it('本番コードを変更し、宣言のないコミットを失敗にする', () => {
    const findings = verifyCommits(
      commit({ productionPaths: ['backend/demo/a.go', 'backend/demo/b.go'] }),
    )
    expect(findings).toEqual([
      expect.stringContaining(
        '01234567 refactor(demo): extract helper: changes production code (backend/demo/a.go and 1 more)',
      ),
    ])
  })

  it('work item の宣言、または具体的なトレーラーのあるコミットを通す', () => {
    expect(
      verifyCommits(
        commit({ workItemDeclared: true }),
        commit({ trailer: `none — ${CONCRETE}` }),
        commit({ productionPaths: [] }),
      ),
    ).toEqual([])
  })

  it('トレーラーの理由が空、または定型句だけなら失敗する', () => {
    expect(
      verifyCommits(commit({ trailer: 'none' }), commit({ trailer: 'none — refactoring only' })),
    ).toEqual([
      expect.stringContaining('Spec-Impact reason names nothing'),
      expect.stringContaining('Spec-Impact reason names nothing'),
    ])
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

  it('差分の規範要素がどの work item の modifies にもなければ失敗する', () => {
    const findings = verifySpecImpact({ items: [], commits: [], diff: changed })
    expect(findings.map((finding) => finding.split(':')[0])).toEqual([
      'REQ-DEMO-001',
      'REQ-DEMO-002',
      `${STANDARDS}#RFC-DEMO-ONE`,
      'spec/modules/demo/main.tsp',
    ])
  })

  it('conforms として挙げただけでは申告にならない', () => {
    const references = [
      requirement('REQ-DEMO-001', 'conforms'),
      requirement('REQ-DEMO-002'),
      requirement('RFC-DEMO-ONE'),
      symbol('Task'),
    ]
    const findings = verifySpecImpact({
      items: [affected(references, range({ diff: changed }))],
      commits: [],
      diff: changed,
    })
    expect(findings.filter((finding) => finding.startsWith('REQ-DEMO-001:'))).toHaveLength(1)
    expect(findings.filter((finding) => finding.includes('no work item changed'))).toHaveLength(1)
  })

  it('すべての追加と変更が modifies に挙がっていれば通す', () => {
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
        diff: changed,
      }),
    ).toEqual([])
  })
})

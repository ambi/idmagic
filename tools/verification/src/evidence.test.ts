import { expect, test } from 'bun:test'
import {
  evaluateVerification,
  executionResults,
  changedOperations,
  referenceResults,
  databaseResults,
  collectReferences,
  collectOperationReferences,
  moduleDependencies,
  parseGoEvents,
  type TestReference,
} from './evidence.ts'

const reference: TestReference = {
  package: 'example/backend/demo/handlers_http',
  name: 'TestContract',
  path: 'backend/demo/handlers_http/contract_test.go',
  normativeIds: ['REQ-DEMO-001'],
  operationIds: ['Create'],
}
const operations = [{ operationId: 'Create', state: 'unverified' as const, reasons: ['dynamic'] }]

test('操作宣言の不正な形式と対応しない配置を行番号付きで診断する', () => {
  const source = `package demo
//verify:operation Create
func TestNoColon(t *testing.T) {}
//verify:operation Create: \t
func TestNoObservation(t *testing.T) {}
//verify:operation CreateExtra: 本文と状態コード
func TestUnknown(t *testing.T) {}
//verify:operation Create: 本文と状態コード

func TestDetached(t *testing.T) {}
//verify:operation Create: 本文と状態コード
func helper(t *testing.T) {}
func TestNested(t *testing.T) {
  //verify:operation Create: 本文と状態コード
}
//verify:operation Create: 本文と状態コード
`
  const result = collectOperationReferences(source, ['Create'])
  expect(result.references).toEqual([])
  expect(result.findings).toEqual([
    { line: 2, message: 'missing colon after operation ID' },
    { line: 4, message: 'missing contract observation after colon' },
    { line: 6, message: 'unknown operation ID: CreateExtra' },
    { line: 8, message: 'directive is not directly above a Go test' },
    { line: 11, message: 'directive is not directly above a Go test' },
    { line: 14, message: 'directive is not directly above a Go test' },
    { line: 16, message: 'directive is not directly above a Go test' },
  ])
})

test('fuzz: 宣言した操作だけを受理し、ID と構文を改変した宣言は診断する', () => {
  for (let seed = 1; seed <= 64; seed += 1) {
    const id = `Create${seed}`
    const sourceFor = (directive: string) => `${directive}\nfunc TestContract(t *testing.T) {}\n`
    const valid = sourceFor(`//verify:operation ${id}: 本文と状態コード`)
    expect(collectOperationReferences(valid, [id])).toEqual({
      references: [{ name: 'TestContract', operationId: id }],
      findings: [],
    })
    expect(
      collectReferences('example', [{ path: reference.path, source: valid }], [], [id]),
    ).toEqual([{ ...reference, normativeIds: [], operationIds: [id] }])
    for (const broken of [
      `//verify:operation ${id}Extra: 本文と状態コード`,
      `//verify:operation ${id}`,
      `//verify:operation ${id}: \t`,
      `//verify:operation: ${id}: 本文と状態コード`,
    ]) {
      const source = sourceFor(broken)
      const result = collectOperationReferences(source, [id])
      expect(result.references).toEqual([])
      expect(result.findings).toHaveLength(1)
      expect(collectReferences('example', [{ path: reference.path, source }], [], [id])).toEqual([])
    }
  }
})

test('同じ操作の別宣言で不正な配置を隠さず、複数操作と通常のコメントを区別する', () => {
  const source = `package demo
//verify:operation Create: 本文と状態コード
//spec:covers REQ-DEMO-001: 応答
//verify:operation Read: 本文と状態コード
func TestContract(t *testing.T) {
  t.Log("//verify:operation Unknown: 本文と状態コード")
}
//verify:operation Create: 本文と状態コード
func helper() {}
// verify:operation Unknown: 通常のコメント
`
  expect(collectOperationReferences(source, ['Create', 'Read'])).toEqual({
    references: [
      { name: 'TestContract', operationId: 'Create' },
      { name: 'TestContract', operationId: 'Read' },
    ],
    findings: [{ line: 8, message: 'directive is not directly above a Go test' }],
  })
})

test('fuzz: イベントの JSON 往復を保存し、壊した形を成功へ昇格させない', () => {
  for (let seed = 1; seed <= 64; seed += 1) {
    const event = {
      Action: 'pass',
      Package: `example/module${seed}`,
      Test: `TestCase${seed}`,
      Output: `quoted " ${seed}\nnext line`,
    }
    expect(parseGoEvents(`${JSON.stringify(event)}\n`).events).toEqual([event])
    for (const broken of [
      null,
      { ...event, Test: seed },
      { ...event, Package: null },
      { ...event, Output: false },
    ]) {
      const result = parseGoEvents(JSON.stringify(broken))
      expect(result.events).toEqual([])
      expect(result.failures).toHaveLength(1)
    }
    expect(parseGoEvents(`invalid-${seed}`).failures).toHaveLength(1)
  }
})

test('規範参照だけがあり実行イベントがないテストは未実行のまま残る', () => {
  const facts = referenceResults([reference], [])
  expect(facts[0]).toMatchObject({ normativeIds: ['REQ-DEMO-001'], state: 'not-run' })
  expect(evaluateVerification(operations, new Set(['Create']), facts, [], false)).toEqual([
    'Create: no successful contract test for changed unverified operation',
  ])
})

test('成功と拒否の契約証拠が分かれる場合は両方の成功を要求する', () => {
  const success = { ...reference, name: 'TestSuccess', state: 'passed' as const }
  for (const state of ['skipped', 'not-run', 'failed'] as const) {
    const refusal = { ...reference, name: 'TestRefusal', state }
    expect(
      evaluateVerification(operations, new Set(['Create']), [success, refusal], [], false),
    ).toEqual([`Create: incomplete contract test evidence (TestRefusal: ${state})`])
  }
  expect(
    evaluateVerification(
      operations,
      new Set(['Create']),
      [success, { ...reference, name: 'TestRefusal', state: 'passed' }],
      [],
      false,
    ),
  ).toEqual([])
})

test('DB パッケージのテストがゼロ、スキップ、起動失敗なら未検証とする', () => {
  const pkg = reference.package
  for (const events of [
    [{ Action: 'pass', Package: pkg }],
    [
      { Action: 'skip', Package: pkg, Test: 'TestStore' },
      { Action: 'pass', Package: pkg },
    ],
    [
      { Action: 'output', Package: pkg, Output: 'pgtest: unavailable; skipping DB tests\n' },
      { Action: 'pass', Package: pkg, Test: 'TestStore' },
      { Action: 'pass', Package: pkg },
    ],
  ]) {
    expect(databaseResults([pkg], events, executionResults(events))[0]?.state).toBe('unverified')
  }
  const success = [
    { Action: 'pass', Package: pkg, Test: 'TestStore' },
    { Action: 'pass', Package: pkg },
  ]
  expect(databaseResults([pkg], success, executionResults(success))[0]?.state).toBe('verified')
})

test('規範参照と代替証拠をテストの直上の宣言から独立して導く', () => {
  const source = `package demo
// REQ-DEMO-002 は別項目の担当。
//spec:covers REQ-DEMO-001: 拒否後の状態
//verify:operation Create: 本文と状態コード
func TestContract(t *testing.T) {
  t.Log("case")
}
//verify:operation Read: 本文と状態コード
//spec:covers REQ-DEMO-001: 応答
func TestRead(t *testing.T) {
  t.Skip("unavailable")
}
`
  const refs = collectReferences(
    'example',
    [{ path: reference.path, source }],
    ['REQ-DEMO-001', 'REQ-DEMO-002'],
    ['Create', 'Read'],
  )
  expect(refs).toEqual([reference, { ...reference, name: 'TestRead', operationIds: ['Read'] }])
})

test('親が pass でも子の skip と fail を成功の代替証拠にしない', () => {
  for (const action of ['skip', 'fail']) {
    const tests = executionResults([
      { Action: 'pass', Package: reference.package, Test: reference.name },
      { Action: action, Package: reference.package, Test: `${reference.name}/response` },
      { Action: 'pass', Package: reference.package },
    ])
    const facts = referenceResults([reference], tests)
    expect(facts[0]?.state).toBe(action === 'skip' ? 'skipped' : 'failed')
    expect(evaluateVerification(operations, new Set(['Create']), facts, [], false)).toHaveLength(1)
  }
})

test('全ケースが成功した同じ実行の契約テストだけが未検証操作を補う', () => {
  const tests = executionResults([
    { Action: 'pass', Package: reference.package, Test: reference.name },
    { Action: 'pass', Package: reference.package },
  ])
  const facts = referenceResults([reference], tests)
  expect(evaluateVerification(operations, new Set(['Create']), facts, [], true)).toEqual([])
  expect(evaluateVerification(operations, new Set(['Other']), [], [], true)).toEqual([])
  expect(
    evaluateVerification(
      [{ ...operations[0]!, state: 'failed' }],
      new Set(['Create']),
      facts,
      [],
      true,
    ),
  ).toHaveLength(1)
})

test('DB の未実行を必須モードで拒否し任意モードでは未検証として保存する', () => {
  const db = [
    { package: reference.package, state: 'unverified' as const, reasons: ['DB unavailable'] },
  ]
  expect(evaluateVerification([], new Set(), [], db, true)).toEqual([
    `${reference.package}: required DB verification is unverified (DB unavailable)`,
  ])
  expect(evaluateVerification([], new Set(), [], db, false)).toEqual([])
})

test('パッケージが失敗または終了していない実行はテストの pass だけで成功にしない', () => {
  for (const action of ['fail', 'start']) {
    const tests = executionResults([
      { Action: 'pass', Package: reference.package, Test: reference.name },
      { Action: action, Package: reference.package },
    ])
    expect(referenceResults([reference], tests)[0]?.state).toBe(
      action === 'fail' ? 'failed' : 'not-run',
    )
  }
})

test('変更モジュールと shared を保守的に選び、未解決経路を対象から落とさない', () => {
  const owners = [
    { operationId: 'Create', module: 'demo' },
    { operationId: 'Read', module: 'other' },
    { operationId: 'Unknown' },
  ]
  expect([...changedOperations(['backend/demo/domain/value.go'], owners)]).toEqual([
    'Create',
    'Unknown',
  ])
  expect([...changedOperations(['backend/shared/http/writer.go'], owners)]).toEqual([
    'Create',
    'Read',
    'Unknown',
  ])
  expect([...changedOperations(['spec/modules/demo/main.tsp'], owners)]).toEqual([
    'Create',
    'Read',
    'Unknown',
  ])
  expect([
    ...changedOperations(
      ['backend/demo/domain/value_test.go', 'docs/development/testing.md'],
      owners,
    ),
  ]).toEqual([])
})

test('別モジュールの変更を、そのモジュールを使う操作まで推移的に伝える', () => {
  const dependencies = moduleDependencies(
    [
      { path: 'backend/other/service.go', source: 'import alias "example/backend/demo/domain"' },
      { path: 'backend/third/service.go', source: 'import "example/backend/other/ports"' },
      { path: 'backend/unrelated/service_test.go', source: 'import "example/backend/demo/domain"' },
    ],
    'example',
  )
  expect([
    ...changedOperations(
      ['backend/demo/domain/value.go'],
      [
        { operationId: 'Caller', module: 'other' },
        { operationId: 'Indirect', module: 'third' },
        { operationId: 'Unrelated', module: 'unrelated' },
      ],
      dependencies,
    ),
  ]).toEqual(['Caller', 'Indirect'])
})

test('生成した子テストのスキップは深さとケース数によらず代替証拠を無効にする', () => {
  for (let depth = 1; depth <= 32; depth += 1) {
    const child = `${reference.name}/${Array.from({ length: depth }, (_, index) => `case${index}`).join('/')}`
    const tests = executionResults([
      { Action: 'pass', Package: reference.package, Test: reference.name },
      { Action: 'skip', Package: reference.package, Test: child },
      { Action: 'pass', Package: reference.package },
    ])
    const facts = referenceResults([reference], tests)
    expect(facts[0]?.state).toBe('skipped')
    expect(evaluateVerification(operations, new Set(['Create']), facts, [], false)).toHaveLength(1)
  }
})

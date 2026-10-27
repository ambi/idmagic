import { describe, expect, it } from 'bun:test'
import {
  renderStateMachine,
  stateMachineFindings,
  updateStateMachineBlocks,
} from './state-machines.ts'

const fixture = 'backend/demo/testdata/DemoLifecycle.statemachine.json'
const machine = {
  name: 'DemoLifecycle',
  states: [
    { name: 'Active', kind: 'initial', meaning: '使える' },
    { name: 'Closed', kind: 'terminal', meaning: '`closed_at` を持つ' },
  ],
  transitions: [
    { from: 'Active', event: 'DemoClosed', guard: 'a || b', to: 'Closed', effects: '' },
  ],
  operations: ['閉じる', '開く'],
  matrix: {
    Active: {
      閉じる: [
        { kind: 'transition', to: 'Closed', condition: '期限内' },
        { kind: 'refusal', response: '409 demo_expired', condition: '期限切れ' },
      ],
      開く: [{ kind: 'noop' }],
    },
    Closed: {
      閉じる: [{ kind: 'noop' }],
      開く: [{ kind: 'refusal', response: '409 demo_closed' }],
    },
  },
}
const tables = [
  '| State | Kind | Meaning |',
  '| --- | --- | --- |',
  '| Active | initial | 使える |',
  '| Closed | terminal | `closed_at` を持つ |',
  '',
  '| From | Event | Guard | To | Effects |',
  '| --- | --- | --- | --- | --- |',
  '| Active | DemoClosed | a \\|\\| b | Closed |  |',
  '',
  '| State | 閉じる | 開く |',
  '| --- | --- | --- |',
  '| Active | → Closed（期限内）<br>拒否：409 demo_expired（期限切れ） | 何もしない |',
  '| Closed | 何もしない | 拒否：409 demo_closed |',
].join('\n')

const document = (block: string) =>
  `## 状態遷移\n\n### DemoLifecycle\n\n説明を残す。\n\n${block}\n\n## 品質\n`
const marked = (body: string) =>
  `<!-- spec:state-machine ${fixture} -->\n${body}\n<!-- /spec:state-machine -->`

describe('状態機械の表の生成', () => {
  it('三つの表を JSON から生成し、ガードの | を戻す', () => {
    expect(renderStateMachine(machine, 'DemoLifecycle')).toBe(tables)
  })

  it('マトリクスの行を、状態の表ではなく一次情報の matrix の順序で並べる', () => {
    const reordered = {
      ...machine,
      matrix: { Closed: machine.matrix.Closed, Active: machine.matrix.Active },
    }
    const rows = renderStateMachine(reordered, 'DemoLifecycle').split('\n').slice(-2)
    expect(rows.map((row) => row.split(' | ')[0])).toEqual(['| Closed', '| Active'])
  })

  it('区切りのコメントの中だけを置き換え、前後の本文を残す', () => {
    const generated = updateStateMachineBlocks(document(marked('古い表')), () =>
      JSON.stringify(machine),
    )
    expect(generated).toBe(document(marked(`\n${tables}\n`)))
  })

  it('JSON の名前が見出しと違う状態機械を拒否する', () => {
    expect(() =>
      updateStateMachineBlocks(document(marked('')), () =>
        JSON.stringify({ ...machine, name: 'OtherLifecycle' }),
      ),
    ).toThrow('state machine source names OtherLifecycle under ### DemoLifecycle')
  })

  it('マトリクスに欠けたセルがある一次情報を拒否する', () => {
    const missing = {
      ...machine,
      matrix: { ...machine.matrix, Closed: { 閉じる: [{ kind: 'noop' }] } },
    }
    expect(() => renderStateMachine(missing, 'DemoLifecycle')).toThrow(
      'state matrix gives no outcome for Closed × 開く',
    )
  })

  it('知らない欄と結果の種類を拒否する', () => {
    expect(() => renderStateMachine({ ...machine, extra: 1 }, 'DemoLifecycle')).toThrow(
      'unknown state machine field extra',
    )
    const unknownKind = {
      ...machine,
      matrix: {
        ...machine.matrix,
        Closed: { 閉じる: [{ kind: 'skip' }], 開く: [{ kind: 'noop' }] },
      },
    }
    expect(() => renderStateMachine(unknownKind, 'DemoLifecycle')).toThrow(
      'state matrix outcome for Closed × 閉じる must be transition, noop, or refusal',
    )
  })

  it('testdata の外の一次情報を拒否する', () => {
    const outside = document(
      '<!-- spec:state-machine docs/demo.statemachine.json -->\n<!-- /spec:state-machine -->',
    )
    expect(() => updateStateMachineBlocks(outside, () => '{}')).toThrow(
      'invalid state machine source path: docs/demo.statemachine.json',
    )
  })
})

describe('状態機械の一次情報の検査', () => {
  it('区切りのコメントのない手書きの状態機械を報告する', () => {
    expect(stateMachineFindings(document(tables), () => undefined)).toEqual([
      'state machine DemoLifecycle is not generated; move its tables to testdata and run mise run generate-spec-from-testdata',
    ])
  })

  it('一次情報とずれた表を報告し、一致する表は報告しない', () => {
    const read = () => JSON.stringify(machine)
    expect(stateMachineFindings(document(marked('古い表')), read)).toEqual([
      'state machine tables drift from their testdata; run mise run generate-spec-from-testdata',
    ])
    expect(stateMachineFindings(document(marked(`\n${tables}\n`)), read)).toEqual([])
  })
})

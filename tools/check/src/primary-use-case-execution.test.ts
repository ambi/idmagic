import { describe, expect, it } from 'bun:test'
import type { VerificationReport } from '../../verification/src/report.ts'
import { passedPrimaryUseCase } from './primary-use-case-execution.ts'

const reference = {
  package: 'demo',
  path: 'backend/demo/demo_test.go',
  name: 'TestDemo',
  normativeIds: ['REQ-DEMO-001'],
  operationIds: [],
  state: 'passed' as const,
}
const report: VerificationReport = {
  input: { revision: 'revision', fingerprint: 'current' },
  tests: [{ package: 'demo', name: 'TestDemo', state: 'passed' }],
  references: [reference],
  databases: [],
  checks: [],
  failures: [],
  changeResistance: { state: 'not-collected' },
}
const passed = (value: VerificationReport) =>
  passedPrimaryUseCase(value, 'current', reference.path, reference.name, 'REQ-DEMO-001')

describe('主要ユースケースの実行との照合', () => {
  it('同じ入力の、選んだテスト自身の成功だけを受理する', () => {
    expect(passed(report)).toBe(true)
    for (const value of [
      { ...report, input: { ...report.input, fingerprint: 'stale' } },
      { ...report, failures: ['gate failed'] },
      { ...report, tests: [] },
      { ...report, references: [{ ...reference, path: 'another_test.go' }] },
      { ...report, references: [{ ...reference, name: 'TestOther' }] },
      { ...report, references: [{ ...reference, normativeIds: ['REQ-OTHER-001'] }] },
    ])
      expect(passed(value)).toBe(false)
  })

  it('Skip、失敗、未実行と子テストの省略を親の成功にしない', () => {
    for (const state of ['skipped', 'failed', 'not-run'] as const) {
      expect(passed({ ...report, references: [{ ...reference, state }] })).toBe(false)
      expect(passed({ ...report, tests: [{ ...report.tests[0]!, state }] })).toBe(false)
      expect(
        passed({
          ...report,
          tests: [...report.tests, { package: 'demo', name: 'TestDemo/required', state }],
        }),
      ).toBe(false)
    }
  })
})

import { describe, expect, it } from 'bun:test'
import { verifyMutationCanary } from './mutation-canary.ts'

// gamut 0.1.0 の mutation-testing-elements 報告から、検算に必要なフィールドを抜き出す。
const report = (boundaryStatus = 'Survived', arithmeticStatus = 'CompileError') => ({
  files: {
    'tools/check/testdata/mutation-canary/canary.go': {
      mutants: [
        { mutatorName: 'CONDITIONALS_BOUNDARY', status: boundaryStatus },
        { mutatorName: 'ARITHMETIC_BASE', status: arithmeticStatus },
      ],
    },
  },
})

describe('verifyMutationCanary', () => {
  it('既知の生存と構築不能を受け入れる', () => {
    expect(verifyMutationCanary(report())).toEqual([])
  })

  it('構築不能を検出成功と数える実行器を拒否する', () => {
    expect(verifyMutationCanary(report('Survived', 'Killed'))).toContain(
      'expected an ARITHMETIC_BASE mutant with status CompileError',
    )
  })

  it('既知の生存を失う実行器を拒否する', () => {
    expect(verifyMutationCanary(report('Killed'))).toContain(
      'expected a CONDITIONALS_BOUNDARY mutant with status Survived',
    )
  })

  it.each([{}, null, { files: [] }, { files: null }])('不正な報告 %j を拒否する', (input) => {
    expect(verifyMutationCanary(input)).toEqual(['mutation report does not contain a files object'])
  })

  it.each([
    { files: {} },
    { files: { 'canary.go': null } },
    { files: { 'canary.go': { mutants: [null, 'Survived'] } } },
    { files: { 'canary.go': { mutants: {} } } },
  ])('判定が欠けた報告 %j を拒否する', (input) => {
    expect(verifyMutationCanary(input)).toHaveLength(2)
  })

  it('ファイルごとに分かれた判定を検算する', () => {
    expect(
      verifyMutationCanary({
        files: {
          'boundary.go': {
            mutants: [{ mutatorName: 'CONDITIONALS_BOUNDARY', status: 'Survived' }],
          },
          'arithmetic.go': {
            mutants: [{ mutatorName: 'ARITHMETIC_BASE', status: 'CompileError' }],
          },
        },
      }),
    ).toEqual([])
  })
})

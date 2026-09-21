import { describe, expect, it } from 'bun:test'
import { pickWorkItemNumber, usedWorkItemNumbers } from './work-item-number.ts'

describe('pickWorkItemNumber', () => {
  it('候補が一つしか残っていない範囲で、その未使用の値を返す', () => {
    const used = new Set([10, 11, 13, 14])

    expect(pickWorkItemNumber(used, () => 0, { min: 10, max: 14 })).toBe(12)
    expect(pickWorkItemNumber(used, () => 0.999, { min: 10, max: 14 })).toBe(12)
  })

  it('空き枠を順に数え、乱数の値に対応する枠を返す', () => {
    const used = new Set([11, 13])
    const range = { min: 10, max: 14 }

    expect(pickWorkItemNumber(used, () => 0, range)).toBe(10)
    expect(pickWorkItemNumber(used, () => 0.5, range)).toBe(12)
    expect(pickWorkItemNumber(used, () => 0.99, range)).toBe(14)
  })

  it('乱数が 1 を返しても範囲の外へ出ない', () => {
    expect(pickWorkItemNumber(new Set(), () => 1, { min: 10, max: 12 })).toBe(12)
  })

  it('範囲が埋まっていたら値を作らず失敗する', () => {
    expect(() => pickWorkItemNumber(new Set([10, 11]), () => 0, { min: 10, max: 11 })).toThrow(
      'no unused work item number remains in 10-11',
    )
  })

  it('既定の範囲は 5 桁に限る', () => {
    expect(pickWorkItemNumber(new Set(), () => 0)).toBe(10_000)
    expect(pickWorkItemNumber(new Set(), () => 1)).toBe(99_999)
  })
})

describe('usedWorkItemNumbers', () => {
  it('ファイル名から番号を読み、桁数を問わず使用済みとして集める', () => {
    expect(
      usedWorkItemNumbers(['wi-1-old.md', 'wi-634-old.md', 'wi-40318-new.md', 'README.md']),
    ).toEqual(new Set([1, 634, 40_318]))
  })
})

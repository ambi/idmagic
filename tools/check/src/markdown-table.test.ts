import { describe, expect, it } from 'bun:test'
import { tableRowCells } from './markdown-table.ts'

describe('tableRowCells', () => {
  it('両端の区切りを落とし、各セルの前後の空白を除く', () => {
    expect(tableRowCells('| a |  b | |')).toEqual(['a', 'b', ''])
  })

  it('両端の区切りを省いた行も同じに読む', () => {
    expect(tableRowCells('  a | b  ')).toEqual(['a', 'b'])
  })

  it('エスケープした区切りをセルの中に残す', () => {
    expect(tableRowCells('| x \\|\\| y | z |')).toEqual(['x \\|\\| y', 'z'])
  })
})

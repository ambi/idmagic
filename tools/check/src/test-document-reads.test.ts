import { describe, expect, it } from 'bun:test'
import { documentPathLiterals } from './test-document-reads.ts'

describe('Go のテストが文書を入力にしないことの検査', () => {
  it('文字列リテラルの文書のパスを行番号とともに返す', () => {
    const source = [
      'package demo_test',
      '',
      'const spec = "../../docs/modules/demo/features/demo.md"',
      'var raw = `docs/development/testing.md`',
    ].join('\n')
    expect(documentPathLiterals(source)).toEqual([
      { line: 3, value: '../../docs/modules/demo/features/demo.md' },
      { line: 4, value: 'docs/development/testing.md' },
    ])
  })

  it('コメント、URL、文書でないパスは返さない', () => {
    const source = [
      'package demo_test',
      '// "シナリオ" (docs/modules/data-keys/) を固定する。',
      '/* docs/modules/demo/ "docs/x.md" */',
      'const url = "https://example.test/docs/guide"',
      'const schema = "../../infra/schema/postgres.sql"',
      "const quote = '\"'",
      'const escaped = "a\\"docs"',
      'const fixture = "testdata/docs/x.json"',
      'const name = "docs"',
    ].join('\n')
    expect(documentPathLiterals(source)).toEqual([])
  })
})

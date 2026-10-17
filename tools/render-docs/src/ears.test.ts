import { describe, expect, it } from 'bun:test'
import { earsCuts } from './ears.ts'

/** 切れ目に従って、名札と本文の並びに直す。表示と同じ分け方を文字列で確かめる。 */
function clauses(statement: string, responder = 'OAuth2'): string[] | undefined {
  const cuts = earsCuts(statement, responder)
  if (!cuts) return undefined
  return cuts.map((cut, index) => {
    const end = cuts[index + 1]?.from ?? statement.length
    return `${cut.kind}: ${statement.slice(cut.to, end)}`
  })
}

describe('earsCuts', () => {
  it('reads an always-on statement as one response', () => {
    expect(clauses('OAuth2 は、同意の作成を提供しない。')).toEqual([
      'then: 同意の作成を提供しない。',
    ])
  })

  it('separates a trigger from the response and drops the marker and the responder', () => {
    expect(clauses('管理者が同意を撤回したとき、OAuth2 は、204 を返す。')).toEqual([
      'when: 管理者が同意を撤回した',
      'then: 204 を返す。',
    ])
  })

  it('reads 場合 as an unwanted input', () => {
    expect(clauses('存在しない同意を受けた場合、OAuth2 は、404 で拒否する。')).toEqual([
      'if: 存在しない同意を受けた',
      'then: 404 で拒否する。',
    ])
  })

  it('keeps the order of a complex preamble', () => {
    expect(
      clauses(
        '`Disabled` のテナントでは、管理者がいる間、再開を要求したとき、Tenancy は、テナントを `Active` にする。',
        'Tenancy',
      ),
    ).toEqual([
      'where: `Disabled` のテナント',
      'while: 管理者がいる',
      'when: 再開を要求した',
      'then: テナントを `Active` にする。',
    ])
  })

  it('reads a state clause that ends with a verb and 間', () => {
    expect(clauses('ジョブが走る間、Jobs は、取り消しを拒否する。', 'Jobs')).toEqual([
      'while: ジョブが走る',
      'then: 取り消しを拒否する。',
    ])
  })

  // 読み取れない文は分けず、表示を原文のままにする。
  it('leaves a statement it cannot read', () => {
    expect(clauses('同意は 365 日で切れる。')).toBeUndefined()
    expect(clauses('OAuth2 は、OAuth2 は、二度名乗る。')).toBeUndefined()
    expect(clauses('前置きに区切りがない、OAuth2 は、返す。')).toBeUndefined()
  })
})

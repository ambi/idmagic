import { describe, expect, it } from 'bun:test'
import { earsCuts } from './ears.ts'

/** 切れ目に従って、名札と本文の並びに直す。表示と同じ分け方を文字列で確かめる。 */
function clauses(statement: string, responder = 'OAuth2'): string[] | undefined {
  const cuts = earsCuts(statement, responder)
  if (!cuts) return undefined
  return cuts.map((cut, index) => {
    const end = cuts[index + 1]?.from ?? statement.length
    return `${cut.keywords.join(' ')}: ${statement.slice(cut.to, end)}`
  })
}

// 型ごとのキーワードは EARS（Alistair Mavin）の型に従う。主体は要件に一つ必要なので残す。
describe('earsCuts', () => {
  it('reads a ubiquitous statement as the system name and its response under Shall', () => {
    expect(clauses('OAuth2 は、同意の作成を提供しない。')).toEqual([
      'shall: OAuth2 は、同意の作成を提供しない。',
    ])
  })

  it('reads an event-driven statement as When and Shall, dropping only the marker', () => {
    expect(clauses('管理者が同意を撤回したとき、OAuth2 は、204 を返す。')).toEqual([
      'when: 管理者が同意を撤回した',
      'shall: OAuth2 は、204 を返す。',
    ])
  })

  it('pairs If with Then before Shall for an unwanted behaviour', () => {
    expect(clauses('存在しない同意を受けた場合、OAuth2 は、404 で拒否する。')).toEqual([
      'if: 存在しない同意を受けた',
      'then shall: OAuth2 は、404 で拒否する。',
    ])
  })

  it('keeps the EARS order of a complex preamble', () => {
    expect(
      clauses(
        '`Disabled` のテナントでは、管理者がいる間、再開を要求したとき、Tenancy は、テナントを `Active` にする。',
        'Tenancy',
      ),
    ).toEqual([
      'where: `Disabled` のテナント',
      'while: 管理者がいる',
      'when: 再開を要求した',
      'shall: Tenancy は、テナントを `Active` にする。',
    ])
  })

  it('reads a state clause that ends with a verb and 間 as While', () => {
    expect(clauses('ジョブが走る間、Jobs は、取り消しを拒否する。', 'Jobs')).toEqual([
      'while: ジョブが走る',
      'shall: Jobs は、取り消しを拒否する。',
    ])
  })

  it('pairs If with Then also behind a state precondition', () => {
    expect(clauses('ジョブが走る間、上限を超えた場合、Jobs は、409 で拒否する。', 'Jobs')).toEqual([
      'while: ジョブが走る',
      'if: 上限を超えた',
      'then shall: Jobs は、409 で拒否する。',
    ])
  })

  // 読み取れない文は分けず、表示を原文のままにする。
  it('leaves a statement it cannot read', () => {
    expect(clauses('同意は 365 日で切れる。')).toBeUndefined()
    expect(clauses('OAuth2 は、OAuth2 は、二度名乗る。')).toBeUndefined()
    expect(clauses('前置きに区切りがない、OAuth2 は、返す。')).toBeUndefined()
  })
})

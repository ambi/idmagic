import { describe, expect, it } from 'bun:test'
import { repositoryChecks, selectChecks } from './registry.ts'

describe('検査 registry', () => {
  it('任意の既存バージョン比較以外の登録規則を集約実行へ一度ずつ接続する', () => {
    expect(selectChecks(['all']).map((check) => check.name)).toEqual(
      repositoryChecks.filter((check) => check.name !== 'api-compat').map((check) => check.name),
    )
    expect(new Set(repositoryChecks.map((check) => check.name)).size).toBe(repositoryChecks.length)
  })

  it('個別検査を同じ registry から選ぶ', () => {
    expect(selectChecks(['work-items']).map((check) => check.name)).toEqual(['work-items'])
    expect(selectChecks(['api-compat']).map((check) => check.name)).toEqual(['api-compat'])
  })
})

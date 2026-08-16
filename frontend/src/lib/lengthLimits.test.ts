import { describe, expect, it } from 'bun:test'
import { LENGTH } from './lengthLimits'

// 入力欄の上限がサーバより緩いと、入力できたのに 422 で弾かれる。この表は
// backend/shared/spec/length.go と同じ数を持たなければならない。
describe('LENGTH', () => {
  it('mirrors the server-side length classes', () => {
    expect(LENGTH).toEqual({
      handle: 64,
      name: 100,
      displayName: 200,
      externalId: 256,
      description: 500,
      uri: 2048,
      expression: 4096,
      plainBody: 8000,
      richBody: 20000,
      email: 254,
      chromeLabel: 80,
      chromeText: 280,
    })
  })
})

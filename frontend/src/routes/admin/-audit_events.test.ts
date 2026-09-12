import { describe, expect, it } from 'bun:test'
import { validateAuditEventsSearch } from './audit_events'

describe('validateAuditEventsSearch', () => {
  it('returns an empty object for an empty search', () => {
    expect(validateAuditEventsSearch({})).toEqual({})
  })

  it('parses known fields from the URL query string (wi-147)', () => {
    expect(
      validateAuditEventsSearch({
        category: 'authentication',
        sub: '00000000-0000-4000-8000-000000000001',
        username: 'alice',
        after: '2026-01-01T00:00:00.000Z',
        before: '2026-01-02T00:00:00.000Z',
        limit: 50,
        filter: ['event.type:eq:UserCreated'],
      }),
    ).toEqual({
      category: 'authentication',
      sub: '00000000-0000-4000-8000-000000000001',
      username: 'alice',
      after: '2026-01-01T00:00:00.000Z',
      before: '2026-01-02T00:00:00.000Z',
      limit: 50,
      filter: ['event.type:eq:UserCreated'],
    })
  })

  //spec:covers EX-SYSTEM-020-02: 古いブックマークに残った横断の状態は検証で落ち、テナント内表示へ戻る。
  // 横断はシステムコンソールの経路が持つ (REQ-SYSTEM-020)。
  it('drops a cross-tenant request left in an old bookmark', () => {
    expect(validateAuditEventsSearch({ allTenants: true })).toEqual({})
    expect(validateAuditEventsSearch({ sub: 'alice', allTenants: true })).toEqual({ sub: 'alice' })
  })

  it('ignores an unknown category rather than throwing', () => {
    expect(validateAuditEventsSearch({ category: 'not-a-real-category' })).toEqual({})
  })

  it('ignores wrong-typed values instead of crashing on a corrupted URL', () => {
    expect(
      validateAuditEventsSearch({
        category: 42,
        sub: 123,
        limit: 'fifty',
        filter: 'not-an-array',
      }),
    ).toEqual({})
  })

  it('drops non-string entries from filter and omits an empty filter array', () => {
    expect(validateAuditEventsSearch({ filter: [1, 2, 3] })).toEqual({})
    expect(validateAuditEventsSearch({ filter: ['actor.username:eq:alice', 7] })).toEqual({
      filter: ['actor.username:eq:alice'],
    })
  })
})

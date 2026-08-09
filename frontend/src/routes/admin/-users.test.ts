import { describe, expect, it } from 'bun:test'
import { adminUsersAPIStatus, validateAdminUsersSearch } from './users'

describe('admin users route search semantics', () => {
  it('uses active as the canonical API default and omits status only for all', () => {
    expect(adminUsersAPIStatus()).toBe('active')
    expect(adminUsersAPIStatus('active')).toBe('active')
    expect(adminUsersAPIStatus('all')).toBeUndefined()
  })

  it('keeps supported URL filters and ignores unsupported values', () => {
    expect(validateAdminUsersSearch({ query: 'alice', status: 'all', cursor: 'next' })).toEqual({
      query: 'alice',
      status: 'all',
      cursor: 'next',
    })
    expect(validateAdminUsersSearch({ status: 'unknown', cursor: 42 })).toEqual({})
  })
})

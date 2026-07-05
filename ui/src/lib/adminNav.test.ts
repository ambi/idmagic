import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { adminNavItems } from './adminNav'

describe('adminNavItems', () => {
  const originalLocation = window.location

  beforeEach(() => {
    vi.stubGlobal('location', {
      ...originalLocation,
      pathname: '/realms/test-tenant',
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('should return all admin nav items with specified item active', () => {
    const items = adminNavItems('users')
    expect(items.length).toBeGreaterThan(5)

    const dashboard = items.find((i) => i.key === 'dashboard')
    const users = items.find((i) => i.key === 'users')
    const settings = items.find((i) => i.key === 'settings')

    expect(dashboard?.active).toBe(false)
    expect(users?.active).toBe(true)
    expect(settings?.active).toBe(false)

    expect(users?.href).toBe('/realms/test-tenant/admin/users')
  })
})

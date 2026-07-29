import { describe, it, expect } from 'bun:test'
import { systemNavItems } from './systemNav'

describe('systemNavItems', () => {
  it('should return system nav items with tenants active', () => {
    const items = systemNavItems('tenants')
    expect(items).toHaveLength(3)
    expect(items[0].key).toBe('tenants')
    expect(items[0].active).toBe(true)
    expect(items[1].key).toBe('key-health')
    expect(items[1].active).toBe(false)
    expect(items[2].key).toBe('data-key-health')
    expect(items[2].active).toBe(false)
  })

  it('should return system nav items with key-health active', () => {
    const items = systemNavItems('key-health')
    expect(items).toHaveLength(3)
    expect(items[0].key).toBe('tenants')
    expect(items[0].active).toBe(false)
    expect(items[1].key).toBe('key-health')
    expect(items[1].active).toBe(true)
    expect(items[2].key).toBe('data-key-health')
    expect(items[2].active).toBe(false)
  })

  it('should return system nav items with data-key-health active', () => {
    const items = systemNavItems('data-key-health')
    expect(items).toHaveLength(3)
    expect(items[0].active).toBe(false)
    expect(items[1].active).toBe(false)
    expect(items[2].key).toBe('data-key-health')
    expect(items[2].active).toBe(true)
  })
})

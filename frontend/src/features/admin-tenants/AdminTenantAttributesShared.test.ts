import { describe, expect, it } from 'bun:test'
import { newAttribute, normalizeAttribute } from './AdminTenantAttributesShared'

describe('normalizeAttribute', () => {
  it('keeps unique only for a single-valued string attribute', () => {
    const draft = { ...newAttribute(), key: ' badge ', unique: true }
    expect(normalizeAttribute(draft).unique).toBe(true)
    expect(normalizeAttribute({ ...draft, type: 'string_array' }).unique).toBe(false)
    expect(normalizeAttribute({ ...draft, type: 'number' }).unique).toBe(false)
    expect(normalizeAttribute({ ...draft, unique: undefined }).unique).toBe(false)
  })
})

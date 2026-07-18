import { describe, expect, it } from 'vitest'
import { domainLabelsDictionary } from '../../lib/i18n/domainLabels.i18n'
import type { AttributeValue, UserAttributeDef } from '../../types'
import { attributeValueToText, groupedAttributeDefs } from './AdminUsersShared'

const tLabels = domainLabelsDictionary.en

function def(overrides: Partial<UserAttributeDef> = {}): UserAttributeDef {
  return {
    key: 'custom_field',
    type: 'string',
    multi_valued: false,
    required: false,
    editable_by_user: true,
    visibility: 'admin_readable',
    pii: false,
    ...overrides,
  }
}

describe('attributeValueToText', () => {
  it('renders each attribute type as its display text', () => {
    expect(attributeValueToText({ type: 'string', string: 'hi' } as AttributeValue)).toBe('hi')
    expect(attributeValueToText({ type: 'date', date: '2026-01-01' } as AttributeValue)).toBe(
      '2026-01-01',
    )
    expect(attributeValueToText({ type: 'number', number: 42 } as AttributeValue)).toBe('42')
    expect(attributeValueToText({ type: 'boolean', boolean: true } as AttributeValue)).toBe('true')
    expect(attributeValueToText({ type: 'boolean', boolean: false } as AttributeValue)).toBe(
      'false',
    )
    expect(
      attributeValueToText({ type: 'string_array', string_array: ['a', 'b'] } as AttributeValue),
    ).toBe('a, b')
  })

  it('falls back to an empty string when the value is unset', () => {
    expect(attributeValueToText({ type: 'string' } as AttributeValue)).toBe('')
    expect(attributeValueToText({ type: 'string_array' } as AttributeValue)).toBe('')
  })
})

describe('groupedAttributeDefs', () => {
  it('buckets defs into profile / organization / custom and preserves that order', () => {
    const defs = [
      def({ key: 'department' }),
      def({ key: 'email', oidc_scope: 'email' }),
      def({ key: 'favorite_color' }),
    ]
    const groups = groupedAttributeDefs(defs, tLabels)
    expect(groups.map((g) => g.key)).toEqual(['profile', 'organization', 'custom'])
    expect(groups.find((g) => g.key === 'profile')?.defs.map((d) => d.key)).toEqual(['email'])
    expect(groups.find((g) => g.key === 'organization')?.defs.map((d) => d.key)).toEqual([
      'department',
    ])
    expect(groups.find((g) => g.key === 'custom')?.defs.map((d) => d.key)).toEqual([
      'favorite_color',
    ])
  })

  it('omits groups with no members', () => {
    const groups = groupedAttributeDefs([def({ key: 'favorite_color' })], tLabels)
    expect(groups).toHaveLength(1)
    expect(groups[0].key).toBe('custom')
    expect(groups[0].title).toBe(tLabels.attributeGroupCustom)
  })

  it('returns an empty list for no defs', () => {
    expect(groupedAttributeDefs([], tLabels)).toEqual([])
  })
})

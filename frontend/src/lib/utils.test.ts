import { describe, it, expect } from 'bun:test'
import { domainLabelsDictionary } from './i18n/domainLabels.i18n'
import { attributeLabel, attributeGroupKey, attributeGroupTitle, cn } from './utils'

describe('cn', () => {
  it('should merge tailwind classes correctly', () => {
    expect(cn('bg-red-500', 'text-white')).toBe('bg-red-500 text-white')
    expect(cn('bg-red-500 bg-blue-500')).toBe('bg-blue-500')
  })

  it('should drop falsy inputs that callers pass for unset or disabled classes', () => {
    const disabled = false
    const className: string | undefined = undefined
    expect(cn('px-2', disabled && 'opacity-50', className, null, '')).toBe('px-2')
    expect(cn('px-2', !disabled && 'opacity-50', disabled ? 'w-4' : 'w-8')).toBe(
      'px-2 opacity-50 w-8',
    )
  })

  it('should let a caller className override the default classes', () => {
    expect(cn('rounded-md px-3 py-2', 'px-4')).toBe('rounded-md py-2 px-4')
  })
})

describe('attributeLabel', () => {
  it('should format with label if present', () => {
    expect(attributeLabel({ key: 'email', label: 'メールアドレス' })).toBe('メールアドレス (email)')
  })

  it('should format with key only if label is missing', () => {
    expect(attributeLabel({ key: 'email', label: '' })).toBe('email')
    expect(attributeLabel({ key: 'email' })).toBe('email')
  })
})

describe('attributeGroupKey', () => {
  it('should return profile for OIDC scope defs', () => {
    expect(attributeGroupKey({ key: 'given_name', oidc_scope: 'profile' })).toBe('profile')
  })

  it('should return organization for organization-related keys', () => {
    expect(attributeGroupKey({ key: 'organization' })).toBe('organization')
    expect(attributeGroupKey({ key: 'employee_id' })).toBe('organization')
  })

  it('should return custom for other keys', () => {
    expect(attributeGroupKey({ key: 'my_custom_attribute' })).toBe('custom')
  })
})

describe('attributeGroupTitle', () => {
  it('should return correct Japanese titles', () => {
    const ja = domainLabelsDictionary.ja
    expect(attributeGroupTitle('profile', ja)).toBe('OIDC 標準クレーム')
    expect(attributeGroupTitle('organization', ja)).toBe('組織情報')
    expect(attributeGroupTitle('custom', ja)).toBe('カスタム属性')
  })

  it('should return correct English titles', () => {
    const en = domainLabelsDictionary.en
    expect(attributeGroupTitle('profile', en)).toBe('OIDC standard claims')
    expect(attributeGroupTitle('organization', en)).toBe('Organization info')
    expect(attributeGroupTitle('custom', en)).toBe('Custom attributes')
  })
})

import { describe, it, expect } from 'vitest'
import type { UserAttributeDef } from '../../types'
import {
  buildDynamicRuleExpression,
  builderAttributeOptions,
  isConditionComplete,
  operatorIdsForType,
} from './dynamicRuleCel'

const customAttr = (key: string, type: UserAttributeDef['type']): UserAttributeDef => ({
  key,
  type,
  multi_valued: false,
  required: false,
  editable_by_user: false,
  visibility: 'private',
  pii: false,
})

// scenario: AdminGroups.update_dynamic_rule / preview_dynamic_rule (builder が生成する CEL)
describe('buildDynamicRuleExpression', () => {
  it('generates a string equality expression matching hand-written CEL', () => {
    expect(
      buildDynamicRuleExpression([
        { attribute: 'preferred_username', operator: 'equals', value: 'alice' },
      ]),
    ).toBe('user.preferred_username == "alice"')
  })

  it('joins multiple conditions with AND', () => {
    expect(
      buildDynamicRuleExpression([
        { attribute: 'department', operator: 'equals', value: 'Engineering' },
        { attribute: 'email_verified', operator: 'isTrue', value: '' },
      ]),
    ).toBe('user.department == "Engineering" && user.email_verified == true')
  })

  it('generates string method calls for contains/startsWith/endsWith', () => {
    expect(
      buildDynamicRuleExpression([
        { attribute: 'email', operator: 'endsWith', value: '@example.com' },
      ]),
    ).toBe('user.email.endsWith("@example.com")')
  })

  it('converts date operators to timestamp() calls', () => {
    expect(
      buildDynamicRuleExpression([
        { attribute: 'hire_date', operator: 'before', value: '2026-01-01' },
      ]),
    ).toBe('user.hire_date < timestamp("2026-01-01T00:00:00Z")')
  })

  it('escapes quotes and backslashes in string values', () => {
    expect(
      buildDynamicRuleExpression([{ attribute: 'name', operator: 'equals', value: 'a"b\\c' }]),
    ).toBe('user.name == "a\\"b\\\\c"')
  })

  it('skips incomplete conditions', () => {
    expect(
      buildDynamicRuleExpression([
        { attribute: 'preferred_username', operator: 'equals', value: 'alice' },
        { attribute: '', operator: '', value: '' },
      ]),
    ).toBe('user.preferred_username == "alice"')
  })
})

describe('operatorIdsForType', () => {
  it('limits operators by attribute type', () => {
    expect(operatorIdsForType('string')).toEqual([
      'equals',
      'notEquals',
      'contains',
      'startsWith',
      'endsWith',
    ])
    expect(operatorIdsForType('boolean')).toEqual(['isTrue', 'isFalse'])
    expect(operatorIdsForType('date')).toEqual(['before', 'after', 'on'])
    expect(operatorIdsForType('number')).toEqual([])
  })
})

describe('isConditionComplete', () => {
  it('requires a value for value operators but not for boolean operators', () => {
    expect(isConditionComplete({ attribute: 'email', operator: 'equals', value: '' })).toBe(false)
    expect(isConditionComplete({ attribute: 'email', operator: 'equals', value: 'x' })).toBe(true)
    expect(
      isConditionComplete({ attribute: 'email_verified', operator: 'isTrue', value: '' }),
    ).toBe(true)
  })
})

describe('builderAttributeOptions', () => {
  it('includes built-ins and only builder-supported custom attribute types', () => {
    const options = builderAttributeOptions([
      customAttr('department', 'string'),
      customAttr('hire_date', 'date'),
      customAttr('level', 'number'),
      customAttr('tags', 'string_array'),
    ])
    const keys = options.map((o) => o.key)
    expect(keys).toContain('preferred_username')
    expect(keys).toContain('email_verified')
    expect(keys).toContain('department')
    expect(keys).toContain('hire_date')
    expect(keys).not.toContain('level')
    expect(keys).not.toContain('tags')
  })
})

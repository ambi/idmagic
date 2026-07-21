import type { AttributeType, UserAttributeDef } from '../../types'

// A single builder row: pick an attribute, an operator, and (for most operators) a value.
export type BuilderCondition = {
  attribute: string
  operator: string
  value: string
}

export type AttributeOption = {
  key: string
  type: AttributeType
}

// Built-in attributes referenceable in dynamic-group CEL. Mirrors the backend
// dynamicRuleDefinitions in idmanagement/group/domain/dynamic_group_rule.go.
export const BUILTIN_DYNAMIC_RULE_ATTRIBUTES: AttributeOption[] = [
  { key: 'id', type: 'string' },
  { key: 'preferred_username', type: 'string' },
  { key: 'name', type: 'string' },
  { key: 'given_name', type: 'string' },
  { key: 'family_name', type: 'string' },
  { key: 'email', type: 'string' },
  { key: 'email_verified', type: 'boolean' },
]

// The builder covers the common single-condition types. Number and string_array
// attributes stay in the advanced (raw CEL) editor and are omitted here.
const SUPPORTED_TYPES: AttributeType[] = ['string', 'boolean', 'date']

const OPERATORS_BY_TYPE: Partial<Record<AttributeType, string[]>> = {
  string: ['equals', 'notEquals', 'contains', 'startsWith', 'endsWith'],
  boolean: ['isTrue', 'isFalse'],
  date: ['before', 'after', 'on'],
}

const VALUELESS_OPERATORS = new Set(['isTrue', 'isFalse'])

export function operatorIdsForType(type: AttributeType): string[] {
  return OPERATORS_BY_TYPE[type] ?? []
}

export function isBuilderSupportedType(type: AttributeType): boolean {
  return SUPPORTED_TYPES.includes(type)
}

export function operatorNeedsValue(operator: string): boolean {
  return !VALUELESS_OPERATORS.has(operator)
}

// Built-in attributes plus the tenant's custom attributes, filtered to the types
// the builder can construct conditions for.
export function builderAttributeOptions(customAttributes: UserAttributeDef[]): AttributeOption[] {
  const custom = customAttributes
    .filter((def) => isBuilderSupportedType(def.type))
    .map((def) => ({ key: def.key, type: def.type }))
  return [...BUILTIN_DYNAMIC_RULE_ATTRIBUTES, ...custom]
}

export function isConditionComplete(condition: BuilderCondition): boolean {
  if (!condition.attribute || !condition.operator) return false
  if (operatorNeedsValue(condition.operator) && condition.value.trim() === '') return false
  return true
}

function celString(value: string): string {
  return `"${value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
}

// <input type="date"> yields YYYY-MM-DD; the backend parses date attributes to
// midnight UTC, so build a matching RFC3339 timestamp literal.
function celTimestamp(dateValue: string): string {
  return `timestamp("${dateValue}T00:00:00Z")`
}

function celForCondition(condition: BuilderCondition): string | null {
  const ref = `user.${condition.attribute}`
  const value = condition.value.trim()
  switch (condition.operator) {
    case 'equals':
      return `${ref} == ${celString(value)}`
    case 'notEquals':
      return `${ref} != ${celString(value)}`
    case 'contains':
      return `${ref}.contains(${celString(value)})`
    case 'startsWith':
      return `${ref}.startsWith(${celString(value)})`
    case 'endsWith':
      return `${ref}.endsWith(${celString(value)})`
    case 'isTrue':
      return `${ref} == true`
    case 'isFalse':
      return `${ref} == false`
    case 'before':
      return `${ref} < ${celTimestamp(value)}`
    case 'after':
      return `${ref} > ${celTimestamp(value)}`
    case 'on':
      return `${ref} == ${celTimestamp(value)}`
    default:
      return null
  }
}

// Build a single boolean CEL expression from AND-joined builder conditions.
// Incomplete conditions are skipped so a half-filled row does not break preview.
export function buildDynamicRuleExpression(conditions: BuilderCondition[]): string {
  const parts: string[] = []
  for (const condition of conditions) {
    if (!isConditionComplete(condition)) continue
    const snippet = celForCondition(condition)
    if (snippet) parts.push(snippet)
  }
  return parts.join(' && ')
}

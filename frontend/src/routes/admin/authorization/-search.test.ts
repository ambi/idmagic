import { describe, expect, it } from 'bun:test'
import { validateCheckSearch } from './check'
import { validateModelSearch } from './model'
import { validateTupleSearch } from './relation-tuples'

describe('authorization search parameters', () => {
  it('accepts only a positive integer version', () => {
    expect(validateModelSearch({ version: 2 })).toEqual({ version: 2 })
    for (const version of [0, -1, 1.5, 'x']) {
      expect(validateModelSearch({ version })).toEqual({})
    }
  })

  it('keeps filter strings, restores numeric identifiers, and drops the rest', () => {
    expect(
      validateTupleSearch({ resource_type: 'doc', resource_id: 42, relation: '', other: 'x' }),
    ).toEqual({ resource_type: 'doc', resource_id: '42' })
  })

  it('keeps only a filled consistency token', () => {
    expect(validateCheckSearch({ consistency: 'c' })).toEqual({ consistency: 'c' })
    expect(validateCheckSearch({ consistency: '' })).toEqual({})
  })
})

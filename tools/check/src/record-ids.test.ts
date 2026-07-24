import { describe, expect, it } from 'bun:test'
import { adrRef, findDuplicates, workItemRef } from './record-ids.ts'

const NS = '/repo/work-items'
const ADR_NS = '/repo/decisions'

describe('workItemRef', () => {
  it('derives the id from the filename stem', () => {
    const { ref, findings } = workItemRef(`${NS}/wi-38-foo.md`, NS)
    expect(findings).toEqual([])
    expect(ref?.id).toBe('wi-38-foo')
  })

  it('derives the id from the filename stem regardless of a legacy context prefix', () => {
    const { ref, findings } = workItemRef(`${NS}/idp-wi-38-foo.md`, NS)
    expect(findings).toEqual([])
    expect(ref?.id).toBe('idp-wi-38-foo')
  })
})

describe('adrRef', () => {
  it('derives adr-<NNN> from an un-prefixed filename', () => {
    const { ref, findings } = adrRef(`${ADR_NS}/ADR-024-durable-keys.md`, ADR_NS)
    expect(findings).toEqual([])
    expect(ref?.id).toBe('adr-024')
  })

  it('derives adr-<NNN> from a legacy prefixed filename, ignoring the prefix', () => {
    const { ref, findings } = adrRef(`${ADR_NS}/idp-ADR-024-durable-keys.md`, ADR_NS)
    expect(findings).toEqual([])
    expect(ref?.id).toBe('adr-024')
  })

  it('flags a filename that is not ADR-NNN-<title>.md', () => {
    const { ref, findings } = adrRef(`${ADR_NS}/README.md`, ADR_NS)
    expect(ref).toBeUndefined()
    expect(findings[0]?.message).toContain('is not')
  })
})

describe('findDuplicates', () => {
  it('reports a repeated id within the same namespace', () => {
    const dups = findDuplicates([
      { path: '/a/wi-23-a.md', namespace: '/a', id: 'wi-23-a' },
      { path: '/a/wi-23-b.md', namespace: '/a', id: 'wi-23-a' },
    ])
    expect(dups).toHaveLength(1)
    expect(dups[0]?.path).toBe('/a/wi-23-b.md')
  })

  it('does not report the same id in different namespaces', () => {
    const dups = findDuplicates([
      { path: '/a/wi-23.md', namespace: '/a', id: 'wi-23' },
      { path: '/b/wi-23.md', namespace: '/b', id: 'wi-23' },
    ])
    expect(dups).toEqual([])
  })
})

import { describe, expect, it } from 'bun:test'
import { findDuplicates, workItemRef } from './record-ids.ts'

const NS = '/repo/work-items'

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

import { describe, expect, it } from 'bun:test'
import { adrRef, findDuplicates, workItemRef } from './record-ids.ts'

const NS = '/repo/work-items'
const ADR_NS = '/repo/decisions'

describe('workItemRef', () => {
  it('accepts an id equal to the filename stem', () => {
    const { ref, findings } = workItemRef(`${NS}/repo-wi-38-foo.md`, NS, { id: 'repo-wi-38-foo' })
    expect(findings).toEqual([])
    expect(ref?.id).toBe('repo-wi-38-foo')
  })

  it('flags an id that does not match the filename stem', () => {
    const { ref, findings } = workItemRef(`${NS}/repo-wi-38-foo.md`, NS, { id: 'repo-wi-39-foo' })
    expect(ref?.id).toBe('repo-wi-39-foo')
    expect(findings[0]?.message).toContain('does not match filename stem')
  })

  it('flags a missing id and yields no ref', () => {
    const { ref, findings } = workItemRef(`${NS}/repo-wi-38-foo.md`, NS, { title: 'x' })
    expect(ref).toBeUndefined()
    expect(findings[0]?.message).toContain('no string `id`')
  })
})

describe('adrRef', () => {
  it('derives <prefix>-adr-<NNN> from a prefixed filename', () => {
    const { ref, findings } = adrRef(`${ADR_NS}/idp-ADR-024-durable-keys.md`, ADR_NS)
    expect(findings).toEqual([])
    expect(ref?.id).toBe('idp-adr-024')
  })

  it('flags an un-prefixed (legacy) ADR filename with a hint', () => {
    const { ref, findings } = adrRef(`${ADR_NS}/ADR-024-durable-keys.md`, ADR_NS)
    expect(ref).toBeUndefined()
    expect(findings[0]?.message).toContain('add a context prefix')
  })
})

describe('findDuplicates', () => {
  it('reports a repeated id within the same namespace', () => {
    const dups = findDuplicates([
      { path: '/a/idp-wi-23-a.md', namespace: '/a', id: 'idp-wi-23-a' },
      { path: '/a/idp-wi-23-b.md', namespace: '/a', id: 'idp-wi-23-a' },
    ])
    expect(dups).toHaveLength(1)
    expect(dups[0]?.path).toBe('/a/idp-wi-23-b.md')
  })

  it('does not report the same id in different namespaces', () => {
    const dups = findDuplicates([
      { path: '/a/idp-wi-23.md', namespace: '/a', id: 'idp-wi-23' },
      { path: '/b/repo-wi-23.md', namespace: '/b', id: 'repo-wi-23' },
    ])
    expect(dups).toEqual([])
  })
})

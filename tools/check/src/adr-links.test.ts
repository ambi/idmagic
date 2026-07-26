import { describe, expect, it } from 'bun:test'
import { type AdrLinkSource, checkAdrLinks, extractAdrLinks } from './adr-links.ts'

const resolvesAll = () => true
const resolvesNone = () => false

describe('extractAdrLinks', () => {
  it('finds a workspace-relative markdown link', () => {
    expect(extractAdrLinks('see [ADR-092](decisions/ADR-092-top-level-dirs.md) for this')).toEqual([
      'decisions/ADR-092-top-level-dirs.md',
    ])
  })

  it('finds a link relative to the referencing file', () => {
    expect(extractAdrLinks('[ADR-129](../../decisions/ADR-129-job-execution-lanes.md)')).toEqual([
      '../../decisions/ADR-129-job-execution-lanes.md',
    ])
  })

  it('ignores a prose glob, which is not a link', () => {
    expect(extractAdrLinks('files named decisions/ADR-092-*.md move together')).toEqual([])
  })

  it('ignores a bare id with no path', () => {
    expect(extractAdrLinks('recorded in ADR-103 and ADR-114')).toEqual([])
  })

  it('deduplicates and sorts repeated references', () => {
    const text = '[a](decisions/ADR-2-b.md) [b](decisions/ADR-1-a.md) [c](decisions/ADR-2-b.md)'
    expect(extractAdrLinks(text)).toEqual(['decisions/ADR-1-a.md', 'decisions/ADR-2-b.md'])
  })
})

describe('checkAdrLinks', () => {
  const source: AdrLinkSource = {
    path: '/repo/ARCHITECTURE.md',
    text: 'see [ADR-1](decisions/ADR-1-a.md)',
  }

  it('reports nothing when every reference resolves', () => {
    expect(checkAdrLinks([source], resolvesAll)).toEqual([])
  })

  it('reports a reference that resolves to no file', () => {
    expect(checkAdrLinks([source], resolvesNone)).toEqual([
      {
        path: '/repo/ARCHITECTURE.md',
        message: "dangling ADR link 'decisions/ADR-1-a.md' resolves to no file",
      },
    ])
  })

  it('checks declared references that never appear in the text', () => {
    const declaredOnly: AdrLinkSource = {
      path: '/repo/work-items/wi-1.md',
      text: 'no links here',
      declared: ['decisions/ADR-33-tenant-resolution.md'],
    }
    expect(checkAdrLinks([declaredOnly], resolvesNone)).toHaveLength(1)
  })

  it('passes the referencing path to the resolver so relative links work', () => {
    const seen: Array<[string, string]> = []
    checkAdrLinks([source], (from, reference) => {
      seen.push([from, reference])
      return true
    })
    expect(seen).toEqual([['/repo/ARCHITECTURE.md', 'decisions/ADR-1-a.md']])
  })
})

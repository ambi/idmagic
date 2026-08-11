import { describe, expect, it } from 'bun:test'
import { renderSpecificationSite } from './render.ts'

const document = {
  path: 'spec/SPECIFICATION.md',
  source: `---
context: repository
updated_at: 2026-08-11
---

# Repository Specification

## Overview

The overview.
`,
}

describe('renderSpecificationSite', () => {
  it('renders document navigation and tagged API operations', () => {
    const result = renderSpecificationSite({
      documents: [document],
      repositoryRoot: '/repo',
      output: '/repo/spec/generated/docs/index.html',
      openapi: {
        paths: {
          '/things': {
            get: { operationId: 'ListThings', tags: ['Things'] },
          },
        },
      },
    })
    expect(result.operations).toBe(1)
    expect(result.tags).toEqual(['Things'])
    expect(result.html).toContain('href="#repository-overview"')
    expect(result.html).toContain('id="api-things"')
  })

  it('rejects unclassified operations', () => {
    expect(() =>
      renderSpecificationSite({
        documents: [document],
        repositoryRoot: '/repo',
        output: '/repo/spec/generated/docs/index.html',
        openapi: { paths: { '/things': { get: { operationId: 'ListThings' } } } },
      }),
    ).toThrow('has no owning context tag')
  })
})

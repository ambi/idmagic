import { describe, expect, it } from 'bun:test'
import {
  type ArchCheckOptions,
  collectSclElements,
  parseArchitectureDoc,
  verifyArchitecture,
} from './arch-check.ts'

const opts = (over: Partial<ArchCheckOptions> = {}): ArchCheckOptions => ({
  archDir: '/ws',
  workspaceRoot: '/ws',
  sclElements: new Set<string>(),
  pathExists: () => true,
  join: (...parts) => parts.join('/'),
  ...over,
})

describe('parseArchitectureDoc', () => {
  it('parses nested frontmatter and H2 body sections into one object', () => {
    const text = `---
context: repo
updated_at: 2026-07-06
modules:
  ra:
    path: tools/ra
    responsibility: "cli"
    realizes: [interfaces.DiscoverWorkspace]
---

# Architecture: repo

## Overview
the map

## Structure
tree

## Structural Decisions
see repo-ADR-001
`
    const data = parseArchitectureDoc(text) as Record<string, unknown>
    expect(data.context).toBe('repo')
    expect((data.modules as Record<string, { path: string }>).ra?.path).toBe('tools/ra')
    expect(data.overview).toBe('the map')
    expect(data.structure).toBe('tree')
    expect(data.structural_decisions).toBe('see repo-ADR-001')
  })
})

describe('collectSclElements', () => {
  it('collects section.Name refs from a parsed SCL doc', () => {
    const refs = collectSclElements({
      interfaces: { DiscoverWorkspace: {} },
      models: { WorkspaceApp: {} },
      authorization: {
        resources: { System: {} },
        principals: { Maintainer: {} },
        policies: { CanRender: {} },
      },
      annotations: { ignored: {} },
    })
    expect(refs.sort()).toEqual([
      'authorization.CanRender',
      'authorization.Maintainer',
      'authorization.System',
      'interfaces.DiscoverWorkspace',
      'models.WorkspaceApp',
    ])
  })
})

describe('verifyArchitecture', () => {
  it('accepts a map whose paths, realizes and contexts all resolve', () => {
    const doc = {
      context: 'repo',
      modules: { ra: { path: 'tools/ra', realizes: ['interfaces.DiscoverWorkspace'] } },
      contexts: { repo: { root: '.' } },
    }
    const report = verifyArchitecture(
      doc,
      opts({ sclElements: new Set(['interfaces.DiscoverWorkspace']) }),
    )
    expect(report.errors).toEqual([])
  })

  it('flags a module path that does not exist', () => {
    const doc = { modules: { ra: { path: 'tools/gone' } } }
    const report = verifyArchitecture(doc, opts({ pathExists: () => false }))
    expect(report.errors).toHaveLength(1)
    expect(report.errors[0]?.message).toContain("path 'tools/gone' does not exist")
  })

  it('flags a realizes ref that no SCL element defines', () => {
    const doc = { modules: { ra: { path: 'tools/ra', realizes: ['interfaces.Ghost'] } } }
    const report = verifyArchitecture(doc, opts())
    expect(report.errors).toHaveLength(1)
    expect(report.errors[0]?.message).toContain("realizes 'interfaces.Ghost'")
  })

  it('flags a context that is not among the declared contexts', () => {
    const doc = { context: 'idp', contexts: { repo: { root: '.' } } }
    const report = verifyArchitecture(doc, opts())
    expect(report.errors.some((e) => e.message.includes("context 'idp' is not among"))).toBe(true)
  })

  it('flags a context root that does not exist', () => {
    const doc = { context: 'repo', contexts: { repo: { root: 'nowhere' } } }
    const report = verifyArchitecture(doc, opts({ pathExists: () => false }))
    expect(report.errors.some((e) => e.message.includes("root 'nowhere' does not exist"))).toBe(
      true,
    )
  })
})

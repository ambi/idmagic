import { describe, expect, it } from 'bun:test'
import {
  type ArchCheckOptions,
  collectSclElements,
  parseArchitectureDoc,
  verifyArchitecture,
} from './arch-check.ts'
import type { SclWorkspaceIndex } from './scl-element-reference.ts'

const sclIndex: SclWorkspaceIndex = {
  root: {},
  contexts: new Map([
    ['Identity', { context: 'Identity', interfaces: { ResolveUser: {} }, models: { User: {} } }],
    ['Audit', { context: 'Audit', interfaces: { WriteEvent: {} } }],
  ]),
}

const opts = (over: Partial<ArchCheckOptions> = {}): ArchCheckOptions => ({
  archDir: '/ws',
  workspaceRoot: '/ws',
  sclIndex,
  expectedContexts: new Map([
    ['Identity', 'spec/contexts/identity.yaml'],
    ['Audit', 'spec/contexts/audit.yaml'],
  ]),
  pathExists: () => true,
  join: (...parts) => parts.join('/'),
  ...over,
})

const validDoc = () => ({
  context: 'repo',
  contexts: {
    Identity: { spec: 'spec/contexts/identity.yaml', summary: 'identity' },
    Audit: { spec: 'spec/contexts/audit.yaml', summary: 'audit' },
  },
  modules: {
    identity_domain: {
      path: 'backend/identity/domain',
      context: 'Identity',
      layer: 'domain',
      role: 'implementation',
      realizes: [{ context: 'Identity', kind: 'model', element: 'User' }],
      depends_on: [] as { module: string; via: string }[],
    },
    identity_api: {
      path: 'backend/identity/api',
      context: 'Identity',
      layer: 'adapters',
      role: 'published_interface',
      realizes: [{ context: 'Identity', kind: 'interface', element: 'ResolveUser' }],
      depends_on: [{ module: 'identity_domain', via: 'published_interface' }],
    },
    audit_adapter: {
      path: 'backend/audit/adapter',
      context: 'Audit',
      layer: 'infrastructure',
      role: 'implementation',
      realizes: [{ context: 'Audit', kind: 'interface', element: 'WriteEvent' }],
      depends_on: [{ module: 'identity_api', via: 'published_interface' }],
    },
  },
  runtime_units: {
    api: {
      kind: 'api',
      entrypoint: 'backend/cmd/api/main.go',
      modules: ['identity_api', 'audit_adapter'],
    },
  },
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
  it('accepts a complete executable map', () => {
    expect(verifyArchitecture(validDoc(), opts())).toEqual({ errors: [], warnings: [] })
  })

  it('requires an exact context-to-spec projection and existing spec paths', () => {
    const doc = validDoc()
    doc.contexts.Identity.spec = 'spec/wrong.yaml'
    delete (doc.contexts as Record<string, unknown>).Audit
    ;(doc.contexts as Record<string, unknown>).Ghost = { spec: 'spec/ghost.yaml' }
    const report = verifyArchitecture(
      doc,
      opts({ pathExists: (path) => !path.endsWith('ghost.yaml') }),
    )
    expect(report.errors.map((error) => error.message)).toEqual(
      expect.arrayContaining([
        expect.stringContaining("context 'Identity' spec must be 'spec/contexts/identity.yaml'"),
        expect.stringContaining("SCL context 'Audit' is missing"),
        expect.stringContaining("context 'Ghost' is not declared"),
        expect.stringContaining("spec 'spec/ghost.yaml' does not exist"),
      ]),
    )
  })

  it('checks module path, context, layer and role', () => {
    const doc = validDoc()
    Object.assign(doc.modules.identity_domain, {
      path: 'backend/missing',
      context: 'Missing',
      layer: 'presentation',
      role: 'helper',
    })
    const report = verifyArchitecture(
      doc,
      opts({ pathExists: (path) => !path.endsWith('backend/missing') }),
    )
    expect(report.errors.map((error) => error.message)).toEqual(
      expect.arrayContaining([
        expect.stringContaining("path 'backend/missing' does not exist"),
        expect.stringContaining("context 'Missing' is not declared"),
        expect.stringContaining("invalid layer 'presentation'"),
        expect.stringContaining("invalid role 'helper'"),
      ]),
    )
  })

  it('resolves direct SCL references and enforces context locality', () => {
    const doc = validDoc()
    doc.modules.identity_domain.realizes = [{ context: 'Audit', kind: 'model', element: 'Missing' }]
    const messages = verifyArchitecture(doc, opts()).errors.map((error) => error.message)
    expect(messages).toEqual(
      expect.arrayContaining([
        expect.stringContaining("cannot realize context 'Audit'"),
        expect.stringContaining("has no model 'Missing'"),
      ]),
    )
  })

  it('flags unknown dependency targets and outer-layer dependencies', () => {
    const doc = validDoc()
    doc.modules.identity_domain.depends_on = [
      { module: 'identity_api', via: 'published_interface' },
      { module: 'missing', via: 'published_interface' },
    ]
    const messages = verifyArchitecture(doc, opts()).errors.map((error) => error.message)
    expect(messages).toEqual(
      expect.arrayContaining([
        expect.stringContaining("layer 'domain' cannot depend on outer layer 'adapters'"),
        expect.stringContaining("depends on unknown module 'missing'"),
      ]),
    )
  })

  it('flags dependency cycles', () => {
    const doc = validDoc()
    doc.modules.identity_domain.depends_on = [
      { module: 'identity_api', via: 'published_interface' },
    ]
    expect(
      verifyArchitecture(doc, opts()).errors.some((error) =>
        error.message.includes('module dependency cycle'),
      ),
    ).toBe(true)
  })

  it('requires role-compatible via for cross-context dependencies', () => {
    const doc = validDoc()
    doc.modules.audit_adapter.depends_on = [{ module: 'identity_domain', via: 'binding' }]
    const messages = verifyArchitecture(doc, opts()).errors.map((error) => error.message)
    expect(messages.some((message) => message.includes("via 'binding' is incompatible"))).toBe(true)
  })

  it('checks runtime entrypoint existence and composed module references', () => {
    const doc = validDoc()
    doc.runtime_units.api.entrypoint = 'backend/cmd/missing/main.go'
    doc.runtime_units.api.modules = ['identity_api', 'missing']
    const report = verifyArchitecture(
      doc,
      opts({ pathExists: (path) => !path.includes('/cmd/missing/') }),
    )
    expect(report.errors.map((error) => error.message)).toEqual(
      expect.arrayContaining([
        expect.stringContaining("entrypoint 'backend/cmd/missing/main.go' does not exist"),
        expect.stringContaining("references unknown module 'missing'"),
      ]),
    )
  })
})

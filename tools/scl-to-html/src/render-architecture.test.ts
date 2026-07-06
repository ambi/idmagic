import { describe, expect, it } from 'bun:test'
import { renderPage } from './page.ts'
import { architectureTocItems, renderArchitectureTab } from './render-architecture.ts'
import type { ArchitectureDoc, SclDocument } from './types.ts'

const arch: ArchitectureDoc = {
  context: 'repo',
  updated_at: '2026-07-06',
  contexts: { repo: { root: '.', summary: 'tools' } },
  modules: {
    ra: { path: 'tools/ra', responsibility: 'cli', realizes: ['interfaces.DiscoverWorkspace'] },
  },
  overview: 'the map',
  structure: 'the tree and `ra` -> `yaml-check`',
  stack: '- TypeScript / Bun',
  structural_decisions: 'see repo-ADR-001',
}

const scl: SclDocument = { system: 'demo', spec_version: '2.0' }

describe('renderArchitectureTab', () => {
  it('renders frontmatter tables and prose', () => {
    const html = renderArchitectureTab(arch)
    expect(html).toContain('tools/ra')
    expect(html).toContain('interfaces.DiscoverWorkspace')
    expect(html).toContain('id="arch-structure"')
    expect(html).toContain('the tree')
    expect(html).toContain('the map')
    expect(html).toContain('id="arch-modules"')
  })

  it('is a placeholder when no map is given', () => {
    expect(renderArchitectureTab(null)).toContain('No architecture map')
  })
})

describe('architectureTocItems', () => {
  it('lists only the sections present', () => {
    const ids = architectureTocItems(arch).map((i) => i.id)
    expect(ids).toContain('arch-overview')
    expect(ids).toContain('arch-modules')
    expect(ids).not.toContain('arch-diagrams')
  })

  it('is empty when no map is given', () => {
    expect(architectureTocItems(null)).toEqual([])
  })
})

describe('renderPage with architecture', () => {
  it('adds an Architecture tab link when a map is present', () => {
    const html = renderPage({ scl, decisions: [], work_items: [], architecture: arch })
    expect(html).toContain('data-tab-link="architecture"')
    expect(html).toContain('data-tab="architecture"')
  })

  it('omits the Architecture tab when no map is present', () => {
    const html = renderPage({ scl, decisions: [], work_items: [], architecture: null })
    expect(html).not.toContain('data-tab-link="architecture"')
  })
})

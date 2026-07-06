/**
 * Render the Architecture tab: the second-layer current-state map
 * (ARCHITECTURE.md). The structured frontmatter (contexts, modules) becomes
 * tables; the prose body sections (overview, structure, stack, structural
 * decisions, cross-cutting, diagrams) are rendered as markdown.
 *
 * Pure — takes a parsed `ArchitectureDoc`, returns an HTML string. Section ids
 * are stable (`arch-*`) so the sidebar TOC and hash router can address them.
 */

import { esc } from './html.ts'
import { renderMarkdown } from './markdown.ts'
import type { ArchitectureDoc } from './types.ts'

type TocItem = { id: string; label: string }

const section = (id: string, title: string, body: string): string =>
  `<section id="${esc(id)}"><h2>${esc(title)}</h2>${body}</section>`

const chips = (values: string[]): string =>
  values.map((v) => `<span class="chip">${esc(v)}</span>`).join(' ')

const renderModules = (modules: ArchitectureDoc['modules']): string => {
  const rows = Object.entries(modules ?? {})
    .map(([id, mod]) => {
      const realizes = mod.realizes?.length ? chips(mod.realizes) : '<span class="muted">—</span>'
      return `<tr>
        <td><span class="name">${esc(id)}</span></td>
        <td><span class="path">${esc(mod.path ?? '')}</span></td>
        <td>${esc(mod.responsibility ?? '')}</td>
        <td>${realizes}</td>
      </tr>`
    })
    .join('')
  return `<table class="fields">
    <thead><tr><th>Module</th><th>Path</th><th>Responsibility</th><th>Realizes</th></tr></thead>
    <tbody>${rows}</tbody>
  </table>`
}

const renderContexts = (contexts: Record<string, { root?: string; summary?: string }>): string => {
  const rows = Object.entries(contexts)
    .map(
      ([prefix, ctx]) => `<tr>
        <td><span class="name">${esc(prefix)}</span></td>
        <td><span class="path">${esc(ctx.root ?? '')}</span></td>
        <td>${esc(ctx.summary ?? '')}</td>
      </tr>`,
    )
    .join('')
  return `<table class="fields">
    <thead><tr><th>Context</th><th>Root</th><th>Summary</th></tr></thead>
    <tbody>${rows}</tbody>
  </table>`
}

/** Build the ordered list of sections that this map actually has. */
const sections = (arch: ArchitectureDoc): Array<{ item: TocItem; body: string }> => {
  const out: Array<{ item: TocItem; body: string }> = []
  const push = (id: string, label: string, body: string | undefined) => {
    if (body) out.push({ item: { id, label }, body })
  }
  push('arch-overview', 'Overview', arch.overview && renderMarkdown(arch.overview))
  push(
    'arch-contexts',
    'Contexts',
    arch.contexts && Object.keys(arch.contexts).length ? renderContexts(arch.contexts) : undefined,
  )
  push(
    'arch-modules',
    'Modules',
    arch.modules && Object.keys(arch.modules).length ? renderModules(arch.modules) : undefined,
  )
  push('arch-structure', 'Structure', arch.structure && renderMarkdown(arch.structure))
  push('arch-stack', 'Stack', arch.stack && renderMarkdown(arch.stack))
  push(
    'arch-decisions',
    'Structural Decisions',
    arch.structural_decisions && renderMarkdown(arch.structural_decisions),
  )
  push(
    'arch-cross-cutting',
    'Cross-cutting Concerns',
    arch.cross_cutting_concerns && renderMarkdown(arch.cross_cutting_concerns),
  )
  push('arch-diagrams', 'Diagrams', arch.diagrams && renderMarkdown(arch.diagrams))
  return out
}

export const architectureTocItems = (arch: ArchitectureDoc | null | undefined): TocItem[] =>
  arch ? sections(arch).map((s) => s.item) : []

export const renderArchitectureTab = (arch: ArchitectureDoc | null | undefined): string => {
  if (!arch) return '<p class="lead">No architecture map.</p>'
  const meta = [
    arch.context ? `<span class="badge badge-context">${esc(arch.context)}</span>` : '',
    arch.updated_at
      ? `<span class="badge badge-version">updated ${esc(arch.updated_at)}</span>`
      : '',
  ]
    .filter(Boolean)
    .join(' ')
  const header = `<div class="page-header">
    <div class="eyebrow">Architecture</div>
    <h1>Architecture</h1>
    <div class="page-meta">${meta}</div>
  </div>`
  const body = sections(arch)
    .map((s) => section(s.item.id, s.item.label, s.body))
    .join('\n')
  return `${header}${body}`
}

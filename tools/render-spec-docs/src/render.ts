import { dirname, posix, relative, resolve } from 'node:path'
import MarkdownIt from 'markdown-it'
import type { CatalogProperty, CatalogSymbol } from './typespec-catalog.ts'

/**
 * What names a normative scenario outside the specification. Collected by
 * searching for the identifier, so the specification itself needs no back
 * references to code and stays the only place the behavior is stated.
 */
export type ScenarioTrace = {
  id: string
  /** Repository-relative code and test paths that name the scenario. */
  sources: string[]
  /** Work item paths that name the scenario. */
  workItems: string[]
}

export type SourceDocument = {
  path: string
  source: string
}

type OpenApiOperation = {
  operationId?: string
  summary?: string
  description?: string
  tags?: string[]
}

export type OpenApiDocument = {
  info?: { title?: string; version?: string }
  paths?: Record<string, Record<string, OpenApiOperation>>
  components?: { schemas?: Record<string, unknown> }
  [key: string]: unknown
}

type RenderedDocument = SourceDocument & {
  id: string
  title: string
  sections: string[]
  outputPath: string
  category: DocumentCategory
  /** For a context document and its children, the context slug they belong to. */
  context?: string
}

type DocumentCategory =
  | 'method'
  | 'whole-system'
  | 'whole-system-child'
  | 'context'
  | 'context-child'

export type RenderedSpecificationSite = {
  files: Record<string, string>
  assets: Record<string, string>
  operations: number
  tags: string[]
  models: number
  mermaidSources: string[]
}

const HTTP_METHODS = new Set(['get', 'put', 'post', 'delete', 'patch', 'head', 'options', 'trace'])

export function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function slug(value: string): string {
  return value
    .normalize('NFKD')
    .toLowerCase()
    .replace(/[^a-z0-9\p{L}]+/gu, '-')
    .replace(/^-|-$/g, '')
}

function stripFrontmatter(source: string): string {
  return source.replace(/^---\n[\s\S]*?\n---\n+/, '')
}

/** The file that declares a boundary is the page a reader lands on. */
function isEntryDocument(name: string): boolean {
  return name === 'SPECIFICATION.md' || name === 'README.md'
}

function documentMetadata(document: SourceDocument): RenderedDocument {
  const title = document.source.match(/^# (.+)$/m)?.[1]?.trim() ?? document.path
  const sections = [...document.source.matchAll(/^## (.+)$/gm)].map(
    (match) => match[1]?.trim() ?? '',
  )
  const rootDocument = document.path.match(/^spec\/([^/]+)$/)?.[1]
  if (rootDocument) {
    const stem = rootDocument.replace(/\.md$/, '')
    return isEntryDocument(rootDocument)
      ? {
          ...document,
          id: 'whole-system',
          title,
          sections,
          outputPath: 'specification/index.html',
          category: 'whole-system',
        }
      : {
          ...document,
          id: `whole-system-${slug(stem)}`,
          title,
          sections,
          outputPath: `specification/${slug(stem)}.html`,
          category: 'whole-system-child',
        }
  }
  const contextDocument = document.path.match(/^spec\/contexts\/([^/]+)\/([^/]+)$/)
  const context = contextDocument?.[1]
  const contextFile = contextDocument?.[2]
  if (context && contextFile) {
    const stem = contextFile.replace(/\.md$/, '')
    return isEntryDocument(contextFile)
      ? {
          ...document,
          id: `context-${context}`,
          title,
          sections,
          outputPath: `contexts/${context}/index.html`,
          category: 'context',
          context,
        }
      : {
          ...document,
          id: `context-${context}-${slug(stem)}`,
          title,
          sections,
          outputPath: `contexts/${context}/${slug(stem)}.html`,
          category: 'context-child',
          context,
        }
  }
  const name = document.path.split('/').at(-1)?.replace(/\.md$/, '') ?? document.path
  return {
    ...document,
    id: `method-${slug(name)}`,
    title,
    sections,
    outputPath: `method/${slug(name)}.html`,
    category: 'method',
  }
}

function pageHref(from: string, to: string, fragment?: string): string {
  const path = from === to ? '' : posix.relative(posix.dirname(from), to)
  const href = path || ''
  return `${href}${fragment ? `#${fragment}` : ''}` || '#'
}

function siteLink(from: string, to: string, label: string, fragment?: string): string {
  return `<a data-site-link href="${escapeHtml(pageHref(from, to, fragment))}">${escapeHtml(label)}</a>`
}

function cleanStateText(value: string): string {
  return value
    .replace(/<br\s*\/?>/gi, ' ')
    .replace(/[`*_]/g, '')
    .replace(/\\\|/g, '|')
    .trim()
}

function mermaidLabel(value: string): string {
  return cleanStateText(value)
    .replaceAll('"', "'")
    .replaceAll(':', ' -')
    .replace(/[<>]/g, '')
    .replace(/\s+/g, ' ')
}

function hasGuard(value: string | undefined): value is string {
  if (!value) return false
  return !['-', '—', '""', "''"].includes(value.trim())
}

function tableCells(line: string): string[] {
  const cells: string[] = []
  let cell = ''
  for (let index = 1; index < line.length - 1; index++) {
    const character = line[index]
    if (character === '\\' && line[index + 1] === '|') {
      cell += '|'
      index++
    } else if (character === '|') {
      cells.push(cell.trim())
      cell = ''
    } else {
      cell += character
    }
  }
  cells.push(cell.trim())
  return cells
}

function stateDiagram(machine: string, rows: string[][]): string {
  const states = new Map<string, string>()
  const stateId = (name: string) => {
    const clean = cleanStateText(name)
    const existing = states.get(clean)
    if (existing) return existing
    const id = `state_${states.size + 1}`
    states.set(clean, id)
    return id
  }
  for (const row of rows) {
    if (row[0]) stateId(row[0])
    if (row[3]) stateId(row[3])
  }
  const lines = ['stateDiagram-v2', `  %% Derived from ${mermaidLabel(machine)}`]
  for (const [name, id] of states) lines.push(`  state "${mermaidLabel(name)}" as ${id}`)
  for (const row of rows) {
    const from = row[0]
    const event = row[1]
    const guard = row[2]
    const to = row[3]
    const effects = row[4]
    if (!from || !to) continue
    const label = [event, hasGuard(guard) ? `[${guard}]` : '']
      .filter(Boolean)
      .map((part) => mermaidLabel(part ?? ''))
      .join(' ')
    lines.push(`  ${stateId(from)} --> ${stateId(to)}${label ? `: ${label}` : ''}`)
    if (effects) lines.push(`  %% Effects: ${mermaidLabel(effects)}`)
  }
  return lines.join('\n')
}

/**
 * `standalone` is a states.md, where every H2 is a machine. In the single
 * canonical document the machines are the H3s under `## State Transitions`.
 */
export function addDerivedStateDiagrams(source: string, standalone = false): string {
  const lines = source.split('\n')
  const insertions = new Map<number, string[]>()
  let inStates = standalone
  for (let index = 0; index < lines.length; index++) {
    const line = lines[index] ?? ''
    if (!standalone) {
      if (line === '## State Transitions') {
        inStates = true
        continue
      }
      if (line.startsWith('## ')) inStates = false
    }
    const isMachine = standalone
      ? line.startsWith('## ') && !line.startsWith('### ')
      : line.startsWith('### ')
    if (!inStates || !isMachine) continue
    const machine = line.slice(standalone ? 3 : 4).trim()
    let table = index + 1
    while (
      table < lines.length &&
      !lines[table]?.startsWith('### ') &&
      !lines[table]?.startsWith('## ') &&
      lines[table] !== '| From | Event | Guard | To | Effects |'
    ) {
      table++
    }
    if (lines[table] !== '| From | Event | Guard | To | Effects |') {
      throw new Error(`${machine} has no canonical state-transition table`)
    }
    if (!/^\|(?:\s*:?-+:?\s*\|){5}$/.test(lines[table + 1] ?? '')) {
      throw new Error(`${machine} has an invalid state-transition table separator`)
    }
    const rows: string[][] = []
    let row = table + 2
    while (row < lines.length && /^\|.*\|$/.test(lines[row] ?? '')) {
      const cells = tableCells(lines[row] ?? '')
      if (cells.length !== 5)
        throw new Error(`${machine} state-transition row must contain exactly five cells`)
      rows.push(cells)
      row++
    }
    if (rows.length === 0) throw new Error(`${machine} state-transition table has no rows`)
    insertions.set(table, ['```mermaid', stateDiagram(machine, rows), '```', ''])
  }
  const output: string[] = []
  for (const [index, line] of lines.entries()) {
    const insertion = insertions.get(index)
    if (insertion) output.push(...insertion)
    output.push(line)
  }
  return output.join('\n')
}

function markdownRenderer(
  documents: RenderedDocument[],
  repositoryRoot: string,
  outputDirectory: string,
  mermaidSources: string[],
) {
  const bySource = new Map(
    documents.map((document) => [resolve(repositoryRoot, document.path), document]),
  )
  const md = new MarkdownIt({ html: false, linkify: true, typographer: false })

  md.renderer.rules.heading_open = (tokens, index, _options, env) => {
    const token = tokens[index]
    const inline = tokens[index + 1]
    const level = token?.tag ?? 'h2'
    const prefix = (env as { document: RenderedDocument }).document.id
    return `<${level} id="${prefix}-${slug(inline?.content ?? '')}">`
  }

  const defaultText =
    md.renderer.rules.text ?? ((tokens, index) => escapeHtml(tokens[index]?.content ?? ''))
  md.renderer.rules.text = (tokens, index, options, env, self) => {
    const content = tokens[index]?.content ?? ''
    const match = content.match(/^(ACTOR|GIVEN|WHEN|THEN|ALT)\s+([\s\S]+)$/)
    if (!match) return defaultText(tokens, index, options, env, self)
    const keyword = match[1]?.toLowerCase() ?? ''
    return `<span class="scenario-keyword ${keyword}">${escapeHtml(match[1] ?? '')}</span> ${escapeHtml(match[2] ?? '')}`
  }

  const defaultFence =
    md.renderer.rules.fence ??
    ((tokens, index, options, _env, self) => self.renderToken(tokens, index, options))
  md.renderer.rules.fence = (tokens, index, options, env, self) => {
    const token = tokens[index]
    if (token?.info.trim() !== 'mermaid') return defaultFence(tokens, index, options, env, self)
    const source = token.content.trim()
    mermaidSources.push(source)
    return `<div class="diagram-shell"><pre class="mermaid">${escapeHtml(source)}</pre></div>\n`
  }

  const defaultLinkOpen =
    md.renderer.rules.link_open ??
    ((tokens, index, options, _env, self) => self.renderToken(tokens, index, options))
  md.renderer.rules.link_open = (tokens, index, options, env, self) => {
    const hrefIndex = tokens[index]?.attrIndex('href') ?? -1
    if (hrefIndex >= 0) {
      const token = tokens[index]
      const href = token?.attrs?.[hrefIndex]?.[1] ?? ''
      if (href && !/^[a-z]+:/i.test(href)) {
        const current = env as { document: RenderedDocument }
        const hash = href.indexOf('#')
        const pathPart = hash >= 0 ? href.slice(0, hash) : href
        const fragment = hash >= 0 ? href.slice(hash + 1) : ''
        const currentSource = resolve(repositoryRoot, current.document.path)
        const absolute = pathPart ? resolve(dirname(currentSource), pathPart) : currentSource
        const target = bySource.get(absolute)
        if (target) {
          const unprefixedFragment = fragment.startsWith(`${target.id}-`)
            ? fragment.slice(target.id.length + 1)
            : fragment
          const targetFragment = fragment ? `${target.id}-${slug(unprefixedFragment)}` : undefined
          token?.attrSet(
            'href',
            pageHref(current.document.outputPath, target.outputPath, targetFragment),
          )
          token?.attrSet('data-site-link', '')
        } else if (!pathPart) {
          token?.attrSet('href', `#${current.document.id}-${slug(fragment)}`)
          token?.attrSet('data-site-link', '')
        } else {
          const output = resolve(outputDirectory, current.document.outputPath)
          const rewritten = relative(dirname(output), absolute).replaceAll('\\', '/')
          token?.attrSet('href', `${rewritten}${fragment ? `#${fragment}` : ''}`)
        }
      }
    }
    return defaultLinkOpen(tokens, index, options, env, self)
  }
  return md
}

function stylesheetHref(page: string): string {
  return pageHref(page, 'assets/site.css')
}

function assetHref(page: string, asset: string): string {
  return pageHref(page, `assets/${asset}`)
}

function navigation(page: string, documents: RenderedDocument[]): string {
  const method = documents.filter((document) => document.category === 'method')
  const root = documents.find((document) => document.category === 'whole-system')
  const rootChildren = documents.filter((document) => document.category === 'whole-system-child')
  const contexts = documents.filter((document) => document.category === 'context')
  // The context a reader is inside is the only one whose files are worth
  // listing; expanding all of them at once buries the context list itself.
  const open = documents.find((document) => document.outputPath === page)?.context
  const link = (entry: RenderedDocument, child: boolean) => {
    const current = entry.outputPath === page ? ' aria-current="page"' : ''
    const cls = child ? ' class="nav-child"' : ''
    return `<a data-site-link${cls}${current} href="${escapeHtml(pageHref(page, entry.outputPath))}">${escapeHtml(entry.title)}</a>`
  }
  const group = (title: string, entries: RenderedDocument[]) =>
    `<section class="nav-group"><h2>${escapeHtml(title)}</h2>${entries
      .map((entry) => {
        const children =
          entry.category === 'context' && entry.context === open
            ? documents.filter(
                (document) =>
                  document.category === 'context-child' && document.context === entry.context,
              )
            : []
        return link(entry, false) + children.map((child) => link(child, true)).join('')
      })
      .join('')}</section>`
  const whole = root ? group('Whole System', [root, ...rootChildren]) : ''
  return `${group('Method', method)}${whole}${group('Contexts', contexts)}<section class="nav-group"><h2>References</h2>${siteLink(page, 'api/index.html', 'API Reference')}${siteLink(page, 'models/index.html', 'Model Catalog')}${siteLink(page, 'traceability/index.html', 'Traceability')}</section>`
}

function breadcrumbs(page: string, current: string): string {
  return `<nav class="breadcrumbs" aria-label="Breadcrumb">${siteLink(page, 'index.html', 'Specification')}<span aria-hidden="true">/</span><span>${escapeHtml(current)}</span></nav>`
}

function shell(args: {
  page: string
  title: string
  current: string
  body: string
  documents: RenderedDocument[]
  head?: string
  scripts?: string
}): string {
  const nav = navigation(args.page, args.documents)
  return `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>${escapeHtml(args.title)} · Specification</title><link rel="stylesheet" href="${escapeHtml(stylesheetHref(args.page))}">${args.head ?? ''}</head>
<body><a class="skip-link" href="#content">Skip to content</a><header class="mobile-header">${siteLink(args.page, 'index.html', 'Specification')}<details><summary>Navigation</summary><nav aria-label="Mobile">${nav}</nav></details></header><aside class="sidebar"><div class="site-title">${siteLink(args.page, 'index.html', 'Specification')}</div><nav aria-label="Primary">${nav}</nav></aside><main id="content">${breadcrumbs(args.page, args.current)}${args.body}</main>${args.scripts ?? ''}</body></html>\n`
}

function documentPage(
  document: RenderedDocument,
  documents: RenderedDocument[],
  markdown: MarkdownIt,
): string {
  const source = addDerivedStateDiagrams(
    stripFrontmatter(document.source),
    document.path.endsWith('/states.md'),
  )
  const body = `<article class="document">${markdown.render(source, { document })}</article>`
  const scripts = `<script src="${escapeHtml(assetHref(document.outputPath, 'mermaid.min.js'))}"></script><script src="${escapeHtml(assetHref(document.outputPath, 'site.js'))}"></script>`
  return shell({
    page: document.outputPath,
    title: document.title,
    current: document.title,
    body,
    documents,
    scripts,
  })
}

function card(page: string, path: string, title: string, description: string): string {
  return `<article class="card"><h2>${siteLink(page, path, title)}</h2><p>${escapeHtml(description)}</p></article>`
}

function landingPage(documents: RenderedDocument[], modelCount: number): string {
  const page = 'index.html'
  const root = documents.find((document) => document.category === 'whole-system')
  const method = documents.filter((document) => document.category === 'method')
  const contexts = documents.filter((document) => document.category === 'context')
  const body = `<section class="hero"><p class="eyebrow">Generated from canonical Markdown and TypeSpec</p><h1>Specification</h1><p>Read the whole-system design, bounded-context specifications, API contract, and complete TypeSpec model catalog without treating this generated site as a source.</p></section>
<section aria-labelledby="start"><h2 id="start">Start here</h2><div class="card-grid">${root ? card(page, root.outputPath, root.title, 'Cross-context ownership, current design, and the DDD context map.') : ''}${card(page, 'api/index.html', 'API Reference', 'The generated OpenAPI rendered by Swagger UI.')}${card(page, 'models/index.html', 'Model Catalog', `${modelCount} repository-owned TypeSpec symbols, including non-HTTP models.`)}</div></section>
<section aria-labelledby="method"><h2 id="method">Method</h2><div class="card-grid">${method.map((entry) => card(page, entry.outputPath, entry.title, 'Specification-first development guidance.')).join('')}</div></section>
<section aria-labelledby="contexts"><h2 id="contexts">Bounded contexts</h2><div class="card-grid">${contexts.map((entry) => card(page, entry.outputPath, entry.title, 'Overview, current design, state transitions, and scenarios.')).join('')}</div></section>`
  return shell({
    page,
    title: 'Specification',
    current: 'Home',
    body,
    documents,
  })
}

function safeJson(value: unknown): string {
  return JSON.stringify(value)
    .replaceAll('<', '\\u003c')
    .replaceAll('>', '\\u003e')
    .replaceAll('&', '\\u0026')
    .replaceAll('\u2028', '\\u2028')
    .replaceAll('\u2029', '\\u2029')
}

function apiPage(
  openapi: OpenApiDocument,
  openapiFileName: string,
  documents: RenderedDocument[],
): string {
  const page = 'api/index.html'
  const body = `<article class="reference-page"><header class="reference-header"><p class="eyebrow">OpenAPI-native reference</p><h1>API Reference</h1><p>Rendered directly from generated OpenAPI by Swagger UI. <a href="../../openapi/${encodeURIComponent(openapiFileName)}">Open raw OpenAPI JSON</a>.</p></header><div class="swagger-shell"><div id="swagger-ui" aria-label="API operations"></div></div></article>`
  const head = `<link rel="stylesheet" href="${escapeHtml(assetHref(page, 'swagger-ui.css'))}">`
  const scripts = `<script src="${escapeHtml(assetHref(page, 'swagger-ui-bundle.js'))}"></script><script>window.addEventListener('DOMContentLoaded',function(){SwaggerUIBundle({spec:${safeJson(openapi)},dom_id:'#swagger-ui',deepLinking:true,displayRequestDuration:true,tryItOutEnabled:false,persistAuthorization:false,docExpansion:'list',defaultModelsExpandDepth:1});});</script>`
  return shell({
    page,
    title: 'API Reference',
    current: 'API Reference',
    body,
    documents,
    head,
    scripts,
  })
}

function modelPath(symbol: CatalogSymbol): string {
  return `models/${slug(symbol.name)}.html`
}

function modelLink(
  page: string,
  name: string,
  symbols: Map<string, CatalogSymbol>,
  label = name,
): string {
  const symbol = symbols.get(name)
  return symbol ? siteLink(page, modelPath(symbol), label) : `<code>${escapeHtml(label)}</code>`
}

function referenceList(
  page: string,
  references: string[],
  symbols: Map<string, CatalogSymbol>,
): string {
  if (references.length === 0) return '<span class="muted">—</span>'
  return references.map((reference) => modelLink(page, reference, symbols)).join(', ')
}

function propertyRows(
  page: string,
  properties: CatalogProperty[],
  symbols: Map<string, CatalogSymbol>,
): string {
  return properties
    .map(
      (property) =>
        `<tr><th scope="row"><code>${escapeHtml(property.name)}</code>${property.optional ? '<span class="optional">optional</span>' : '<span class="required">required</span>'}</th><td><code>${escapeHtml(property.type)}</code>${property.default ? `<div class="meta">Default: <code>${escapeHtml(property.default)}</code></div>` : ''}</td><td>${property.doc ? escapeHtml(property.doc) : '<span class="muted">No description.</span>'}${property.constraints.length ? `<ul class="compact">${property.constraints.map((constraint) => `<li><code>${escapeHtml(constraint)}</code></li>`).join('')}</ul>` : ''}${property.references.length ? `<div class="meta">References: ${referenceList(page, property.references, symbols)}</div>` : ''}</td></tr>`,
    )
    .join('')
}

function scenarioIndex(documents: RenderedDocument[]): ScenarioEntry[] {
  const entries: ScenarioEntry[] = []
  for (const document of documents) {
    // Method documents illustrate the grammar in fenced examples; only
    // specification documents declare scenarios.
    if (document.category === 'method') continue
    for (const match of document.source.matchAll(/^### (REQ-[A-Z0-9-]+): (.+)$/gm)) {
      entries.push({
        id: match[1] ?? '',
        title: match[2] ?? '',
        document,
        anchor: `${document.id}-${slug(`${match[1]}: ${match[2]}`)}`,
      })
    }
  }
  return entries.sort((a, b) => a.id.localeCompare(b.id))
}

type ScenarioEntry = {
  id: string
  title: string
  document: RenderedDocument
  anchor: string
}

function traceabilityPage(documents: RenderedDocument[], traces: ScenarioTrace[]): string {
  const page = 'traceability/index.html'
  const byId = new Map(traces.map((trace) => [trace.id, trace]))
  const scenarios = scenarioIndex(documents)
  const covered = scenarios.filter((scenario) => (byId.get(scenario.id)?.sources.length ?? 0) > 0)
  const paths = (label: string, entries: string[]) =>
    entries.length === 0
      ? `<span class="trace-empty">no ${escapeHtml(label)}</span>`
      : `<ul class="trace-paths">${entries
          .map((entry) => `<li><code>${escapeHtml(entry)}</code></li>`)
          .join('')}</ul>`
  const rows = scenarios
    .map((scenario) => {
      const trace = byId.get(scenario.id)
      const href = `${pageHref(page, scenario.document.outputPath)}#${scenario.anchor}`
      return `<tr><th scope="row"><a data-site-link href="${escapeHtml(href)}">${escapeHtml(scenario.id)}</a><span class="trace-title">${escapeHtml(scenario.title)}</span></th><td>${paths('reference', trace?.sources ?? [])}</td><td>${paths('work item', trace?.workItems ?? [])}</td></tr>`
    })
    .join('')
  const body = `<header class="reference-header"><p class="eyebrow">Derived from the repository</p><h1>Traceability</h1><p>Every normative scenario, the code and tests that name its identifier, and the work items that recorded it. References are collected by searching for the identifier, so a scenario appears here only once something outside the specification names it. ${covered.length} of ${scenarios.length} scenarios carry at least one reference.</p></header><table class="trace-table"><thead><tr><th scope="col">Scenario</th><th scope="col">Code and tests</th><th scope="col">Work items</th></tr></thead><tbody>${rows}</tbody></table>`
  return shell({ page, title: 'Traceability', current: 'Traceability', body, documents })
}

function modelIndex(models: CatalogSymbol[], documents: RenderedDocument[]): string {
  const page = 'models/index.html'
  const groups = new Map<string, CatalogSymbol[]>()
  for (const model of models) {
    const entries = groups.get(model.namespace) ?? []
    entries.push(model)
    groups.set(model.namespace, entries)
  }
  const body = `<header class="reference-header"><p class="eyebrow">TypeSpec program</p><h1>Model Catalog</h1><p>Repository-owned models, enums, unions, and scalars are included whether or not an HTTP operation exposes them. Transport wrappers in <code>Operations</code> namespaces remain in the OpenAPI-native API Reference.</p><label class="model-search">Filter models <input type="search" data-model-search placeholder="Name, namespace, or description" autocomplete="off"></label></header>${[
    ...groups.entries(),
  ]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(
      ([namespace, entries]) =>
        `<section class="model-group" data-model-group><h2>${escapeHtml(namespace)}</h2><div class="model-list">${entries
          .map(
            (entry) =>
              `<article data-model-card data-search="${escapeHtml(`${entry.name} ${entry.doc ?? ''}`.toLowerCase())}"><div><span class="kind">${escapeHtml(entry.kind)}</span>${entry.apiExposed ? '<span class="api-exposed">API-exposed</span>' : ''}</div><h3>${siteLink(page, modelPath(entry), entry.shortName)}</h3><p>${escapeHtml(entry.doc ?? 'No description.')}</p></article>`,
          )
          .join('')}</div></section>`,
    )
    .join('')}`
  return shell({
    page,
    title: 'Model Catalog',
    current: 'Model Catalog',
    body,
    documents,
  })
}

function modelPage(
  model: CatalogSymbol,
  symbols: Map<string, CatalogSymbol>,
  documents: RenderedDocument[],
): string {
  const page = modelPath(model)
  const exposure = model.apiExposed
    ? '<span class="api-exposed">API-exposed</span>'
    : '<span class="not-exposed">Not API-exposed</span>'
  const properties = model.properties.length
    ? `<section><h2>Properties</h2><div class="table-wrap"><table><thead><tr><th>Name</th><th>Type</th><th>Description and constraints</th></tr></thead><tbody>${propertyRows(page, model.properties, symbols)}</tbody></table></div></section>`
    : ''
  const members = model.members.length
    ? `<section><h2>${model.kind === 'union' ? 'Variants' : 'Members'}</h2><div class="table-wrap"><table><thead><tr><th>Name</th><th>Value or type</th><th>Description</th></tr></thead><tbody>${model.members
        .map(
          (member) =>
            `<tr><th scope="row"><code>${escapeHtml(member.name)}</code></th><td><code>${escapeHtml(member.value ?? member.type ?? '—')}</code></td><td>${escapeHtml(member.doc ?? '—')}</td></tr>`,
        )
        .join('')}</tbody></table></div></section>`
    : ''
  const body = `<article class="model-detail"><header><p class="eyebrow">${escapeHtml(model.namespace)} · ${escapeHtml(model.kind)}</p><h1>${escapeHtml(model.shortName)}</h1><div class="badges">${exposure}</div><p>${escapeHtml(model.doc ?? 'No description.')}</p><p class="qualified"><strong>TypeSpec symbol</strong> <code>${escapeHtml(model.name)}</code></p>${model.base ? `<p><strong>Base</strong> ${modelLink(page, model.base, symbols)}</p>` : ''}</header>${properties}${members}<section><h2>References</h2><p>${referenceList(page, model.references, symbols)}</p></section></article>`
  return shell({
    page,
    title: model.shortName,
    current: model.shortName,
    body,
    documents,
  })
}

function inspectOpenApi(openapi: OpenApiDocument): { operations: number; tags: string[] } {
  let operations = 0
  const tags = new Set<string>()
  for (const [path, pathItem] of Object.entries(openapi.paths ?? {})) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!HTTP_METHODS.has(method.toLowerCase())) continue
      operations++
      const owners = operation.tags ?? []
      if (owners.length === 0 || owners.includes('default'))
        throw new Error(`${method.toUpperCase()} ${path} has no owning context tag`)
      for (const tag of owners) tags.add(tag)
    }
  }
  return { operations, tags: [...tags].sort() }
}

function validateSiteLinks(files: Record<string, string>): void {
  const ids = new Map(
    Object.entries(files).map(([path, html]) => [
      path,
      new Set([...html.matchAll(/\sid="([^"]+)"/g)].map((match) => match[1] ?? '')),
    ]),
  )
  const graph = new Map<string, Set<string>>()
  for (const [path, html] of Object.entries(files)) {
    const targets = new Set<string>()
    for (const anchor of html.matchAll(/<a\b[^>]*>/g)) {
      const tag = anchor[0]
      if (!tag.includes('data-site-link')) continue
      const href = tag.match(/\shref="([^"]+)"/)?.[1]
      if (!href) continue
      const [pathPart, fragment] = href.split('#', 2)
      const target = pathPart ? posix.normalize(posix.join(posix.dirname(path), pathPart)) : path
      if (!files[target]) throw new Error(`${path} links to missing generated page ${target}`)
      if (fragment && !ids.get(target)?.has(fragment))
        throw new Error(`${path} links to missing fragment ${target}#${fragment}`)
      targets.add(target)
    }
    graph.set(path, targets)
  }
  const reached = new Set<string>()
  const pending = ['index.html']
  while (pending.length) {
    const current = pending.shift()
    if (!current || reached.has(current)) continue
    reached.add(current)
    for (const target of graph.get(current) ?? []) pending.push(target)
  }
  const unreachable = Object.keys(files).filter((path) => !reached.has(path))
  if (unreachable.length)
    throw new Error(`generated pages are unreachable: ${unreachable.join(', ')}`)
}

const styles = `
:root{color-scheme:light dark;--bg:#f4f6fb;--panel:#fff;--panel-2:#f8f9fd;--text:#182033;--muted:#667085;--line:#d9dfeb;--accent:#3457d5;--accent-soft:#e9eeff;--code:#edf1f8;--shadow:0 12px 32px rgba(25,35,60,.08);--actor:#6d28d9;--given:#176b87;--when:#9a5b00;--then:#157347;--alt:#b42318;--diagram-line:#294cba}
@media(prefers-color-scheme:dark){:root{--bg:#0f131b;--panel:#171c27;--panel-2:#1d2431;--text:#eef2f8;--muted:#a7b0c1;--line:#30394b;--accent:#9db1ff;--accent-soft:#242f52;--code:#242b39;--shadow:none;--actor:#c4a7ff;--given:#7bd6f0;--when:#ffc36b;--then:#74d6a0;--alt:#ff9a91;--diagram-line:#b9c8ff}}
*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;color:var(--text);background:var(--bg);font:15px/1.7 Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;overflow-wrap:anywhere}.skip-link{position:fixed;z-index:20;top:8px;left:8px;transform:translateY(-160%);padding:8px 12px;background:var(--panel);border:2px solid var(--accent);border-radius:8px}.skip-link:focus{transform:none}.sidebar{position:fixed;inset:0 auto 0 0;width:300px;overflow:auto;padding:24px 18px;border-right:1px solid var(--line);background:var(--panel)}.site-title{margin:0 8px 18px;font-size:18px;font-weight:800}.site-title a{text-decoration:none}.nav-group{margin:18px 0}.nav-group h2{margin:0 8px 6px;color:var(--muted);font-size:11px;letter-spacing:.09em;text-transform:uppercase}.nav-group a{display:block;padding:5px 8px;color:var(--text);text-decoration:none;border-radius:7px}.nav-group a.nav-child{padding-left:22px;font-size:13px;color:var(--muted)}.nav-group a:hover,.nav-group a[aria-current=page]{color:var(--accent);background:var(--accent-soft)}main{width:min(1120px,calc(100% - 340px));margin-left:320px;padding:30px 26px 96px}.breadcrumbs{display:flex;gap:8px;align-items:center;margin:0 0 18px;color:var(--muted);font-size:13px}.mobile-header{display:none}.document,.reference-page,.model-detail,.hero{padding:38px 46px;border:1px solid var(--line);border-radius:16px;background:var(--panel);box-shadow:var(--shadow)}.hero{margin-bottom:28px;background:linear-gradient(145deg,var(--panel),var(--accent-soft))}.hero h1{margin:.1em 0;font-size:42px}.eyebrow{margin:0;color:var(--accent);font-size:12px;font-weight:800;letter-spacing:.09em;text-transform:uppercase}h1,h2,h3,h4{line-height:1.25;scroll-margin-top:18px}h1{font-size:32px}h2{margin-top:38px;padding-bottom:8px;border-bottom:1px solid var(--line)}h3{margin-top:28px}a{color:var(--accent);text-underline-offset:2px}a:focus-visible,summary:focus-visible,input:focus-visible{outline:3px solid var(--accent);outline-offset:3px;border-radius:4px}code{padding:.12em .35em;border-radius:5px;background:var(--code);font-size:.92em}pre{max-width:100%;overflow:auto;padding:16px;border-radius:10px;background:var(--code)}pre code{padding:0}.card-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(230px,1fr));gap:16px}.card{padding:20px;border:1px solid var(--line);border-radius:12px;background:var(--panel)}.card h2{margin:0;border:0;padding:0;font-size:18px}.card p{margin:.5em 0 0;color:var(--muted)}.diagram-shell{max-width:100%;overflow:auto;margin:20px 0;padding:16px;border:1px solid var(--line);border-radius:12px;background:var(--panel-2)}.diagram-shell .mermaid{min-width:560px;background:transparent}.diagram-shell .mermaid svg .edgePath path,.diagram-shell .mermaid svg .flowchart-link,.diagram-shell .mermaid svg .transition{stroke:var(--diagram-line)!important;stroke-width:2.4px!important}.diagram-shell .mermaid svg marker path{fill:var(--diagram-line)!important;stroke:var(--diagram-line)!important}.scenario-keyword{display:inline-block;min-width:58px;margin-right:5px;padding:1px 7px;border:1px solid currentColor;border-radius:999px;font-size:11px;font-weight:800;letter-spacing:.04em;text-align:center}.scenario-keyword.actor{color:var(--actor)}.scenario-keyword.given{color:var(--given)}.scenario-keyword.when{color:var(--when)}.scenario-keyword.then{color:var(--then)}.scenario-keyword.alt{color:var(--alt)}li:has(>.scenario-keyword){margin:.45em 0}li>ul li:has(>.scenario-keyword.alt){padding-left:8px;border-left:3px solid var(--alt)}.reference-header{margin-bottom:24px}.reference-page{max-width:none}.swagger-shell{color-scheme:light;margin:24px -20px -20px;padding:20px;overflow:auto;border-radius:12px;background:#fff;color:#3b4151}.model-group{margin-top:32px}.model-list{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:12px}.model-list article{padding:16px;border:1px solid var(--line);border-radius:10px;background:var(--panel)}.model-list h3{margin:.4em 0}.model-list p{color:var(--muted)}.trace-table th[scope=row]{display:grid;gap:2px;text-align:left;vertical-align:top}.trace-title{color:var(--muted);font-weight:400}.trace-paths{margin:0;padding-left:16px}.trace-paths code{font-size:12px}.trace-empty{color:var(--muted)}
.model-search{display:grid;max-width:520px;gap:6px;margin-top:20px;font-weight:700}.model-search input{width:100%;padding:10px 12px;color:var(--text);background:var(--panel);border:1px solid var(--line);border-radius:8px;font:inherit}.kind,.api-exposed,.not-exposed,.required,.optional{display:inline-block;margin:0 6px 4px 0;padding:2px 7px;border-radius:999px;font-size:11px;font-weight:800}.kind,.optional{color:var(--muted);background:var(--code)}.api-exposed,.required{color:#fff;background:#28664b}.not-exposed{color:var(--muted);border:1px solid var(--line)}.qualified{padding:12px;border-radius:8px;background:var(--panel-2)}.badges{margin:.5em 0}.table-wrap,table{max-width:100%;overflow:auto}table{width:100%;border-collapse:collapse;display:block}th,td{padding:10px 12px;border:1px solid var(--line);text-align:left;vertical-align:top}th{background:var(--panel-2)}.meta{margin-top:7px;color:var(--muted);font-size:13px}.compact{margin:.5em 0;padding-left:20px}.muted{color:var(--muted)}[hidden]{display:none!important}
@media(max-width:900px){.sidebar{display:none}.mobile-header{display:flex;position:sticky;z-index:10;top:0;justify-content:space-between;align-items:flex-start;padding:12px 18px;border-bottom:1px solid var(--line);background:var(--panel)}.mobile-header>details{position:relative}.mobile-header details>nav{position:absolute;right:0;width:min(86vw,320px);max-height:75vh;overflow:auto;padding:12px;border:1px solid var(--line);border-radius:10px;background:var(--panel);box-shadow:var(--shadow)}main{width:auto;margin:0;padding:18px}.document,.reference-page,.model-detail,.hero{padding:24px 20px}.hero h1{font-size:34px}.diagram-shell .mermaid{min-width:480px}}
@media print{.sidebar,.mobile-header,.breadcrumbs,.skip-link{display:none}main{width:auto;margin:0;padding:0}.document,.reference-page,.model-detail,.hero{border:0;box-shadow:none;padding:0}a{color:inherit;text-decoration:none}}
`

const siteScript = `
window.addEventListener('DOMContentLoaded',function(){
  if(window.mermaid&&document.querySelector('.mermaid')){
    var dark=window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches;
    window.mermaid.initialize({startOnLoad:false,securityLevel:'strict',theme:'base',themeVariables:dark?{background:'#1d2431',primaryColor:'#242f52',primaryTextColor:'#eef2f8',primaryBorderColor:'#b9c8ff',lineColor:'#b9c8ff',textColor:'#eef2f8',edgeLabelBackground:'#171c27',tertiaryColor:'#1d2431'}:{background:'#f8f9fd',primaryColor:'#e9eeff',primaryTextColor:'#182033',primaryBorderColor:'#294cba',lineColor:'#294cba',textColor:'#182033',edgeLabelBackground:'#fff',tertiaryColor:'#f8f9fd'}});
    window.mermaid.run({querySelector:'.mermaid'});
  }
  var search=document.querySelector('[data-model-search]');
  if(search){search.addEventListener('input',function(){
    var query=search.value.trim().toLowerCase();
    document.querySelectorAll('[data-model-card]').forEach(function(card){card.hidden=Boolean(query&&!card.dataset.search.includes(query));});
    document.querySelectorAll('[data-model-group]').forEach(function(group){group.hidden=group.querySelectorAll('[data-model-card]:not([hidden])').length===0;});
  });}
});
`

export function renderSpecificationSite(args: {
  documents: SourceDocument[]
  openapi: OpenApiDocument
  repositoryRoot: string
  outputDirectory: string
  openapiFileName: string
  models: CatalogSymbol[]
  traces?: ScenarioTrace[]
}): RenderedSpecificationSite {
  const documents = args.documents.map(documentMetadata)
  const openapi = inspectOpenApi(args.openapi)
  const modelPaths = new Map<string, string>()
  for (const model of args.models) {
    const path = modelPath(model)
    const previous = modelPaths.get(path)
    if (previous) throw new Error(`TypeSpec model URL collision: ${previous} and ${model.name}`)
    modelPaths.set(path, model.name)
  }
  const symbols = new Map(args.models.map((model) => [model.name, model]))
  const mermaidSources: string[] = []
  const markdown = markdownRenderer(
    documents,
    args.repositoryRoot,
    args.outputDirectory,
    mermaidSources,
  )
  const files: Record<string, string> = {
    'index.html': landingPage(documents, args.models.length),
    'api/index.html': apiPage(args.openapi, args.openapiFileName, documents),
    'models/index.html': modelIndex(args.models, documents),
    'traceability/index.html': traceabilityPage(documents, args.traces ?? []),
  }
  for (const document of documents)
    files[document.outputPath] = documentPage(document, documents, markdown)
  for (const model of args.models) files[modelPath(model)] = modelPage(model, symbols, documents)
  validateSiteLinks(files)
  return {
    files,
    assets: { 'site.css': styles, 'site.js': siteScript },
    operations: openapi.operations,
    tags: openapi.tags,
    models: args.models.length,
    mermaidSources,
  }
}

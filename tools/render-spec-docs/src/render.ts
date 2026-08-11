import { dirname, relative, resolve } from 'node:path'
import MarkdownIt from 'markdown-it'

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

type OpenApiDocument = {
  info?: { title?: string; version?: string }
  paths?: Record<string, Record<string, OpenApiOperation>>
  components?: {
    schemas?: Record<string, { description?: string; properties?: Record<string, unknown> }>
  }
}

type RenderedDocument = SourceDocument & {
  id: string
  title: string
  sections: string[]
}

const HTTP_METHODS = new Set(['get', 'put', 'post', 'delete', 'patch', 'head', 'options', 'trace'])

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

function slug(value: string): string {
  return value
    .normalize('NFKD')
    .toLowerCase()
    .replace(/[^a-z0-9\p{L}]+/gu, '-')
    .replace(/^-|-$/g, '')
}

function documentId(path: string): string {
  if (path === 'spec/SPECIFICATION.md') return 'repository'
  const context = path.match(/^spec\/contexts\/([^/]+)\/SPECIFICATION\.md$/)?.[1]
  if (context) return `context-${context}`
  return `guide-${slug(path.replace(/\.md$/, ''))}`
}

function metadata(document: SourceDocument): RenderedDocument {
  const title = document.source.match(/^# (.+)$/m)?.[1]?.trim() ?? document.path
  const sections = [...document.source.matchAll(/^## (.+)$/gm)].map(
    (match) => match[1]?.trim() ?? '',
  )
  return { ...document, id: documentId(document.path), title, sections }
}

function stripFrontmatter(source: string): string {
  return source.replace(/^---\n[\s\S]*?\n---\n+/, '')
}

function markdownRenderer(documents: RenderedDocument[], repositoryRoot: string, output: string) {
  const ids = new Map(
    documents.map((document) => [resolve(repositoryRoot, document.path), document.id]),
  )
  const md = new MarkdownIt({ html: false, linkify: true, typographer: false })

  md.renderer.rules.heading_open = (tokens, index, _options, env) => {
    const token = tokens[index]
    const inline = tokens[index + 1]
    const level = token?.tag ?? 'h2'
    const prefix = (env as { documentId: string }).documentId
    return `<${level} id="${prefix}-${slug(inline?.content ?? '')}">`
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
        const current = env as { documentId: string; sourcePath: string }
        const hash = href.indexOf('#')
        const pathPart = hash >= 0 ? href.slice(0, hash) : href
        const fragment = hash >= 0 ? href.slice(hash + 1) : ''
        const absolute = pathPart
          ? resolve(dirname(resolve(repositoryRoot, current.sourcePath)), pathPart)
          : resolve(repositoryRoot, current.sourcePath)
        const targetId = ids.get(absolute)
        let rewritten: string
        if (targetId) {
          rewritten = `#${targetId}${fragment ? `-${fragment}` : ''}`
        } else if (!pathPart) {
          rewritten = `#${current.documentId}${fragment ? `-${fragment}` : ''}`
        } else {
          rewritten = relative(dirname(output), absolute).replaceAll('\\', '/')
          if (fragment) rewritten += `#${fragment}`
        }
        token?.attrSet('href', rewritten)
      }
    }
    return defaultLinkOpen(tokens, index, options, env, self)
  }
  return md
}

function apiReference(openapi: OpenApiDocument): {
  html: string
  operations: number
  tags: string[]
} {
  const grouped = new Map<
    string,
    Array<{ method: string; path: string; operation: OpenApiOperation }>
  >()
  let operations = 0
  for (const [path, pathItem] of Object.entries(openapi.paths ?? {})) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!HTTP_METHODS.has(method.toLowerCase())) continue
      operations++
      const tags = operation.tags ?? []
      if (tags.length === 0 || tags.includes('default')) {
        throw new Error(`${method.toUpperCase()} ${path} has no owning context tag`)
      }
      for (const tag of tags) {
        const entries = grouped.get(tag) ?? []
        entries.push({ method: method.toUpperCase(), path, operation })
        grouped.set(tag, entries)
      }
    }
  }

  const tags = [...grouped.keys()].sort()
  const groups = tags
    .map((tag) => {
      const rows = (grouped.get(tag) ?? [])
        .sort((a, b) => a.path.localeCompare(b.path) || a.method.localeCompare(b.method))
        .map(
          ({ method, path, operation }) => `<tr>
<td><span class="method method-${method.toLowerCase()}">${method}</span></td>
<td><code>${escapeHtml(path)}</code></td>
<td><code>${escapeHtml(operation.operationId ?? '')}</code></td>
<td>${escapeHtml(operation.summary ?? operation.description ?? '')}</td>
</tr>`,
        )
        .join('\n')
      return `<section class="api-group" id="api-${slug(tag)}">
<h2>${escapeHtml(tag)}</h2>
<table><thead><tr><th>Method</th><th>Path</th><th>Operation</th><th>Description</th></tr></thead>
<tbody>${rows}</tbody></table>
</section>`
    })
    .join('\n')

  const schemas = Object.entries(openapi.components?.schemas ?? {})
    .sort(([a], [b]) => a.localeCompare(b))
    .map(
      ([name, schema]) =>
        `<tr><td><code>${escapeHtml(name)}</code></td><td>${escapeHtml(
          Object.keys(schema.properties ?? {}).join(', '),
        )}</td><td>${escapeHtml(schema.description ?? '')}</td></tr>`,
    )
    .join('\n')

  return {
    operations,
    tags,
    html: `<article class="document" id="api-reference">
<h1>API Reference</h1>
<p>Generated from TypeSpec. <a href="../openapi/idmagic.openapi.json">OpenAPI JSON</a></p>
${groups}
<section id="api-models"><h2>Models</h2>
<table><thead><tr><th>Schema</th><th>Properties</th><th>Description</th></tr></thead>
<tbody>${schemas}</tbody></table></section>
</article>`,
  }
}

const styles = `
:root { color-scheme: light dark; --bg: #f7f8fa; --panel: #fff; --text: #172033; --muted: #667085; --line: #d9deea; --accent: #3859d9; --code: #eef1f8; }
@media (prefers-color-scheme: dark) { :root { --bg: #11141b; --panel: #181d27; --text: #eef1f7; --muted: #9da8bb; --line: #303849; --accent: #8ca5ff; --code: #242b39; } }
* { box-sizing: border-box; }
html { scroll-behavior: smooth; }
body { margin: 0; color: var(--text); background: var(--bg); font: 15px/1.65 system-ui, sans-serif; }
nav { position: fixed; inset: 0 auto 0 0; width: 290px; overflow: auto; padding: 24px 18px; border-right: 1px solid var(--line); background: var(--panel); }
nav h1 { margin: 0 0 18px; font-size: 18px; }
nav h2 { margin: 20px 0 6px; color: var(--muted); font-size: 11px; letter-spacing: .08em; text-transform: uppercase; }
nav a { display: block; padding: 3px 8px; color: var(--text); text-decoration: none; border-radius: 5px; }
nav a:hover { color: var(--accent); background: var(--code); }
nav .sub { padding-left: 20px; color: var(--muted); font-size: 13px; }
main { width: min(1180px, calc(100% - 330px)); margin-left: 310px; padding: 36px 30px 100px; }
.document { margin: 0 0 42px; padding: 34px 42px; border: 1px solid var(--line); border-radius: 12px; background: var(--panel); }
h1, h2, h3, h4 { line-height: 1.25; scroll-margin-top: 18px; }
h1 { font-size: 30px; } h2 { margin-top: 36px; border-bottom: 1px solid var(--line); padding-bottom: 8px; } h3 { margin-top: 28px; }
a { color: var(--accent); } code { padding: .12em .35em; border-radius: 4px; background: var(--code); }
pre { overflow: auto; padding: 16px; border-radius: 8px; background: var(--code); } pre code { padding: 0; }
table { width: 100%; border-collapse: collapse; display: block; overflow-x: auto; }
th, td { padding: 9px 12px; border: 1px solid var(--line); text-align: left; vertical-align: top; }
th { background: var(--code); white-space: nowrap; }
.method { display: inline-block; min-width: 58px; padding: 2px 7px; border-radius: 4px; color: white; text-align: center; font-size: 12px; font-weight: 700; }
.method-get { background: #25734f; } .method-post { background: #315dcc; } .method-put, .method-patch { background: #a35b12; } .method-delete { background: #b93838; }
@media (max-width: 850px) { nav { position: static; width: auto; max-height: 45vh; border-right: 0; border-bottom: 1px solid var(--line); } main { width: auto; margin: 0; padding: 18px; } .document { padding: 22px; } }
`

export function renderSpecificationSite(args: {
  documents: SourceDocument[]
  openapi: OpenApiDocument
  repositoryRoot: string
  output: string
}): { html: string; operations: number; tags: string[] } {
  const documents = args.documents.map(metadata)
  const markdown = markdownRenderer(documents, args.repositoryRoot, args.output)
  const api = apiReference(args.openapi)
  const guideDocs = documents.filter((document) => !document.path.startsWith('spec/'))
  const productDocs = documents.filter((document) => document.path.startsWith('spec/'))

  const navGroup = (title: string, entries: RenderedDocument[]) =>
    `<h2>${title}</h2>${entries
      .map(
        (document) =>
          `<a href="#${document.id}">${escapeHtml(document.title)}</a>${document.sections
            .map(
              (section) =>
                `<a class="sub" href="#${document.id}-${slug(section)}">${escapeHtml(section)}</a>`,
            )
            .join('')}`,
      )
      .join('')}`
  const apiNav = api.tags
    .map((tag) => `<a class="sub" href="#api-${slug(tag)}">${escapeHtml(tag)}</a>`)
    .join('')
  const nav = `${navGroup('Method', guideDocs)}${navGroup('Specifications', productDocs)}<h2>API</h2><a href="#api-reference">API Reference</a>${apiNav}<a class="sub" href="#api-models">Models</a>`

  const renderedDocs = documents
    .map(
      (document) => `<article class="document" id="${document.id}">
${markdown.render(stripFrontmatter(document.source), {
  documentId: document.id,
  sourcePath: document.path,
})}
</article>`,
    )
    .join('\n')

  return {
    operations: api.operations,
    tags: api.tags,
    html: `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Specification</title><style>${styles}</style></head>
<body><nav><h1>Specification</h1>${nav}</nav><main>${renderedDocs}${api.html}</main></body></html>\n`,
  }
}

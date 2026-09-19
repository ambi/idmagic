import { dirname, posix, relative, resolve } from 'node:path'
import MarkdownIt, { type MarkdownIt as MarkdownItInstance } from 'markdown-it'
import { parseScenarioDocument } from '../../check/src/gherkin-scenarios.ts'
import {
  CONTEXT_DOCUMENTS,
  DOMAIN_DOCUMENTS,
  SYSTEM_DOCUMENT_PATHS,
} from '../../workspace/src/document-layout.ts'
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
  /** Reason the executable example remains in the migration debt baseline. */
  debt?: string
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
  /** Position within the group the document is listed in. */
  order: number
  /** For a context document and its children, the context slug they belong to. */
  context?: string
}

type NavigationDirectory = {
  name: string
  document?: RenderedDocument
  documents: RenderedDocument[]
  directories: NavigationDirectory[]
}

type DocumentCategory =
  | 'format'
  | 'domain'
  | 'domain-child'
  | 'development'
  | 'development-child'
  | 'operations'
  | 'operations-child'
  | 'runbook'
  | 'whole-system'
  | 'whole-system-child'
  | 'context'
  | 'context-child'

export type RenderedDocumentationSite = {
  files: Record<string, string>
  assets: Record<string, string>
  operations: number
  tags: string[]
  models: number
  mermaidSources: string[]
}

/** 用語表の 1 列目の見出し。日本語の文書と、まだ英語の Context 文書の両方を含む。 */
const TERM_HEADINGS = new Set(['用語', 'Term'])

const HTTP_METHODS = new Set(['get', 'put', 'post', 'delete', 'patch', 'head', 'options', 'trace'])
const SITE_TITLE = 'IdMagic ドキュメント'

export function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

/**
 * TypeSpec の文書コメントは Markdown として描画し、コードを表すバッククォートをそのまま表示しない。
 * 生の HTML は無効なままとし、信頼できないマークアップをエスケープする。
 */
const inlineMarkdown = new MarkdownIt({ html: false, linkify: false, typographer: false })

function renderDoc(value: string | undefined, fallback = '説明未記載。'): string {
  return inlineMarkdown.renderInline(value ?? fallback)
}

function slug(value: string): string {
  return value
    .normalize('NFKD')
    .toLowerCase()
    .replace(/[^a-z0-9\p{L}]+/gu, '-')
    .replace(/^-|-$/g, '')
}

/**
 * markdown-it は href を百分率符号化して属性に載せる。見出しの綴りは元の文字から
 * 作るので、断片は綴りに直す前に復号する。復号できない綴りはそのまま扱う。
 */
function decodeFragment(fragment: string): string {
  try {
    return decodeURIComponent(fragment)
  } catch {
    return fragment
  }
}

function stripFrontmatter(source: string): string {
  return source.replace(/^---\n[\s\S]*?\n---\n+/, '')
}

/**
 * The canonical layout already states the order the documents are meant to be
 * read in, so the listing follows the name list rather than the alphabet.
 */
function canonicalOrder(names: readonly string[], name: string, fallback: number): number {
  const index = names.indexOf(name)
  return index < 0 ? fallback : index
}

/**
 * 表題の Markdown は本文としては組まれるが、名札、パンくず、`<title>` では地の文になる。
 * 記法を残すと `` `/token` のエラー率 `` や `**必須**` がそのまま読み手へ出る。囲みの
 * 種類を数え上げると強調や参照を取りこぼすので、インラインとして組んでからタグを外す。
 * 呼び出し側が改めて escape するため、実体参照は元の文字へ戻す。
 */
function plainTitle(value: string): string {
  return inlineMarkdown
    .renderInline(value)
    .replace(/<[^>]*>/g, '')
    .replaceAll('&quot;', '"')
    .replaceAll('&#39;', "'")
    .replaceAll('&lt;', '<')
    .replaceAll('&gt;', '>')
    .replaceAll('&amp;', '&')
}

function documentMetadata(document: SourceDocument, index: number): RenderedDocument {
  const declaredTitle = plainTitle(document.source.match(/^# (.+)$/m)?.[1]?.trim() ?? document.path)
  const title =
    document.path === 'docs/domain/scenarios.feature.md' ? 'システム横断シナリオ' : declaredTitle
  const sections = [...document.source.matchAll(/^## (.+)$/gm)].map(
    (match) => match[1]?.trim() ?? '',
  )
  const operationsDocument = document.path.match(/^docs\/operations\/([^/]+)$/)?.[1]
  if (operationsDocument) {
    const stem = operationsDocument.replace(/\.md$/, '')
    return operationsDocument === 'README.md'
      ? {
          ...document,
          id: 'operations',
          title,
          sections,
          outputPath: 'operations/index.html',
          category: 'operations',
          order: 0,
        }
      : {
          ...document,
          id: `operations-${slug(stem)}`,
          title,
          sections,
          outputPath: `operations/${slug(stem)}.html`,
          category: 'operations-child',
          order: canonicalOrder(SYSTEM_DOCUMENT_PATHS, document.path, index),
        }
  }
  const runbookDocument = document.path.match(/^docs\/runbooks\/([^/]+)$/)?.[1]
  if (runbookDocument) {
    const stem = runbookDocument.replace(/\.md$/, '')
    return {
      ...document,
      id: `runbook-${slug(stem)}`,
      title,
      sections,
      outputPath: `operations/runbooks/${slug(stem)}.html`,
      category: 'runbook',
      order: index,
    }
  }
  const domainDocument = document.path.match(/^docs\/domain\/([^/]+\.md)$/)?.[1]
  if (domainDocument) {
    const stem =
      domainDocument === 'scenarios.feature.md' ? 'scenarios' : domainDocument.replace(/\.md$/, '')
    return domainDocument === 'README.md'
      ? {
          ...document,
          id: 'domain',
          title,
          sections,
          outputPath: 'domain/index.html',
          category: 'domain',
          order: 0,
        }
      : {
          ...document,
          id: `domain-${slug(stem)}`,
          title,
          sections,
          outputPath: `domain/${slug(stem)}.html`,
          category: 'domain-child',
          order: canonicalOrder(DOMAIN_DOCUMENTS, domainDocument, index),
        }
  }
  const systemDocument = document.path.match(/^docs\/(.+)$/)?.[1]
  if (
    systemDocument &&
    !systemDocument.startsWith('domain/') &&
    !systemDocument.startsWith('development/')
  ) {
    const segments = systemDocument.split('/')
    const file = segments.at(-1) ?? systemDocument
    const stem = file === 'scenarios.feature.md' ? 'scenarios' : file.replace(/\.md$/, '')
    const directory = segments.slice(0, -1).map(slug).join('/')
    const outputPath =
      file === 'README.md'
        ? directory
          ? `docs/${directory}/index.html`
          : 'docs/index.html'
        : `docs/${directory ? `${directory}/` : ''}${slug(stem)}.html`
    return systemDocument === 'README.md'
      ? {
          ...document,
          id: 'whole-system',
          title,
          sections,
          outputPath: 'index.html',
          category: 'whole-system',
          order: 0,
        }
      : {
          ...document,
          id: `whole-system-${slug(systemDocument)}`,
          title,
          sections,
          outputPath,
          category: 'whole-system-child',
          order: canonicalOrder(SYSTEM_DOCUMENT_PATHS, document.path, index),
        }
  }
  const developmentDocument = document.path.match(/^docs\/development\/([^/]+)$/)?.[1]
  if (developmentDocument) {
    const stem = developmentDocument.replace(/\.md$/, '')
    return developmentDocument === 'README.md'
      ? {
          ...document,
          id: 'development',
          title,
          sections,
          outputPath: 'development/index.html',
          category: 'development',
          order: 0,
        }
      : {
          ...document,
          id: `development-${slug(stem)}`,
          title,
          sections,
          outputPath: `development/${slug(stem)}.html`,
          category: 'development-child',
          order: index,
        }
  }
  const contextDocument = document.path.match(/^docs\/domain\/([^/]+)\/([^/]+)$/)
  const context = contextDocument?.[1]
  const contextFile = contextDocument?.[2]
  if (context && contextFile) {
    const stem =
      contextFile === 'scenarios.feature.md' ? 'scenarios' : contextFile.replace(/\.md$/, '')
    return contextFile === 'README.md'
      ? {
          ...document,
          id: `context-${context}`,
          title,
          sections,
          outputPath: `domain/${context}/index.html`,
          category: 'context',
          order: index,
          context,
        }
      : {
          ...document,
          id: `context-${context}-${slug(stem)}`,
          title,
          sections,
          outputPath: `domain/${context}/${slug(stem)}.html`,
          category: 'context-child',
          order: canonicalOrder(CONTEXT_DOCUMENTS, contextFile, index),
          context,
        }
  }
  const name = document.path.split('/').at(-1)?.replace(/\.md$/, '') ?? document.path
  return {
    ...document,
    id: `format-${slug(name)}`,
    title,
    sections,
    outputPath: `format/${slug(name)}.html`,
    category: 'format',
    order: index,
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

  // Gherkin のキーワードは英語の普通の語でもある。印を付けるのは規範シナリオの
  // 一次情報に限り、方法論文書の "When the work is complete, ..." のような本文や、
  // その中の箇条書きは対象にしない。キーワードと主役の名札は行頭にしか立たない
  // ので、インラインの先頭であることも条件にする。
  const defaultText =
    md.renderer.rules.text ?? ((tokens, index) => escapeHtml(tokens[index]?.content ?? ''))
  md.renderer.rules.text = (tokens, index, options, env, self) => {
    const content = tokens[index]?.content ?? ''
    const leading =
      index === 0 &&
      (env as { document: RenderedDocument }).document.path.endsWith('scenarios.feature.md')
    // ステップの残りが `applications:read` のようなコード片から始まると、この
    // テキストトークンはキーワードだけで終わる。残りを必須にすると、そういう
    // ステップだけ印が消える。
    const match = leading
      ? content.match(/^(Given|When|Then|And|But)(?:\s+([\s\S]*))?$/)
      : undefined
    if (!match) {
      // 主役は実行ステップではないので、`Rule` の説明として書かれる。
      const actor = leading ? content.match(/^(Primary actor):\s*([\s\S]*)$/) : undefined
      if (!actor) return defaultText(tokens, index, options, env, self)
      return `<span class="scenario-actor">${escapeHtml(actor[1] ?? '')}</span> ${escapeHtml(actor[2] ?? '')}`
    }
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

  const defaultTableOpen =
    md.renderer.rules.table_open ??
    ((tokens, index, options, _env, self) => self.renderToken(tokens, index, options))
  md.renderer.rules.table_open = (tokens, index, options, env, self) => {
    // A glossary term is a single identifier. Letting the layout break it apart
    // to widen the definition column leaves the term column unreadable.
    const heading = tokens.slice(index, index + 6).find((token) => token.type === 'inline')
    if (TERM_HEADINGS.has(heading?.content.trim() ?? ''))
      tokens[index]?.attrJoin('class', 'term-table')
    return defaultTableOpen(tokens, index, options, env, self)
  }

  const defaultLinkOpen =
    md.renderer.rules.link_open ??
    ((tokens, index, options, _env, self) => self.renderToken(tokens, index, options))
  md.renderer.rules.link_open = (tokens, index, options, env, self) => {
    const hrefIndex = tokens[index]?.attrIndex('href') ?? -1
    if (hrefIndex >= 0) {
      const token = tokens[index]
      // markdown-it 15 の属性値は string | number。href は常に文字列だが型の上では絞り込みが要る。
      const href = String(token?.attrs?.[hrefIndex]?.[1] ?? '')
      if (href && !/^[a-z]+:/i.test(href)) {
        const current = env as { document: RenderedDocument }
        const hash = href.indexOf('#')
        const pathPart = hash >= 0 ? href.slice(0, hash) : href
        const fragment = hash >= 0 ? decodeFragment(href.slice(hash + 1)) : ''
        const currentSource = resolve(repositoryRoot, current.document.path)
        const absolute = pathPart ? resolve(dirname(currentSource), pathPart) : currentSource
        // `[設計](design/)` のようなディレクトリへの参照は、リポジトリでは読めるが生成
        // サイトには対応するページがない。その段の索引へ向ける。
        const target = bySource.get(absolute) ?? bySource.get(resolve(absolute, 'README.md'))
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

function inGroup(documents: RenderedDocument[], category: DocumentCategory): RenderedDocument[] {
  return documents
    .filter((document) => document.category === category)
    .sort((a, b) => a.order - b.order)
}

/**
 * A child repeats its owner's name in its own title, which reads as noise
 * directly beneath it. Dropping that prefix leaves the kind of content the
 * file holds, which is what tells the children apart.
 */
function childLabel(entry: RenderedDocument, documents: RenderedDocument[]): string {
  // 「運用手順」の枝の下では、題名の末尾の種類名は枝の名札と同じことを言う。
  if (entry.category === 'runbook') return entry.title.replace(/のランブック$/, '')
  // 入れ子の段そのものが所属を示す段では、名札は題名だけでよい。
  if (entry.category !== 'context-child') return entry.title
  if (entry.path.endsWith('scenarios.feature.md')) return 'シナリオ'
  const owner = documents.find(
    (document) => document.category === 'context' && document.context === entry.context,
  )
  if (!owner) return entry.title
  const japanesePossessive = `${owner.title} の`
  if (entry.title.startsWith(japanesePossessive))
    return entry.title.slice(japanesePossessive.length)
  const spacedPrefix = `${owner.title} `
  return entry.title.startsWith(spacedPrefix) ? entry.title.slice(spacedPrefix.length) : entry.title
}

/**
 * `docs/` から見た段。ディレクトリの索引はその段自身に、ほかの文書は索引の一段下に
 * 置く。上から下へ分解した体系は、この入れ子でしか読み手に伝わらない。
 */
function documentTree(documents: RenderedDocument[]): NavigationDirectory {
  const root: NavigationDirectory = { name: 'docs', documents: [], directories: [] }
  for (const document of documents) {
    const parts = document.path.slice('docs/'.length).split('/')
    const file = parts.pop() ?? ''
    let node = root
    for (const part of parts) {
      let directory = node.directories.find((candidate) => candidate.name === part)
      if (!directory) {
        directory = { name: part, documents: [], directories: [] }
        node.directories.push(directory)
      }
      node = directory
    }
    if (file === 'README.md') node.document = document
    else node.documents.push(document)
  }
  return root
}

function navigation(page: string, documents: RenderedDocument[]): string {
  const formats = inGroup(documents, 'format')
  const index = (category: DocumentCategory) =>
    documents.find((document) => document.category === category)
  const development = index('development')
  const operations = index('operations')
  const rootChildren = inGroup(documents, 'whole-system-child')
  const contexts = inGroup(documents, 'context')
  const runbooks = inGroup(documents, 'runbook')
  const link = (entry: RenderedDocument, label?: string, extraClass = '') => {
    const marker = entry.outputPath === page ? ' aria-current="page"' : ''
    const display = label ?? entry.title
    return `<a data-site-link class="nav-link${extraClass}"${marker} href="${escapeHtml(pageHref(page, entry.outputPath))}">${escapeHtml(display)}</a>`
  }
  const leaf = (entry: RenderedDocument) =>
    `<li class="nav-item">${link(
      entry,
      childLabel(entry, documents),
      entry.category === 'context-child' ? ' nav-context-child' : '',
    )}</li>`
  const containsCurrent = (node: NavigationDirectory): boolean =>
    node.document?.outputPath === page ||
    node.documents.some((document) => document.outputPath === page) ||
    node.directories.some(containsCurrent)
  /**
   * 枝は `details` として畳む。子は常に HTML へ載るので、どのページからでも到達でき、
   * 開くかどうかの判断だけを現在位置に任せられる。読み手は自分で開くこともできる。
   * 子を出す条件と到達性を一つの規則で両立させるため、例外は置かない。
   */
  const directory = (node: NavigationDirectory): string => {
    const children = [...node.documents.map(leaf), ...node.directories.map(directory)].join('')
    const heading = node.document
      ? link(node.document)
      : `<span class="nav-label">${escapeHtml(node.name)}</span>`
    if (!children) return `<li class="nav-branch">${heading}</li>`
    return `<li class="nav-branch"><details class="nav-directory"${containsCurrent(node) ? ' open' : ''}><summary>${heading}</summary><ul>${children}</ul></details></li>`
  }
  const referenceLink = (path: string, label: string) => {
    const marker = page === path ? ' aria-current="page"' : ''
    return `<li class="nav-item"><a data-site-link class="nav-link"${marker} href="${escapeHtml(pageHref(page, path))}">${escapeHtml(label)}</a></li>`
  }
  const group = (title: string, body: string, index?: RenderedDocument | string) => {
    const indexPath = typeof index === 'string' ? index : index?.outputPath
    const marker = indexPath === page ? ' aria-current="page"' : ''
    const heading = indexPath
      ? `<a data-site-link class="nav-section-link"${marker} href="${escapeHtml(pageHref(page, indexPath))}">${escapeHtml(title)}</a>`
      : escapeHtml(title)
    const open = marker || body.includes('aria-current="page"') ? ' open' : ''
    return `<details class="nav-section"${open}><summary>${heading}</summary><ul class="nav-tree">${body}</ul></details>`
  }
  const tree = documentTree(rootChildren)
  /**
   * 「システム設計」は 7 領域の索引にすぎず、抽象的な名前のまま段を一つ増やす。枝を
   * 廃して領域を区分の直下へ上げる。索引そのものは `docs/README.md` の文書体系から辿る。
   */
  const designNode = tree.directories.find((node) => node.name === 'design')
  const areas = tree.directories.flatMap((node) =>
    node.name === 'design' ? node.directories.map(directory) : [directory(node)],
  )
  const domainBody = [
    ...inGroup(documents, 'domain-child').map(leaf),
    ...contexts.map((entry) =>
      directory({
        name: entry.title,
        document: entry,
        documents: inGroup(documents, 'context-child').filter(
          (document) => document.context === entry.context,
        ),
        directories: [],
      }),
    ),
  ].join('')
  const designIndex = designNode?.document
  const designBody = [
    ...(designNode?.documents ?? []).map(leaf),
    ...tree.documents.map(leaf),
    ...areas,
  ].join('')
  const operationsBody = [
    ...inGroup(documents, 'operations-child').map(leaf),
    ...(runbooks.length
      ? [directory({ name: '運用手順', documents: runbooks, directories: [] })]
      : []),
  ].join('')
  return [
    designBody ? group('設計文書', designBody, designIndex) : '',
    domainBody ? group('ドメイン設計文書', domainBody, index('domain')) : '',
    development
      ? group('開発文書', inGroup(documents, 'development-child').map(leaf).join(''), development)
      : '',
    operations ? group('運用文書', operationsBody, operations) : '',
    group(
      'リファレンス',
      `${referenceLink('api/index.html', 'API リファレンス')}${referenceLink('models/index.html', 'モデルカタログ')}${referenceLink('traceability/index.html', 'トレーサビリティ')}`,
      'reference/index.html',
    ),
    group('フォーマット', formats.map(leaf).join(''), 'format/index.html'),
  ].join('')
}

function breadcrumbs(page: string, current: string): string {
  return `<nav class="breadcrumbs" aria-label="パンくず">${siteLink(page, 'index.html', SITE_TITLE)}<span aria-hidden="true">/</span><span>${escapeHtml(current)}</span></nav>`
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
  const documentTitle = args.page === 'index.html' ? SITE_TITLE : `${args.title} · ${SITE_TITLE}`
  return `<!doctype html>
<html lang="ja"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>${escapeHtml(documentTitle)}</title><link rel="stylesheet" href="${escapeHtml(stylesheetHref(args.page))}">${args.head ?? ''}</head>
<body><a class="skip-link" href="#content">本文へ移動</a><header class="mobile-header">${siteLink(args.page, 'index.html', SITE_TITLE)}<details><summary>ナビゲーション</summary><nav aria-label="モバイル">${nav}</nav></details></header><aside class="sidebar"><div class="site-title">${siteLink(args.page, 'index.html', SITE_TITLE)}</div><nav aria-label="主要">${nav}</nav></aside><main id="content">${args.page === 'index.html' ? '' : breadcrumbs(args.page, args.current)}${args.body}</main>${args.scripts ?? ''}</body></html>\n`
}

function modelGroupId(context: string): string {
  return `context-${slug(context)}`
}

/** API リファレンスとモデルカタログは一つに保ち、宣言元の Context から該当箇所へ案内する。 */
function contextReference(args: {
  document: RenderedDocument
  tags: string[]
  operations: ApiOperation[]
  models: CatalogSymbol[]
}): string {
  const context = args.document.context
  if (!context || (args.operations.length === 0 && args.models.length === 0)) return ''
  const page = args.document.outputPath
  const id = `${args.document.id}-api-and-models`
  const filtered = (tag: string) =>
    `<a data-site-link href="${escapeHtml(`${pageHref(page, 'api/index.html')}?tag=${encodeURIComponent(tag)}`)}">${escapeHtml(tag)}</a>`
  const operations = args.operations.length
    ? `<h3 id="${id}-operations">API</h3><p class="muted">API リファレンスでは ${args.tags.map(filtered).join('、')} のタグで掲載する。</p><div class="table-wrap"><table><thead><tr><th scope="col">メソッド</th><th scope="col">パス</th><th scope="col">説明</th></tr></thead><tbody>${args.operations
        .map(
          (operation) =>
            `<tr><th scope="row"><code>${escapeHtml(operation.method)}</code></th><td><code>${escapeHtml(operation.path)}</code></td><td>${operation.summary || operation.description ? renderDoc(operation.summary ?? operation.description ?? '') : '<span class="muted">説明未記入</span>'}</td></tr>`,
        )
        .join('')}</tbody></table></div>`
    : ''
  const models = args.models.length
    ? `<h3 id="${id}-models">モデル</h3><p class="muted">${args.models.length} 個の TypeSpec シンボルを、モデルカタログの ${siteLink(page, 'models/index.html', context, modelGroupId(context))} にも掲載する。</p><ul class="symbol-links">${args.models
        .map((model) => `<li>${siteLink(page, modelPath(model), model.shortName)}</li>`)
        .join('')}</ul>`
    : ''
  return `<section class="context-reference"><h2 id="${id}">API とモデル</h2><p>この Context が宣言する TypeSpec から生成した情報である。</p>${operations}${models}</section>`
}

/**
 * 目次は組み上げた本文から作る。見出しの id を綴りから作り直すと、囲み記号の中の行や
 * 生成した節を取りこぼすので、出力に実在する見出しだけを並べる。
 */
function pageOutline(body: string): string {
  const headings = [...body.matchAll(/<h([23]) id="([^"]+)">([\s\S]*?)<\/h\1>/g)].map((match) => ({
    level: match[1] ?? '2',
    id: match[2] ?? '',
    // 見出しの中の `<code>` は目次では区別しない。本文は既に escape 済みである。
    text: (match[3] ?? '').replace(/<[^>]*>/g, '').trim(),
  }))
  // 見出しが一つしかないページの目次は、本文の書き出しを繰り返すだけである。
  if (headings.length < 2) return ''
  const items = headings
    .map(
      (heading) =>
        `<li class="page-toc-h${heading.level}"><a data-site-link href="#${escapeHtml(heading.id)}">${heading.text}</a></li>`,
    )
    .join('')
  return `<nav class="page-toc" aria-label="このページの内容"><p class="page-toc-title">このページの内容</p><ul>${items}</ul></nav>`
}

function documentPage(
  document: RenderedDocument,
  documents: RenderedDocument[],
  markdown: MarkdownItInstance,
  reference: string,
): string {
  const source = addDerivedStateDiagrams(
    stripFrontmatter(document.source),
    document.path.endsWith('/states.md'),
  )
  const article = `<article class="document">${markdown.render(source, { document })}${reference}</article>`
  const body = `<div class="page">${article}${pageOutline(article)}</div>`
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

/** フォーマット文書が何を定めるか。題名だけでは対象が分からない。 */
const FORMAT_SUMMARIES: Record<string, string> = {
  'DOCUMENTATION_GUIDE.md': '文書の種類、置き場所、それぞれが所有する内容',
  'SPECIFICATION_FORMAT.md': '規範シナリオと標準仕様の書き方、ID の付け方',
  'WORK_ITEM_FORMAT.md': '作業項目の書き方、必要な証拠、完了の記録',
}

function safeJson(value: unknown): string {
  return JSON.stringify(value)
    .replaceAll('<', '\\u003c')
    .replaceAll('>', '\\u003e')
    .replaceAll('&', '\\u0026')
    .replaceAll('\u2028', '\\u2028')
    .replaceAll('\u2029', '\\u2029')
}

/**
 * `#/components/schemas/...` のような内部参照であっても、Swagger UI 5.x は `baseDoc` の URI を
 * 取得して解決する。`baseDoc` は文書の URL から必ず組まれるので、`file://` で開いた生成物では
 * どの渡し方でも取得が拒まれ、参照が解決しない。
 *
 * 参照を残さなければ、解決そのものが起きない。組み立て時にすべての `$ref` を展開して渡す。
 * 展開した節には `$$ref` を残すので、Swagger UI は元のモデル名を表示できる。
 */
/**
 * `$ref` を展開した文書。Swagger UI へ渡すのはこれで、参照が残らないので解決の取得が起きない。
 * 自分自身へ戻る参照は展開できないため、そこだけ型だけの節へ置き換えて打ち切る。SCIM の
 * 属性定義が入れ子の属性を持つ 1 種類だけがこれに当たる。
 */
function dereference(openapi: OpenApiDocument): unknown {
  const schemas = (openapi.components?.schemas ?? {}) as Record<string, unknown>
  const SCHEMA_REF = /^#\/components\/schemas\/(.+)$/
  const expand = (node: unknown, open: readonly string[]): unknown => {
    if (Array.isArray(node)) return node.map((item) => expand(item, open))
    if (node === null || typeof node !== 'object') return node
    const entries = node as Record<string, unknown>
    const reference = typeof entries.$ref === 'string' ? entries.$ref.match(SCHEMA_REF) : null
    const name = reference?.[1]
    if (name === undefined) {
      return Object.fromEntries(
        Object.entries(entries).map(([key, value]) => [key, expand(value, open)]),
      )
    }
    const target = schemas[name]
    if (target === undefined) return entries
    if (open.includes(name)) {
      return { type: 'object', description: `${name} と同じ形の入れ子。` }
    }
    const expanded = expand(target, [...open, name])
    // Swagger UI は `$$ref` からモデル名を表示する。展開しても名前を失わない。
    return expanded !== null && typeof expanded === 'object' && !Array.isArray(expanded)
      ? { ...(expanded as Record<string, unknown>), $$ref: entries.$ref }
      : expanded
  }
  return expand(openapi, [])
}

function apiPage(
  openapi: OpenApiDocument,
  openapiFileName: string,
  documents: RenderedDocument[],
): string {
  const page = 'api/index.html'
  const body = `<article class="reference-page"><header class="reference-header"><p class="eyebrow">OpenAPI に基づくリファレンス</p><h1>API リファレンス</h1><p>生成した OpenAPI を Swagger UI で直接表示する。<code>?tag=</code> を指定すると、一つの Context の操作へ絞り込める。<a href="../openapi/${encodeURIComponent(openapiFileName)}">OpenAPI JSON を開く</a>。</p></header><div class="swagger-shell"><div id="swagger-ui" aria-label="API 操作"></div></div></article>`
  const head = `<link rel="stylesheet" href="${escapeHtml(assetHref(page, 'swagger-ui.css'))}">`
  // Swagger UI が実行時に作る表示文言は、DOM の更新後に既知の固定語だけを日本語へ置き換える。
  const swaggerTranslations = {
    'Filter by tag': 'タグで絞り込む',
    Authorize: '認証',
    Schemas: 'スキーマ',
    Parameters: 'パラメーター',
    Responses: 'レスポンス',
    'Response body': 'レスポンスボディ',
    'Response headers': 'レスポンスヘッダー',
    'Request body': 'リクエストボディ',
    'Request URL': 'リクエスト URL',
    'Server response': 'サーバーレスポンス',
    'No parameters': 'パラメーターなし',
    'Try it out': '試す',
    Execute: '実行',
    Clear: 'クリア',
    Cancel: 'キャンセル',
    Close: '閉じる',
    Code: 'コード',
    Details: '詳細',
    Example: '例',
    'Example Value': '値の例',
    Model: 'モデル',
    Schema: 'スキーマ',
  }
  const scripts = `<script src="${escapeHtml(assetHref(page, 'swagger-ui-bundle.js'))}"></script><script>window.addEventListener('DOMContentLoaded',function(){var tag=new URLSearchParams(window.location.search).get('tag');var specification=${safeJson(dereference(openapi))};var translations=${safeJson(swaggerTranslations)};var root=document.querySelector('#swagger-ui');var localize=function(){root.querySelectorAll('input').forEach(function(input){if(translations[input.placeholder])input.placeholder=translations[input.placeholder]});var walker=document.createTreeWalker(root,NodeFilter.SHOW_TEXT);var node;while(node=walker.nextNode()){var key=node.nodeValue.trim();if(translations[key])node.nodeValue=node.nodeValue.replace(key,translations[key])}};new MutationObserver(localize).observe(root,{childList:true,subtree:true});SwaggerUIBundle({spec:specification,dom_id:'#swagger-ui',deepLinking:true,displayRequestDuration:true,tryItOutEnabled:false,persistAuthorization:false,docExpansion:tag?'full':'list',defaultModelsExpandDepth:1,filter:tag||true,onComplete:localize});});</script>`
  return shell({
    page,
    title: 'API リファレンス',
    current: 'API リファレンス',
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
        `<tr><th scope="row"><code>${escapeHtml(property.name)}</code>${property.optional ? '<span class="optional">任意</span>' : '<span class="required">必須</span>'}</th><td><code>${escapeHtml(property.type)}</code>${property.default ? `<div class="meta">既定値: <code>${escapeHtml(property.default)}</code></div>` : ''}</td><td>${renderDoc(property.doc, propertyDescription(property))}${property.constraints.length ? `<ul class="compact">${property.constraints.map((constraint) => `<li><code>${escapeHtml(constraint)}</code></li>`).join('')}</ul>` : ''}${property.references.length ? `<div class="meta">参照: ${referenceList(page, property.references, symbols)}</div>` : ''}</td></tr>`,
    )
    .join('')
}

const commonPropertyDescriptions: Record<string, string> = {
  occurredAt: '事象が発生した日時。',
  created_at: 'レコードを作成した日時。',
  updated_at: 'レコードを最後に更新した日時。',
  issued_at: '資格情報または要求を発行した日時。',
  expires_at: '値が失効する日時。',
  disabled_at: '対象を無効化した日時。',
  revoked_at: '対象を失効させた日時。',
  tenantId: '対象テナントの識別子。',
  tenant_id: '対象テナントの識別子。',
  userId: '対象利用者の識別子。',
  user_id: '対象利用者の識別子。',
  actorUserId: '操作を実行した利用者の識別子。',
  targetUserId: '操作対象となる利用者の識別子。',
  clientId: 'OAuthクライアントの識別子。',
  client_id: 'OAuthクライアントの識別子。',
  applicationId: '対象アプリケーションの識別子。',
  application_id: '対象アプリケーションの識別子。',
  agentId: '対象エージェントの識別子。',
  agent_id: '対象エージェントの識別子。',
  groupId: '対象グループの識別子。',
  group_id: '対象グループの識別子。',
  sessionId: '対象セッションの識別子。',
  streamId: '対象ストリームの識別子。',
  workflowId: '対象ワークフローの識別子。',
  runId: 'ワークフロー実行の識別子。',
  jobId: '対象ジョブの識別子。',
  connectionId: '対象接続の識別子。',
  deliveryId: '対象配送の識別子。',
  exportId: '対象エクスポートの識別子。',
  id: 'このデータを一意に識別する値。',
  name: '利用者に表示する名前。',
  display_name: '利用者に表示する名前。',
  description: '対象の目的や内容を説明する文。',
  status: '現在の処理状態。',
  state: '現在の状態。',
  kind: '対象の種類。',
  type: '対象の型または分類。',
  reason: 'この結果になった理由。',
  email: '利用者のメールアドレス。',
  scopes: '許可する OAuth スコープの集合。',
  scope: '許可または要求する OAuth スコープ。',
  roles: '割り当てられたロールの集合。',
  attributes: '対象に付随する属性。',
  subject: '処理または資格情報の主体。',
  audience: '資格情報を受け取る対象。',
  issuer: '資格情報または表明の発行者。',
  code: '処理結果または手続きを識別するコード。',
  version: 'データまたは仕様のバージョン。',
  rules: '適用する規則の集合。',
  label: '画面に表示する短い名称。',
  mode: '処理方式。',
  protocol: '通信に使用するプロトコル。',
  realm: 'テナントを URL 上で識別するレルム。',
  jti: 'JWT を一意に識別する値。',
  nonce: '要求と応答を対応付ける一回限りの値。',
  schema: '値が従うスキーマ。',
  revision: '対象定義の改訂番号。',
  locale: '表示や通知に使用するロケール。',
}

function propertyDescription(property: CatalogProperty): string {
  const known = commonPropertyDescriptions[property.name]
  if (known) return known
  if (/(?:Id|_id)$/.test(property.name)) return `\`${property.name}\` が指す対象の識別子。`
  if (/(?:At|_at)$/.test(property.name)) return `\`${property.name}\` が表す出来事の日時。`
  if (/^(?:is|has|can|allow|enable|require|include)[A-Z_]/.test(property.name))
    return `\`${property.name}\` の条件を満たすかどうか。`
  return `このモデルで \`${property.name}\` が表す値。`
}

function scenarioIndex(documents: RenderedDocument[]): ScenarioEntry[] {
  const entries: ScenarioEntry[] = []
  for (const document of documents) {
    if (!document.path.endsWith('scenarios.feature.md')) continue
    for (const rule of parseScenarioDocument(document.source).rules) {
      entries.push({
        id: rule.id,
        title: rule.name.replace(new RegExp(`^${rule.id}(?::)?\\s*`), ''),
        document,
        anchor: `${document.id}-${slug(`Rule: ${rule.name}`)}`,
        kind: 'rule',
      })
      for (const example of rule.examples) {
        entries.push({
          id: example.id,
          title: example.name.replace(new RegExp(`^${example.id}\\s*`), ''),
          document,
          anchor: `${document.id}-${slug(`${example.outline ? 'Scenario Outline' : 'Example'}: ${example.name}`)}`,
          kind: 'example',
          parentId: rule.id,
        })
      }
    }
  }
  return entries.sort((a, b) => a.id.localeCompare(b.id))
}

type ScenarioEntry = {
  id: string
  title: string
  document: RenderedDocument
  anchor: string
  kind: 'rule' | 'example'
  parentId?: string
}

/**
 * 生成した区分の入口。中身は下位のページが持つので、この段は何がどこにあるかだけを言う。
 * 入口が無いと、サイドバーの区分名だけがクリックできない例外になる。
 */
function divisionIndex(args: {
  page: string
  title: string
  lead: string
  entries: { path: string; label: string; content: string }[]
  documents: RenderedDocument[]
}): string {
  const rows = args.entries
    .map(
      (entry) =>
        `<tr><th scope="row">${siteLink(args.page, entry.path, entry.label)}</th><td>${escapeHtml(entry.content)}</td></tr>`,
    )
    .join('')
  const body = `<article class="document"><h1>${escapeHtml(args.title)}</h1><p>${escapeHtml(args.lead)}</p><div class="table-wrap"><table><thead><tr><th scope="col">ページ</th><th scope="col">示すもの</th></tr></thead><tbody>${rows}</tbody></table></div></article>`
  return shell({
    page: args.page,
    title: args.title,
    current: args.title,
    body,
    documents: args.documents,
  })
}

function traceabilityPage(documents: RenderedDocument[], traces: ScenarioTrace[]): string {
  const page = 'traceability/index.html'
  const byId = new Map(traces.map((trace) => [trace.id, trace]))
  const scenarios = scenarioIndex(documents)
  const examples = scenarios.filter((scenario) => scenario.kind === 'example')
  const covered = examples.filter((scenario) => (byId.get(scenario.id)?.sources.length ?? 0) > 0)
  const paths = (label: string, entries: string[]) =>
    entries.length === 0
      ? `<span class="trace-empty">${escapeHtml(label)}なし</span>`
      : `<ul class="trace-paths">${entries
          .map((entry) => `<li><code>${escapeHtml(entry)}</code></li>`)
          .join('')}</ul>`
  const rows = scenarios
    .map((scenario) => {
      const trace = byId.get(scenario.id)
      const href = `${pageHref(page, scenario.document.outputPath)}#${scenario.anchor}`
      const debt = trace?.debt
        ? `<span class="trace-debt">負債: ${escapeHtml(trace.debt)}</span>`
        : '<span class="trace-empty">負債なし</span>'
      return `<tr class="trace-${scenario.kind}"><th scope="row"><a data-site-link href="${escapeHtml(href)}">${escapeHtml(scenario.id)}</a><span class="trace-title">${escapeHtml(scenario.title)}</span>${scenario.parentId ? `<span class="trace-parent">${escapeHtml(scenario.parentId)}</span>` : ''}</th><td>${paths('テスト参照', trace?.sources ?? [])}${scenario.kind === 'example' ? debt : ''}</td><td>${paths('作業項目', trace?.workItems ?? [])}</td></tr>`
    })
    .join('')
  const body = `<header class="reference-header"><p class="eyebrow">リポジトリから生成</p><h1>トレーサビリティ</h1><p>すべての規範的な規則と実行可能な例を示す。例へのテスト対応は EX 識別子を名指しするプロダクトテストだけから算出し、テスト未対応の場合は移行負債の理由を示す。${examples.length} 件中 ${covered.length} 件の例がテスト参照を持つ。</p></header><table class="trace-table"><thead><tr><th scope="col">規則／例</th><th scope="col">テストまたは負債</th><th scope="col">作業項目</th></tr></thead><tbody>${rows}</tbody></table>`
  return shell({ page, title: 'トレーサビリティ', current: 'トレーサビリティ', body, documents })
}

/** モデルは一つの契約名前空間を共有するため、宣言元ディレクトリで分類する。 */
function modelGroups(
  models: CatalogSymbol[],
  documents: RenderedDocument[],
): Array<{ id: string; title: string; entries: CatalogSymbol[] }> {
  const groups: Array<{ id: string; title: string; entries: CatalogSymbol[] }> = []
  const owners = new Set<string>()
  for (const document of inGroup(documents, 'context')) {
    const context = document.context
    if (!context) continue
    owners.add(context)
    const entries = models.filter((model) => model.context === context)
    if (entries.length) groups.push({ id: modelGroupId(context), title: document.title, entries })
  }
  const rest = new Map<string, CatalogSymbol[]>()
  for (const model of models) {
    if (model.context && owners.has(model.context)) continue
    const entries = rest.get(model.namespace) ?? []
    entries.push(model)
    rest.set(model.namespace, entries)
  }
  for (const [namespace, entries] of [...rest.entries()].sort(([a], [b]) => a.localeCompare(b)))
    groups.push({ id: `namespace-${slug(namespace)}`, title: namespace, entries })
  return groups
}

function modelIndex(models: CatalogSymbol[], documents: RenderedDocument[]): string {
  const page = 'models/index.html'
  const body = `<header class="reference-header"><p class="eyebrow">TypeSpec プログラム</p><h1>モデルカタログ</h1><p>リポジトリが所有するモデル、列挙、共用体、スカラーを、HTTP 操作への公開有無にかかわらず宣言元の Bounded Context ごとに掲載する。<code>Operations</code> 名前空間の転送用ラッパーは OpenAPI に基づく API リファレンスへ掲載する。</p><label class="model-search">モデルを絞り込む <input type="search" data-model-search placeholder="名前、Context、説明" autocomplete="off"></label></header>${modelGroups(
    models,
    documents,
  )
    .map(
      (group) =>
        `<section class="model-group" data-model-group><h2 id="${escapeHtml(group.id)}">${escapeHtml(group.title)}</h2><div class="model-list">${group.entries
          .map(
            (entry) =>
              `<article data-model-card data-search="${escapeHtml(`${entry.name} ${group.title} ${entry.doc ?? ''}`.toLowerCase())}"><div><span class="kind">${escapeHtml(entry.kind)}</span>${entry.apiExposed ? '<span class="api-exposed">API 公開</span>' : ''}</div><h3>${siteLink(page, modelPath(entry), entry.shortName)}</h3><p>${renderDoc(entry.doc, `\`${entry.shortName}\` が表すデータ構造。`)}</p></article>`,
          )
          .join('')}</div></section>`,
    )
    .join('')}`
  return shell({
    page,
    title: 'モデルカタログ',
    current: 'モデルカタログ',
    body,
    documents,
    // 絞り込み欄を使うページだけにサイトスクリプトを読み込む。
    scripts: `<script src="${escapeHtml(assetHref(page, 'site.js'))}"></script>`,
  })
}

function modelPage(
  model: CatalogSymbol,
  symbols: Map<string, CatalogSymbol>,
  documents: RenderedDocument[],
): string {
  const page = modelPath(model)
  const exposure = model.apiExposed
    ? '<span class="api-exposed">API 公開</span>'
    : '<span class="not-exposed">API 非公開</span>'
  const properties = model.properties.length
    ? `<section><h2>プロパティ</h2><div class="table-wrap"><table><thead><tr><th>名前</th><th>型</th><th>説明と制約</th></tr></thead><tbody>${propertyRows(page, model.properties, symbols)}</tbody></table></div></section>`
    : ''
  const members = model.members.length
    ? `<section><h2>${model.kind === 'union' ? 'バリアント' : 'メンバー'}</h2><div class="table-wrap"><table><thead><tr><th>名前</th><th>値または型</th><th>説明</th></tr></thead><tbody>${model.members
        .map(
          (member) =>
            `<tr><th scope="row"><code>${escapeHtml(member.name)}</code></th><td><code>${escapeHtml(member.value ?? member.type ?? '—')}</code></td><td>${renderDoc(member.doc, `\`${member.name}\` が表す選択肢。`)}</td></tr>`,
        )
        .join('')}</tbody></table></div></section>`
    : ''
  const body = `<article class="model-detail"><header><p class="eyebrow">${escapeHtml(model.namespace)} · ${escapeHtml(model.kind)}</p><h1>${escapeHtml(model.shortName)}</h1><div class="badges">${exposure}</div><p>${renderDoc(model.doc, `\`${model.shortName}\` が表すデータ構造。`)}</p><p class="qualified"><strong>TypeSpec シンボル</strong> <code>${escapeHtml(model.name)}</code></p>${model.base ? `<p><strong>基底</strong> ${modelLink(page, model.base, symbols)}</p>` : ''}</header>${properties}${members}<section><h2>参照</h2><p>${referenceList(page, model.references, symbols)}</p></section></article>`
  return shell({
    page,
    title: model.shortName,
    current: model.shortName,
    body,
    documents,
  })
}

type ApiOperation = {
  method: string
  path: string
  summary?: string
  description?: string
  tags: string[]
}

function inspectOpenApi(openapi: OpenApiDocument): { operations: ApiOperation[]; tags: string[] } {
  const operations: ApiOperation[] = []
  const tags = new Set<string>()
  for (const [path, pathItem] of Object.entries(openapi.paths ?? {})) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!HTTP_METHODS.has(method.toLowerCase())) continue
      const owners = operation.tags ?? []
      if (owners.length === 0 || owners.includes('default'))
        throw new Error(`${method.toUpperCase()} ${path} has no owning context tag`)
      for (const tag of owners) tags.add(tag)
      operations.push({
        method: method.toUpperCase(),
        path,
        summary: operation.summary,
        description: operation.description,
        tags: owners,
      })
    }
  }
  operations.sort((a, b) => a.path.localeCompare(b.path) || a.method.localeCompare(b.method))
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
      const [locator, fragment] = href.split('#', 2)
      // A query selects a view of the target page, not another page.
      const pathPart = locator?.split('?', 1)[0]
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
:root{color-scheme:light dark;--bg:#fff;--bg-soft:#f6f7f9;--text:#1c2024;--muted:#5c646f;--line:#e3e6ea;--line-strong:#c8ced7;--accent:#1f4fd8;--accent-soft:#eef2ff;--code:#f4f5f7;--given:#0f6b84;--when:#8a5200;--then:#0f6b45;--diagram-line:#294cba;--measure:760px;--sidebar:290px;--toc:216px}
@media(prefers-color-scheme:dark){:root{--bg:#14171c;--bg-soft:#1a1e25;--text:#e7ebf1;--muted:#9aa3b0;--line:#282e38;--line-strong:#3b4350;--accent:#93adff;--accent-soft:#1e2740;--code:#1e232b;--given:#7bd6f0;--when:#ffc36b;--then:#74d6a0;--diagram-line:#b9c8ff}}
*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;color:var(--text);background:var(--bg);font:16px/1.75 Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;overflow-wrap:anywhere}.skip-link{position:fixed;z-index:20;top:8px;left:8px;transform:translateY(-160%);padding:8px 12px;background:var(--bg);border:2px solid var(--accent);border-radius:8px}.skip-link:focus{transform:none}
.sidebar{position:fixed;inset:0 auto 0 0;width:var(--sidebar);overflow:auto;padding:26px 14px 64px;border-right:1px solid var(--line);background:var(--bg-soft)}.site-title{margin:0 10px 22px;font-size:16px;font-weight:700;letter-spacing:-.01em}.site-title a{color:inherit;text-decoration:none}
.nav-section{margin:0 0 4px}.nav-section>summary{padding:7px 10px;color:var(--muted);font-size:12px;font-weight:700;letter-spacing:.08em;cursor:pointer;list-style:none}.nav-section>summary::-webkit-details-marker{display:none}.nav-section>summary::before{content:"▸";display:inline-block;width:14px;color:var(--line-strong)}.nav-section[open]>summary::before{content:"▾"}.nav-section-link{color:inherit;text-decoration:none}.nav-section-link:hover,.nav-section-link[aria-current=page]{color:var(--accent)}
.nav-tree{margin:0 0 10px;padding:0;list-style:none}.nav-tree ul{margin:0;padding:0 0 0 18px;list-style:none}.nav-item,.nav-branch{margin:0}.nav-link,.nav-label{display:block;padding:4px 10px;border-left:2px solid transparent;color:var(--text);font-size:13.5px;line-height:1.5;text-decoration:none}.nav-label{cursor:pointer}.nav-link:hover{background:var(--accent-soft)}.nav-link[aria-current=page]{border-left-color:var(--accent);color:var(--accent);background:var(--accent-soft);font-weight:700}
.nav-directory>summary{display:flex;align-items:flex-start;cursor:pointer;list-style:none}.nav-directory>summary::-webkit-details-marker{display:none}.nav-directory>summary::before{content:"▸";flex:none;width:16px;padding:4px 0;color:var(--line-strong);font-size:10px;line-height:1.9;text-align:center}.nav-directory[open]>summary::before{content:"▾"}.nav-directory>summary>.nav-link,.nav-directory>summary>.nav-label{flex:1;min-width:0}.nav-item>.nav-link{padding-left:26px}
main{margin-left:var(--sidebar);padding:34px 40px 140px}main:has(.swagger-shell){max-width:none;padding-inline:26px}.breadcrumbs{display:flex;gap:8px;align-items:center;margin:0 0 26px;color:var(--muted);font-size:13px}.mobile-header{display:none}
.page{display:grid;grid-template-columns:minmax(0,var(--measure)) minmax(0,var(--toc));gap:56px;align-items:start}.landing{max-width:880px}
.page-toc{position:sticky;top:34px;max-height:calc(100vh - 80px);overflow:auto;padding-left:16px;border-left:1px solid var(--line);font-size:13px;line-height:1.5}.page-toc-title{margin:0 0 8px;color:var(--muted);font-weight:700}.page-toc ul{margin:0;padding:0;list-style:none}.page-toc a{display:block;padding:4px 0;color:var(--muted);text-decoration:none}.page-toc a:hover{color:var(--text)}.page-toc a[aria-current=location]{color:var(--accent)}.page-toc-h3 a{padding-left:14px;font-size:12.5px}
h1,h2,h3,h4{line-height:1.35;letter-spacing:-.012em;scroll-margin-top:26px}h1{margin:0 0 .7em;font-size:30px}h2{margin:2.4em 0 .7em;font-size:21px}h3{margin:1.9em 0 .5em;font-size:17px}h4{margin:1.6em 0 .4em;font-size:15px}p,ul,ol{margin:0 0 1.1em}
a{color:var(--accent);text-underline-offset:2px}a:focus-visible,summary:focus-visible,input:focus-visible{outline:3px solid var(--accent);outline-offset:3px;border-radius:4px}
code{padding:.1em .34em;border:1px solid var(--line);border-radius:4px;background:var(--code);font-size:.87em}pre{max-width:100%;overflow:auto;margin:1.4em 0;padding:16px 18px;border:1px solid var(--line);border-radius:8px;background:var(--code)}pre code{padding:0;border:0;background:none;font-size:.86em}
.hero{margin:0 0 44px;padding:0 0 30px;border-bottom:1px solid var(--line)}.hero h1{font-size:34px}.hero p{color:var(--muted);font-size:17px}.hero-links{display:flex;flex-wrap:wrap;gap:20px;margin:0;font-size:15px;font-weight:600}
.context-links{display:flex;flex-wrap:wrap;gap:8px 18px;margin:1em 0;padding:0;list-style:none;font-size:14px}
.table-wrap,table{max-width:100%;overflow:auto}table{width:100%;margin:1.4em 0;border-collapse:collapse;display:block;font-size:14.5px;line-height:1.65}th,td{padding:9px 14px 9px 0;border:0;border-bottom:1px solid var(--line);text-align:left;vertical-align:top;overflow-wrap:break-word}thead th{padding-left:10px;border-bottom:1px solid var(--line-strong);background:var(--bg-soft);font-size:13px}tbody th[scope=row]{font-weight:600}.term-table td:first-child{white-space:nowrap}
.diagram-shell{max-width:100%;overflow:auto;margin:1.6em 0;padding:18px;border:1px solid var(--line);border-radius:10px;background:var(--bg-soft)}.diagram-shell .mermaid{min-width:560px;background:transparent}.diagram-shell .mermaid svg .edgePath path,.diagram-shell .mermaid svg .flowchart-link,.diagram-shell .mermaid svg .transition{stroke:var(--diagram-line)!important;stroke-width:2.4px!important}.diagram-shell .mermaid svg marker path{fill:var(--diagram-line)!important;stroke:var(--diagram-line)!important}
.scenario-keyword{display:inline-block;min-width:58px;margin-right:5px;padding:1px 7px;border:1px solid currentColor;border-radius:999px;font-size:11px;font-weight:800;letter-spacing:.04em;text-align:center}.scenario-keyword.given,.scenario-keyword.and{color:var(--given)}.scenario-keyword.when,.scenario-keyword.but{color:var(--when)}.scenario-keyword.then{color:var(--then)}li:has(>.scenario-keyword){margin:.45em 0}.scenario-actor{display:inline-block;margin-right:6px;padding:1px 9px;border:1px dashed currentColor;border-radius:999px;color:var(--muted);font-size:11px;font-weight:700;letter-spacing:.04em}p:has(>.scenario-actor){margin:.35em 0 .9em}
.reference-header{margin-bottom:26px}.reference-page{max-width:none}.swagger-shell{color-scheme:light;margin:24px 0 0;padding:20px;overflow:auto;border:1px solid var(--line);border-radius:10px;background:#fff;color:#3b4151}.swagger-shell .swagger-ui .wrapper{max-width:none;padding-inline:0}
.model-group{margin-top:34px}.model-list{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:12px}.model-list article{padding:16px;border:1px solid var(--line);border-radius:10px;background:var(--bg-soft)}.model-list h3{margin:.4em 0}.model-list p{color:var(--muted)}.model-search{display:grid;max-width:520px;gap:6px;margin-top:22px;font-weight:700}.model-search input{width:100%;padding:10px 12px;color:var(--text);background:var(--bg);border:1px solid var(--line-strong);border-radius:8px;font:inherit}
.trace-table th[scope=row]{display:grid;gap:2px;text-align:left;vertical-align:top}.trace-example th[scope=row]{padding-left:28px}.trace-title,.trace-parent{color:var(--muted);font-weight:400}.trace-parent,.trace-debt{font-size:12px}.trace-debt{display:block;margin-top:6px;color:var(--muted)}.trace-paths{margin:0;padding-left:16px}.trace-paths code{font-size:12px}.trace-empty{color:var(--muted)}
.kind,.api-exposed,.not-exposed,.required,.optional{display:inline-block;margin:0 6px 4px 0;padding:2px 7px;border-radius:999px;font-size:11px;font-weight:800}.kind,.optional{color:var(--muted);background:var(--code)}.api-exposed,.required{color:#fff;background:#28664b}.not-exposed{color:var(--muted);border:1px solid var(--line)}.qualified{padding:12px;border-radius:8px;background:var(--bg-soft)}.badges{margin:.5em 0}.context-reference{margin-top:44px;padding-top:10px;border-top:1px solid var(--line)}.symbol-links{display:flex;flex-wrap:wrap;gap:6px 14px;margin:.6em 0;padding:0;list-style:none}.meta{margin-top:7px;color:var(--muted);font-size:13px}.compact{margin:.5em 0;padding-left:20px}.muted{color:var(--muted)}[hidden]{display:none!important}
@media(max-width:1200px){.page{grid-template-columns:minmax(0,1fr)}.page-toc{display:none}}
@media(max-width:900px){.sidebar{display:none}.mobile-header{display:flex;position:sticky;z-index:10;top:0;justify-content:space-between;align-items:flex-start;padding:12px 18px;border-bottom:1px solid var(--line);background:var(--bg)}.mobile-header>details{position:relative}.mobile-header details>nav{position:absolute;right:0;width:min(86vw,320px);max-height:75vh;overflow:auto;padding:12px;border:1px solid var(--line);border-radius:10px;background:var(--bg);box-shadow:0 12px 32px rgba(20,23,28,.18)}main{margin:0;padding:20px 18px 80px}main:has(.swagger-shell){padding-inline:18px}.hero h1{font-size:28px}.diagram-shell .mermaid{min-width:480px}}
@media print{.sidebar,.mobile-header,.breadcrumbs,.skip-link,.page-toc{display:none}main{margin:0;padding:0}.page{display:block}a{color:inherit;text-decoration:none}}
`

const siteScript = `
window.addEventListener('DOMContentLoaded',function(){
  if(window.mermaid&&document.querySelector('.mermaid')){
    var dark=window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches;
    window.mermaid.initialize({startOnLoad:false,securityLevel:'strict',layout:'dagre',look:'classic',theme:'base',themeVariables:dark?{background:'#1a1e25',primaryColor:'#1e2740',primaryTextColor:'#e7ebf1',primaryBorderColor:'#b9c8ff',lineColor:'#b9c8ff',textColor:'#e7ebf1',edgeLabelBackground:'#14171c',tertiaryColor:'#1a1e25'}:{background:'#f6f7f9',primaryColor:'#eef2ff',primaryTextColor:'#1c2024',primaryBorderColor:'#294cba',lineColor:'#294cba',textColor:'#1c2024',edgeLabelBackground:'#fff',tertiaryColor:'#f6f7f9'}});
    window.mermaid.run({querySelector:'.mermaid'});
  }
  var search=document.querySelector('[data-model-search]');
  if(search){search.addEventListener('input',function(){
    var query=search.value.trim().toLowerCase();
    document.querySelectorAll('[data-model-card]').forEach(function(card){card.hidden=Boolean(query&&!card.dataset.search.includes(query));});
    document.querySelectorAll('[data-model-group]').forEach(function(group){group.hidden=group.querySelectorAll('[data-model-card]:not([hidden])').length===0;});
  });}
  var toc=document.querySelector('.page-toc');
  if(toc&&window.IntersectionObserver){
    var order=[];var links={};var visible={};
    toc.querySelectorAll('a[href^="#"]').forEach(function(anchor){
      var id=anchor.getAttribute('href').slice(1);
      if(document.getElementById(id)){order.push(id);links[id]=anchor;}
    });
    var mark=function(){
      var current=order.filter(function(id){return visible[id];})[0];
      order.forEach(function(id){
        if(id===current)links[id].setAttribute('aria-current','location');
        else links[id].removeAttribute('aria-current');
      });
    };
    var observer=new IntersectionObserver(function(entries){
      entries.forEach(function(entry){visible[entry.target.id]=entry.isIntersecting;});
      mark();
    },{rootMargin:'-80px 0px -70% 0px'});
    order.forEach(function(id){observer.observe(document.getElementById(id));});
  }
});
`

export function renderDocumentationSite(args: {
  documents: SourceDocument[]
  openapi: OpenApiDocument
  repositoryRoot: string
  outputDirectory: string
  openapiFileName: string
  models: CatalogSymbol[]
  traces?: ScenarioTrace[]
  contextTags?: Record<string, string[]>
}): RenderedDocumentationSite {
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
  const formats = inGroup(documents, 'format')
  const files: Record<string, string> = {
    'api/index.html': apiPage(args.openapi, args.openapiFileName, documents),
    'models/index.html': modelIndex(args.models, documents),
    'traceability/index.html': traceabilityPage(documents, args.traces ?? []),
    'reference/index.html': divisionIndex({
      page: 'reference/index.html',
      title: 'リファレンス',
      lead: 'TypeSpec と規範シナリオから生成した参照である。人が書く文書ではないので、内容を直すには生成元を直す。',
      entries: [
        {
          path: 'api/index.html',
          label: 'API リファレンス',
          content: '公開する HTTP 操作、要求と応答の形、認証方式',
        },
        {
          path: 'models/index.html',
          label: 'モデルカタログ',
          content: `HTTP に公開しないものを含む、リポジトリ所有の TypeSpec シンボル ${args.models.length} 個`,
        },
        {
          path: 'traceability/index.html',
          label: 'トレーサビリティ',
          content: '規範 ID と、それを名指すテスト、実装、作業項目の対応',
        },
      ],
      documents,
    }),
    'format/index.html': divisionIndex({
      page: 'format/index.html',
      title: 'フォーマット',
      lead: '文書と記録の書き方を定める。プロダクトの振る舞いではなく、このリポジトリの進め方が対象である。',
      entries: formats.map((entry) => ({
        path: entry.outputPath,
        label: entry.title,
        content: FORMAT_SUMMARIES[entry.path] ?? '書き方の規則',
      })),
      documents,
    }),
  }
  for (const document of documents) {
    const tags = (document.context ? args.contextTags?.[document.context] : undefined) ?? []
    const reference =
      document.category === 'context'
        ? contextReference({
            document,
            tags,
            operations: openapi.operations.filter((operation) =>
              operation.tags.some((tag) => tags.includes(tag)),
            ),
            models: args.models.filter((model) => model.context === document.context),
          })
        : ''
    files[document.outputPath] = documentPage(document, documents, markdown, reference)
  }
  for (const model of args.models) files[modelPath(model)] = modelPage(model, symbols, documents)
  validateSiteLinks(files)
  return {
    files,
    assets: { 'site.css': styles, 'site.js': siteScript },
    operations: openapi.operations.length,
    tags: openapi.tags,
    models: args.models.length,
    mermaidSources,
  }
}

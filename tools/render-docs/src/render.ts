import { dirname, posix, relative, resolve } from 'node:path'
import MarkdownIt, { type MarkdownIt as MarkdownItInstance } from 'markdown-it'
import { specificationRules } from '../../check/src/feature-specification.ts'
import { ruleBodies } from '../../check/src/gherkin-scenarios.ts'
import { documentKind } from '../../check/src/specification-doc.ts'
import { CONTEXT_DOCUMENTS, SYSTEM_DOCUMENT_PATHS } from '../../workspace/src/document-layout.ts'
import type { CatalogProperty, CatalogSymbol } from './typespec-catalog.ts'

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
  /** 機能スライスの文書だけが持つ、所属する機能スライスのディレクトリ名。 */
  feature?: string
}

type NavigationDirectory = {
  name: string
  document?: RenderedDocument
  documents: RenderedDocument[]
  directories: NavigationDirectory[]
}

type DocumentCategory =
  | 'format'
  | 'format-index'
  | 'modules'
  | 'development'
  | 'development-child'
  | 'operations'
  | 'operations-child'
  | 'runbook'
  | 'whole-system'
  | 'whole-system-child'
  | 'context'
  | 'context-child'
  | 'feature'
  | 'feature-child'

export type RenderedDocumentationSite = {
  files: Record<string, string>
  assets: Record<string, string>
  operations: number
  tags: string[]
  models: number
  mermaidSources: string[]
}

/** 用語表の 1 列目の見出し。日本語の文書と、まだ英語のモジュール文書の両方を含む。 */
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

/**
 * 機能スライスの子の並び。機能仕様の章を先に、内部設計と例の付録を後に置く。
 * 章は仕様の続きなので、仕様の直後に読めるようにする。
 */
function featureChildOrder(name: string, fallback: number): number {
  if (name === 'design.md') return Number.MAX_SAFE_INTEGER - 1
  if (name === 'acceptance.feature.md') return Number.MAX_SAFE_INTEGER
  return fallback
}

/**
 * 文書のページの位置。`docs/` の Markdown はリポジトリと同じ相対パスに置き、URL から元の
 * 文書を辿れるようにする。入口の `docs/README.md` だけはサイトの最上位に置く。
 */
function sitePath(path: string): string {
  if (path === 'docs/README.md') return 'index.html'
  const segments = path.split('/')
  const file = segments.pop() ?? ''
  const directory = segments.map(slug).join('/')
  if (file === 'README.md') return `${directory}/index.html`
  return `${directory}/${slug(file.replace(/(?:\.feature)?\.md$/, ''))}.html`
}

function documentMetadata(document: SourceDocument, index: number): RenderedDocument {
  const declaredTitle = plainTitle(document.source.match(/^# (.+)$/m)?.[1]?.trim() ?? document.path)
  const title =
    document.path === 'docs/requirements/scenarios.feature.md'
      ? 'システム横断シナリオ'
      : declaredTitle
  const sections = [...document.source.matchAll(/^## (.+)$/gm)].map(
    (match) => match[1]?.trim() ?? '',
  )
  const base = { ...document, title, sections, outputPath: sitePath(document.path) }
  const file = document.path.split('/').at(-1) ?? ''
  const stem = file.replace(/(?:\.feature)?\.md$/, '')
  const isIndex = file === 'README.md'
  if (document.path === 'docs/README.md') {
    return { ...base, id: 'whole-system', category: 'whole-system', order: 0 }
  }
  if (document.path.startsWith('docs/formats/')) {
    return {
      ...base,
      id: `format-${slug(stem)}`,
      category: isIndex ? 'format-index' : 'format',
      order: canonicalOrder(SYSTEM_DOCUMENT_PATHS, document.path, index),
    }
  }
  if (/^docs\/operations\/[^/]+$/.test(document.path)) {
    return isIndex
      ? { ...base, id: 'operations', category: 'operations', order: 0 }
      : {
          ...base,
          id: `operations-${slug(stem)}`,
          category: 'operations-child',
          order: canonicalOrder(SYSTEM_DOCUMENT_PATHS, document.path, index),
        }
  }
  if (/^docs\/runbooks\/[^/]+$/.test(document.path)) {
    return { ...base, id: `runbook-${slug(stem)}`, category: 'runbook', order: index }
  }
  if (/^docs\/development\/[^/]+$/.test(document.path)) {
    return isIndex
      ? { ...base, id: 'development', category: 'development', order: 0 }
      : { ...base, id: `development-${slug(stem)}`, category: 'development-child', order: index }
  }
  if (document.path === 'docs/modules/README.md') {
    return { ...base, id: 'modules', category: 'modules', order: 0 }
  }
  // モジュールより下の段。内部設計、機能群、機能群の下の機能スライスがある。
  // `feature` はその段のモジュールからの相対パスである。
  const featureDocument = document.path.match(/^docs\/modules\/([^/]+)\/(.+)\/([^/]+)$/)
  if (featureDocument) {
    const [, context = '', feature = '', featureFile = ''] = featureDocument
    const node = feature.split('/').map(slug).join('-')
    return featureFile === 'README.md'
      ? {
          ...base,
          id: `context-${context}-${node}`,
          category: 'feature',
          order: index,
          context,
          feature,
        }
      : {
          ...base,
          id: `context-${context}-${node}-${slug(stem)}`,
          category: 'feature-child',
          order: featureChildOrder(featureFile, index),
          context,
          feature,
        }
  }
  const contextDocument = document.path.match(/^docs\/modules\/([^/]+)\/([^/]+)$/)
  const context = contextDocument?.[1]
  const contextFile = contextDocument?.[2]
  if (context && contextFile) {
    return contextFile === 'README.md'
      ? { ...base, id: `context-${context}`, category: 'context', order: index, context }
      : {
          ...base,
          id: `context-${context}-${slug(stem)}`,
          category: 'context-child',
          order: canonicalOrder(CONTEXT_DOCUMENTS, contextFile, index),
          context,
        }
  }
  return {
    ...base,
    id: `whole-system-${slug(document.path.replace(/^docs\//, ''))}`,
    category: 'whole-system-child',
    order: canonicalOrder(SYSTEM_DOCUMENT_PATHS, document.path, index),
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

/** 機能仕様の状態機械は、`## 状態遷移` の節の下の H3 である。 */
export function addDerivedStateDiagrams(source: string): string {
  const lines = source.split('\n')
  const insertions = new Map<number, string[]>()
  let inStates = false
  for (let index = 0; index < lines.length; index++) {
    const line = lines[index] ?? ''
    if (line === '## 状態遷移') {
      inStates = true
      continue
    }
    if (line.startsWith('## ')) inStates = false
    if (!inStates || !line.startsWith('### ')) continue
    const machine = line.slice(4).trim()
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

/**
 * 文書が一次情報へ張ったリンクのうち、生成サイトでは生成したビューを開くもの。
 * リポジトリで読む人は TypeSpec へ、サイトで読む人はそこから作ったリファレンスへ着く。
 */
const GENERATED_VIEWS = new Map([['spec/main.tsp', 'reference/index.html']])

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
      /(?:scenarios|acceptance)\.feature\.md$/.test(
        (env as { document: RenderedDocument }).document.path,
      )
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
    return `<div class="diagram-shell"><button type="button" class="diagram-zoom" data-diagram-zoom>拡大表示</button><pre class="mermaid">${escapeHtml(source)}</pre></div>\n`
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
        const generatedView = GENERATED_VIEWS.get(relative(repositoryRoot, absolute))
        if (generatedView) {
          token?.attrSet('href', pageHref(current.document.outputPath, generatedView))
          token?.attrSet('data-site-link', '')
        } else if (target) {
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
  if (entry.category === 'runbook') return entry.title.replace(/の運用手順書$/, '')
  // 入れ子の段そのものが所属を示す段では、名札は題名だけでよい。
  if (entry.category !== 'context-child' && entry.category !== 'feature-child') return entry.title
  if (entry.path.endsWith('acceptance.feature.md')) return '例'
  const owner = documents.find((document) =>
    entry.category === 'feature-child'
      ? document.category === 'feature' &&
        document.context === entry.context &&
        document.feature === entry.feature
      : document.category === 'context' && document.context === entry.context,
  )
  if (!owner) return entry.title
  // 内部設計の文書は「Demo の重要な設計判断」のように、段ではなくモジュールの名前を冠する。
  const context = documents.find(
    (document) => document.category === 'context' && document.context === entry.context,
  )
  for (const prefix of [owner.title, context?.title]) {
    if (!prefix) continue
    // 英字の名前には空白を挟んで「の」を続け、日本語の名前には直接続ける。
    for (const possessive of [`${prefix} の`, `${prefix}の`, `${prefix} `]) {
      if (entry.title.startsWith(possessive)) return entry.title.slice(possessive.length)
    }
  }
  return entry.title
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
      entry.category === 'context-child' || entry.category === 'feature-child'
        ? ' nav-context-child'
        : '',
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
  /**
   * 区分は `docs/README.md` の表と同じ名前と順序で並べる。要件文書と全体設計文書は、
   * それぞれのディレクトリの索引を区分の見出しにし、設計領域は全体設計文書の直下に置く。
   */
  const tree = documentTree(rootChildren)
  const division = (name: string) => tree.directories.find((node) => node.name === name)
  const divisionBody = (node: NavigationDirectory | undefined) =>
    node ? [...node.documents.map(leaf), ...node.directories.map(directory)].join('') : ''
  const requirementsNode = division('requirements')
  const designNode = division('design')
  // 段は入れ子にする。機能スライスが機能群の下に、内部設計の文書が
  // `design` の下に並ぶ。親の段は、その段の相対パスを接頭辞に持つ段を子に持つ。
  const node = (feature: RenderedDocument): NavigationDirectory => ({
    name: feature.title,
    document: feature,
    documents: inGroup(documents, 'feature-child').filter(
      (document) => document.context === feature.context && document.feature === feature.feature,
    ),
    directories: inGroup(documents, 'feature')
      .filter(
        (child) =>
          child.context === feature.context &&
          child.feature?.startsWith(`${feature.feature}/`) &&
          !child.feature.slice((feature.feature?.length ?? 0) + 1).includes('/'),
      )
      .map(node),
  })
  const modulesBody = [
    ...contexts.map((entry) =>
      directory({
        name: entry.title,
        document: entry,
        documents: inGroup(documents, 'context-child').filter(
          (document) => document.context === entry.context,
        ),
        directories: inGroup(documents, 'feature')
          .filter((feature) => feature.context === entry.context && !feature.feature?.includes('/'))
          .map(node),
      }),
    ),
  ].join('')
  const operationsBody = [
    ...inGroup(documents, 'operations-child').map(leaf),
    ...(runbooks.length
      ? [directory({ name: '運用手順', documents: runbooks, directories: [] })]
      : []),
  ].join('')
  return [
    requirementsNode
      ? group('要件文書', divisionBody(requirementsNode), requirementsNode.document)
      : '',
    designNode ? group('全体設計文書', divisionBody(designNode), designNode.document) : '',
    modulesBody ? group('モジュール設計文書', modulesBody, index('modules')) : '',
    development
      ? group('開発文書', inGroup(documents, 'development-child').map(leaf).join(''), development)
      : '',
    operations ? group('運用文書', operationsBody, operations) : '',
    group(
      'リファレンス',
      `${referenceLink('api/index.html', 'API リファレンス')}${referenceLink('models/index.html', 'モデルカタログ')}`,
      'reference/index.html',
    ),
    group('フォーマット', formats.map(leaf).join(''), index('format-index')),
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

/** API リファレンスとモデルカタログは一つに保ち、宣言元のモジュールから該当箇所へ案内する。 */
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
  return `<section class="context-reference"><h2 id="${id}">API とモデル</h2><p>このモジュールが宣言する TypeSpec から生成した情報である。</p>${operations}${models}</section>`
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
  const source = addDerivedStateDiagrams(stripFrontmatter(document.source))
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
  const body = `<article class="reference-page"><header class="reference-header"><p class="eyebrow">OpenAPI に基づくリファレンス</p><h1>API リファレンス</h1><p>生成した OpenAPI を Swagger UI で直接表示する。<code>?tag=</code> を指定すると、一つのモジュールの操作へ絞り込める。<a href="../openapi/${encodeURIComponent(openapiFileName)}">OpenAPI JSON を開く</a>。</p></header><div class="swagger-shell"><div id="swagger-ui" aria-label="API 操作"></div></div></article>`
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
  taskId: '対象プロビジョニングタスクの識別子。',
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

/** 機能仕様が見出しで宣言した要件。生成する一覧とトレーサビリティが読む。 */
type DeclaredRule = {
  id: string
  title: string
  document: RenderedDocument
  anchor: string
  openQuestions: string[]
  /** 要件が属する操作。操作の見出しより前に宣言した要件にはない。 */
  operation?: SpecificationOperation
}

/** 機能仕様の操作。README では「操作」の節の H3、章では H2 が一つの操作である。 */
type SpecificationOperation = {
  title: string
  line: number
  document: RenderedDocument
  anchor: string
}

const OPEN_QUESTION = /^- (?:\*\*)?要判断(?:\*\*)?[：:]\s*(.*)$/

/**
 * 機能仕様の操作の見出し。章のページは操作の節を分けたものなので、その H2 が操作になる。
 * README では「操作」の節の下の H3 だけを操作とし、モデルや状態遷移の H3 と区別する。
 */
function specificationOperations(document: RenderedDocument): SpecificationOperation[] {
  const chapter = !document.path.endsWith('/README.md')
  const operations: SpecificationOperation[] = []
  let section = ''
  let fenced = false
  for (const [index, text] of document.source.split('\n').entries()) {
    if (text.startsWith('```')) fenced = !fenced
    if (fenced) continue
    const h2 = text.match(/^## (.+)$/)?.[1]?.trim()
    if (h2 !== undefined) section = h2
    const title = chapter
      ? h2
      : section === '操作'
        ? text.match(/^### (.+)$/)?.[1]?.trim()
        : undefined
    if (title === undefined || title.startsWith('REQ-')) continue
    operations.push({ title, line: index + 1, document, anchor: `${document.id}-${slug(title)}` })
  }
  return operations
}

function declaredRules(documents: RenderedDocument[]): DeclaredRule[] {
  const rules: DeclaredRule[] = []
  for (const document of documents) {
    if (documentKind(document.path) !== 'specification') continue
    const bodies = new Map(ruleBodies(document.source).map((body) => [body.id, body.lines]))
    const operations = specificationOperations(document)
    for (const rule of specificationRules(document.source)) {
      rules.push({
        id: rule.id,
        title: rule.title,
        document,
        anchor: `${document.id}-${slug(`${rule.id} ${rule.title}`)}`,
        openQuestions: (bodies.get(rule.id) ?? []).flatMap(
          ({ text }) => text.match(OPEN_QUESTION)?.[1] ?? [],
        ),
        operation: operations.findLast((operation) => operation.line < rule.line),
      })
    }
  }
  return rules
}

/**
 * 機能仕様の操作の一覧。操作ごとに、その下で宣言した要件と未決事項の数を並べる。
 * 手で書いた一覧は、操作や要件を加えたときに更新が漏れても検出されないので、見出しから作る。
 * 一覧は機能仕様の本体（README）にだけ置き、章の操作も同じ一覧に並べる。
 */
function featureOperationIndex(
  document: RenderedDocument,
  documents: RenderedDocument[],
  rules: DeclaredRule[],
): string {
  if (documentKind(document.path) !== 'specification' || !document.path.endsWith('/README.md'))
    return ''
  const pages = documents.filter(
    (entry) =>
      entry.context === document.context &&
      entry.feature === document.feature &&
      documentKind(entry.path) === 'specification',
  )
  const operations = [
    ...pages.filter((entry) => entry === document),
    ...pages.filter((entry) => entry !== document),
  ].flatMap(specificationOperations)
  if (operations.length === 0) return ''
  const page = document.outputPath
  const rows = operations
    .map((operation) => {
      const own = rules.filter(
        (rule) =>
          rule.operation?.document === operation.document && rule.operation.line === operation.line,
      )
      const ruleLinks = own
        .map((rule) => siteLink(page, rule.document.outputPath, rule.id, rule.anchor))
        .join('、')
      const questions = own.reduce((count, rule) => count + rule.openQuestions.length, 0)
      return `<tr><th scope="row">${siteLink(page, operation.document.outputPath, operation.title, operation.anchor)}</th><td>${ruleLinks || '—'}</td><td>${questions}</td><td>${escapeHtml(operation.document === document ? 'このページ' : operation.document.title)}</td></tr>`
    })
    .join('')
  return `<p class="muted">操作の一覧：操作の見出しと、その下で宣言した要件から生成した。</p><div class="table-wrap"><table><thead><tr><th scope="col">操作</th><th scope="col">要件</th><th scope="col">未決事項</th><th scope="col">記載</th></tr></thead><tbody>${rows}</tbody></table></div>`
}

/** 生成した HTML を、ページの中の見出しの直後に差し込む。見出しがなければページを変えない。 */
function insertAfterHeading(page: string, id: string, html: string): string {
  if (!html) return page
  const open = page.indexOf(`id="${id}"`)
  if (open < 0) return page
  const close = page.indexOf('</h2>', open)
  if (close < 0) return page
  const at = close + '</h2>'.length
  return `${page.slice(0, at)}${html}${page.slice(at)}`
}

/**
 * 機能スライスの要件一覧と未決事項。手で書いた一覧は、要件を加えたときに更新が漏れても
 * どの検査にも見つからないので、要件の見出しと要判断の欄から作る。
 */
function featureRuleIndex(document: RenderedDocument, rules: DeclaredRule[]): string {
  const own = rules.filter(
    (rule) =>
      rule.document.context === document.context && rule.document.feature === document.feature,
  )
  if (own.length === 0) return ''
  const page = document.outputPath
  const rows = own
    .map(
      (rule) =>
        `<tr><th scope="row">${siteLink(page, rule.document.outputPath, rule.id, rule.anchor)}</th><td>${escapeHtml(rule.title)}</td><td>${escapeHtml(rule.document === document ? 'このページ' : rule.document.title)}</td></tr>`,
    )
    .join('')
  const questions = own.flatMap((rule) =>
    rule.openQuestions.map(
      (question) =>
        `<li>${siteLink(page, rule.document.outputPath, rule.id, rule.anchor)}：${inlineMarkdown.renderInline(question)}</li>`,
    ),
  )
  const index = `<h2 id="${document.id}-要件一覧">要件一覧</h2><p class="muted">この機能の仕様が宣言する要件の見出しから生成した。</p><div class="table-wrap"><table><thead><tr><th scope="col">要件</th><th scope="col">題名</th><th scope="col">記載</th></tr></thead><tbody>${rows}</tbody></table></div>`
  const open = questions.length
    ? `<h2 id="${document.id}-未決事項">未決事項</h2><p class="muted">要件の要判断の欄から生成した。要判断は work item として起票し、この一覧から消す。</p><ul>${questions.join('')}</ul>`
    : ''
  return `<section class="context-reference">${index}${open}</section>`
}

/** モジュールの機能地図。機能群、機能、要件と未決事項の数を、要件の見出しから作る。 */
function featureMap(
  document: RenderedDocument,
  documents: RenderedDocument[],
  rules: DeclaredRule[],
  sourcePaths: readonly string[],
  contextAliases: Record<string, string>,
): string {
  const context = document.context
  const nodes = inGroup(documents, 'feature').filter(
    (entry) => entry.context === context && entry.feature !== 'design',
  )
  const leaves = nodes.filter(
    (entry) => !nodes.some((other) => other.feature?.startsWith(`${entry.feature}/`)),
  )
  if (leaves.length === 0) return ''
  const page = document.outputPath
  const rows = leaves
    .map((leaf) => {
      const group = nodes.find(
        (entry) => entry !== leaf && leaf.feature?.startsWith(`${entry.feature}/`),
      )
      const own = rules.filter(
        (rule) => rule.document.context === context && rule.document.feature === leaf.feature,
      )
      const questions = own.reduce((count, rule) => count + rule.openQuestions.length, 0)
      const flatten = (value: string) => value.replaceAll('-', '')
      const feature = leaf.feature?.split('/').at(-1) ?? ''
      const paths = sourcePaths.filter((path) => {
        const [root, owner = '', name = ''] = path.split('/')
        if (root === 'spec')
          return path.startsWith(`spec/contexts/${context}/`) && path.endsWith('.tsp')
        return (
          root === 'backend' &&
          flatten(contextAliases[owner] ?? owner) === flatten(context ?? '') &&
          flatten(name) === flatten(feature)
        )
      })
      const pathList = (entries: string[]) =>
        entries.length
          ? `<ul>${entries.map((path) => `<li><code>${escapeHtml(path)}</code></li>`).join('')}</ul>`
          : '該当なし'
      const testPath = (path: string) =>
        path.endsWith('_test.go') ||
        path.includes('/testing_contract/') ||
        path.endsWith('.examples.json')
      return `<tr><td>${group ? siteLink(page, group.outputPath, group.title) : '—'}</td><th scope="row">${siteLink(page, leaf.outputPath, leaf.title)}</th><td>${own.length}</td><td>${questions}</td><td>${pathList(paths.filter((path) => !testPath(path)))}</td><td>${pathList(paths.filter(testPath))}</td></tr>`
    })
    .join('')
  return `<section class="context-reference"><h2 id="${document.id}-機能地図">機能地図</h2><p class="muted">機能スライス、要件、配置とモジュール名の対応から生成した探索用の候補であり、被覆の証明ではない。</p><div class="table-wrap"><table><thead><tr><th scope="col">機能群</th><th scope="col">機能</th><th scope="col">要件</th><th scope="col">未決事項</th><th scope="col">実装と契約の候補</th><th scope="col">テストと具体例の一次情報</th></tr></thead><tbody>${rows}</tbody></table></div></section>`
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
  const body = `<header class="reference-header"><p class="eyebrow">TypeSpec プログラム</p><h1>モデルカタログ</h1><p>リポジトリが所有するモデル、列挙、共用体、スカラーを、HTTP 操作への公開有無にかかわらず宣言元のモジュールごとに掲載する。<code>Operations</code> 名前空間の転送用ラッパーは OpenAPI に基づく API リファレンスへ掲載する。</p><label class="model-search">モデルを絞り込む <input type="search" data-model-search placeholder="名前、モジュール、説明" autocomplete="off"></label></header>${modelGroups(
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
        throw new Error(`${method.toUpperCase()} ${path} has no owning module tag`)
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
:root{color-scheme:light dark;--bg:#fff;--bg-soft:#f6f7f9;--text:#1c2024;--muted:#5c646f;--line:#e3e6ea;--line-strong:#c8ced7;--accent:#1f4fd8;--accent-soft:#eef2ff;--code:#f4f5f7;--given:#0f6b84;--when:#8a5200;--then:#0f6b45;--diagram-line:#294cba;--measure:960px;--wide:1400px;--sidebar:290px;--toc:216px}
@media(prefers-color-scheme:dark){:root{--bg:#14171c;--bg-soft:#1a1e25;--text:#e7ebf1;--muted:#9aa3b0;--line:#282e38;--line-strong:#3b4350;--accent:#93adff;--accent-soft:#1e2740;--code:#1e232b;--given:#7bd6f0;--when:#ffc36b;--then:#74d6a0;--diagram-line:#b9c8ff}}
*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;color:var(--text);background:var(--bg);font:16px/1.75 Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;overflow-wrap:anywhere}.skip-link{position:fixed;z-index:20;top:8px;left:8px;transform:translateY(-160%);padding:8px 12px;background:var(--bg);border:2px solid var(--accent);border-radius:8px}.skip-link:focus{transform:none}
.sidebar{position:fixed;inset:0 auto 0 0;width:var(--sidebar);overflow:auto;padding:26px 14px 64px;border-right:1px solid var(--line);background:var(--bg-soft)}.site-title{margin:0 10px 22px;font-size:16px;font-weight:700;letter-spacing:-.01em}.site-title a{color:inherit;text-decoration:none}
.nav-section{margin:0 0 4px}.nav-section>summary{padding:7px 10px;color:var(--muted);font-size:12px;font-weight:700;letter-spacing:.08em;cursor:pointer;list-style:none}.nav-section>summary::-webkit-details-marker{display:none}.nav-section>summary::before{content:"▸";display:inline-block;width:14px;color:var(--line-strong)}.nav-section[open]>summary::before{content:"▾"}.nav-section-link{color:inherit;text-decoration:none}.nav-section-link:hover,.nav-section-link[aria-current=page]{color:var(--accent)}
.nav-tree{margin:0 0 10px;padding:0;list-style:none}.nav-tree ul{margin:0;padding:0 0 0 18px;list-style:none}.nav-item,.nav-branch{margin:0}.nav-link,.nav-label{display:block;padding:4px 10px;border-left:2px solid transparent;color:var(--text);font-size:13.5px;line-height:1.5;text-decoration:none}.nav-label{cursor:pointer}.nav-link:hover{background:var(--accent-soft)}.nav-link[aria-current=page]{border-left-color:var(--accent);color:var(--accent);background:var(--accent-soft);font-weight:700}
.nav-directory>summary{display:flex;align-items:flex-start;cursor:pointer;list-style:none}.nav-directory>summary::-webkit-details-marker{display:none}.nav-directory>summary::before{content:"▸";flex:none;width:16px;padding:4px 0;color:var(--line-strong);font-size:10px;line-height:1.9;text-align:center}.nav-directory[open]>summary::before{content:"▾"}.nav-directory>summary>.nav-link,.nav-directory>summary>.nav-label{flex:1;min-width:0}.nav-item>.nav-link{padding-left:26px}
main{margin-left:var(--sidebar);padding:34px 40px 140px}main:has(.swagger-shell){max-width:none;padding-inline:26px}.breadcrumbs{display:flex;gap:8px;align-items:center;margin:0 0 26px;color:var(--muted);font-size:13px}.mobile-header{display:none}
.page{display:grid;grid-template-columns:minmax(0,var(--wide)) minmax(0,var(--toc));gap:56px;align-items:start}.document>:is(p,ul,ol,blockquote,dl,h1,h2,h3,h4){max-width:var(--measure)}.landing{max-width:880px}
.page-toc{position:sticky;top:34px;max-height:calc(100vh - 80px);overflow:auto;padding-left:16px;border-left:1px solid var(--line);font-size:13px;line-height:1.5}.page-toc-title{margin:0 0 8px;color:var(--muted);font-weight:700}.page-toc ul{margin:0;padding:0;list-style:none}.page-toc a{display:block;padding:4px 0;color:var(--muted);text-decoration:none}.page-toc a:hover{color:var(--text)}.page-toc a[aria-current=location]{color:var(--accent)}.page-toc-h3 a{padding-left:14px;font-size:12.5px}
h1,h2,h3,h4{line-height:1.35;letter-spacing:-.012em;scroll-margin-top:26px}h1{margin:0 0 .7em;font-size:30px}h2{margin:2.4em 0 .7em;font-size:21px}h3{margin:1.9em 0 .5em;font-size:17px}h4{margin:1.6em 0 .4em;font-size:15px}p,ul,ol{margin:0 0 1.1em}
a{color:var(--accent);text-underline-offset:2px}a:focus-visible,summary:focus-visible,input:focus-visible{outline:3px solid var(--accent);outline-offset:3px;border-radius:4px}
code{padding:.1em .34em;border:1px solid var(--line);border-radius:4px;background:var(--code);font-size:.87em}pre{max-width:100%;overflow:auto;margin:1.4em 0;padding:16px 18px;border:1px solid var(--line);border-radius:8px;background:var(--code)}pre code{padding:0;border:0;background:none;font-size:.86em}
.hero{margin:0 0 44px;padding:0 0 30px;border-bottom:1px solid var(--line)}.hero h1{font-size:34px}.hero p{color:var(--muted);font-size:17px}.hero-links{display:flex;flex-wrap:wrap;gap:20px;margin:0;font-size:15px;font-weight:600}
.context-links{display:flex;flex-wrap:wrap;gap:8px 18px;margin:1em 0;padding:0;list-style:none;font-size:14px}
.table-wrap,table{max-width:100%;overflow:auto}table{width:100%;margin:1.4em 0;border-collapse:collapse;display:block;font-size:14.5px;line-height:1.65}th,td{padding:9px 14px 9px 0;border:0;border-bottom:1px solid var(--line);text-align:left;vertical-align:top;overflow-wrap:break-word}thead th{padding-left:10px;border-bottom:1px solid var(--line-strong);background:var(--bg-soft);font-size:13px}tbody th[scope=row]{font-weight:600}.term-table td:first-child{white-space:nowrap}
.diagram-shell{position:relative;max-width:100%;overflow:auto;margin:1.6em 0;padding:18px;border:1px solid var(--line);border-radius:10px;background:var(--bg-soft)}.diagram-shell .mermaid{min-width:560px;background:transparent}:is(.diagram-shell .mermaid,.diagram-canvas) svg :is(.edgePath path,.flowchart-link,.transition){stroke:var(--diagram-line)!important;stroke-width:2.4px!important}:is(.diagram-shell .mermaid,.diagram-canvas) svg marker path{fill:var(--diagram-line)!important;stroke:var(--diagram-line)!important}
.diagram-zoom,.diagram-viewer-toolbar button{padding:4px 12px;color:var(--text);background:var(--bg);border:1px solid var(--line-strong);border-radius:6px;font:inherit;font-size:13px;cursor:pointer}.diagram-zoom:hover,.diagram-viewer-toolbar button:hover{background:var(--accent-soft)}.diagram-zoom:focus-visible,.diagram-viewer-toolbar button:focus-visible{outline:3px solid var(--accent);outline-offset:2px}.diagram-zoom{display:block;width:fit-content;margin:-6px -6px 6px auto}
.diagram-viewer{width:100vw;max-width:none;height:100vh;max-height:none;margin:0;padding:0;border:0;color:var(--text);background:var(--bg)}.diagram-viewer::backdrop{background:rgba(20,23,28,.6)}.diagram-viewer-toolbar{position:absolute;z-index:1;top:12px;right:12px;display:flex;gap:8px}.diagram-viewport{width:100%;height:100%;overflow:hidden;cursor:grab;touch-action:none}.diagram-viewport.dragging{cursor:grabbing}.diagram-canvas{transform-origin:0 0}.diagram-canvas svg{display:block;max-width:none}
.scenario-keyword{display:inline-block;min-width:58px;margin-right:5px;padding:1px 7px;border:1px solid currentColor;border-radius:999px;font-size:11px;font-weight:800;letter-spacing:.04em;text-align:center}.scenario-keyword.given,.scenario-keyword.and{color:var(--given)}.scenario-keyword.when,.scenario-keyword.but{color:var(--when)}.scenario-keyword.then{color:var(--then)}li:has(>.scenario-keyword){margin:.45em 0}.scenario-actor{display:inline-block;margin-right:6px;padding:1px 9px;border:1px dashed currentColor;border-radius:999px;color:var(--muted);font-size:11px;font-weight:700;letter-spacing:.04em}p:has(>.scenario-actor){margin:.35em 0 .9em}
.reference-header{margin-bottom:26px}.reference-page{max-width:none}.swagger-shell{color-scheme:light;margin:24px 0 0;padding:20px;overflow:auto;border:1px solid var(--line);border-radius:10px;background:#fff;color:#3b4151}.swagger-shell .swagger-ui .wrapper{max-width:none;padding-inline:0}
.model-group{margin-top:34px}.model-list{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:12px}.model-list article{padding:16px;border:1px solid var(--line);border-radius:10px;background:var(--bg-soft)}.model-list h3{margin:.4em 0}.model-list p{color:var(--muted)}.model-search{display:grid;max-width:520px;gap:6px;margin-top:22px;font-weight:700}.model-search input{width:100%;padding:10px 12px;color:var(--text);background:var(--bg);border:1px solid var(--line-strong);border-radius:8px;font:inherit}
.kind,.api-exposed,.not-exposed,.required,.optional{display:inline-block;margin:0 6px 4px 0;padding:2px 7px;border-radius:999px;font-size:11px;font-weight:800}.kind,.optional{color:var(--muted);background:var(--code)}.api-exposed,.required{color:#fff;background:#28664b}.not-exposed{color:var(--muted);border:1px solid var(--line)}.qualified{padding:12px;border-radius:8px;background:var(--bg-soft)}.badges{margin:.5em 0}.context-reference{margin-top:44px;padding-top:10px;border-top:1px solid var(--line)}.symbol-links{display:flex;flex-wrap:wrap;gap:6px 14px;margin:.6em 0;padding:0;list-style:none}.meta{margin-top:7px;color:var(--muted);font-size:13px}.compact{margin:.5em 0;padding-left:20px}.muted{color:var(--muted)}[hidden]{display:none!important}
@media(max-width:1200px){.page{grid-template-columns:minmax(0,1fr)}.page-toc{display:none}}
@media(max-width:900px){.sidebar{display:none}.mobile-header{display:flex;position:sticky;z-index:10;top:0;justify-content:space-between;align-items:flex-start;padding:12px 18px;border-bottom:1px solid var(--line);background:var(--bg)}.mobile-header>details{position:relative}.mobile-header details>nav{position:absolute;right:0;width:min(86vw,320px);max-height:75vh;overflow:auto;padding:12px;border:1px solid var(--line);border-radius:10px;background:var(--bg);box-shadow:0 12px 32px rgba(20,23,28,.18)}main{margin:0;padding:20px 18px 80px}main:has(.swagger-shell){padding-inline:18px}.hero h1{font-size:28px}.diagram-shell .mermaid{min-width:480px}}
@media print{.sidebar,.mobile-header,.breadcrumbs,.skip-link,.page-toc{display:none}main{margin:0;padding:0}.page{display:block}a{color:inherit;text-decoration:none}}
`

const siteScript = `
// 図は複製せず、表示している間だけダイアログへ移す。複製すると Mermaid が SVG に振った id が
// 重複し、マーカーを指す url(#...) が元の図の要素へ解決されてしまう。
function createDiagramViewer(){
  var dialog=document.createElement('dialog');
  dialog.className='diagram-viewer';
  dialog.setAttribute('aria-label','図の拡大表示');
  dialog.innerHTML='<div class="diagram-viewer-toolbar"><button type="button" data-zoom-in>拡大</button><button type="button" data-zoom-out>縮小</button><button type="button" data-zoom-fit>全体表示</button><button type="button" data-zoom-close>閉じる</button></div><div class="diagram-viewport"><div class="diagram-canvas"></div></div>';
  document.body.appendChild(dialog);
  var viewport=dialog.querySelector('.diagram-viewport');
  var canvas=dialog.querySelector('.diagram-canvas');
  var view={scale:1,x:0,y:0,width:1,height:1};
  var shown=null;
  var pointers={};
  var apply=function(){canvas.style.transform='translate('+view.x+'px,'+view.y+'px) scale('+view.scale+')';};
  var viewportSize=function(){return{width:viewport.clientWidth||window.innerWidth,height:viewport.clientHeight||window.innerHeight};};
  var fit=function(){
    var size=viewportSize();
    view.scale=Math.min(size.width/view.width,size.height/view.height)*0.95;
    view.x=(size.width-view.width*view.scale)/2;
    view.y=(size.height-view.height*view.scale)/2;
    apply();
  };
  var zoomAt=function(factor,x,y){
    var next=Math.min(Math.max(view.scale*factor,0.05),20);
    view.x=x-(x-view.x)*(next/view.scale);
    view.y=y-(y-view.y)*(next/view.scale);
    view.scale=next;
    apply();
  };
  var zoomAtCenter=function(factor){var size=viewportSize();zoomAt(factor,size.width/2,size.height/2);};
  var pan=function(dx,dy){view.x+=dx;view.y+=dy;apply();};
  dialog.querySelector('[data-zoom-in]').addEventListener('click',function(){zoomAtCenter(1.25);});
  dialog.querySelector('[data-zoom-out]').addEventListener('click',function(){zoomAtCenter(0.8);});
  dialog.querySelector('[data-zoom-fit]').addEventListener('click',fit);
  dialog.querySelector('[data-zoom-close]').addEventListener('click',function(){dialog.close();});
  dialog.addEventListener('keydown',function(event){
    var step={ArrowLeft:[40,0],ArrowRight:[-40,0],ArrowUp:[0,40],ArrowDown:[0,-40]}[event.key];
    if(event.key==='+'||event.key==='=')zoomAtCenter(1.25);
    else if(event.key==='-')zoomAtCenter(0.8);
    else if(event.key==='0')fit();
    else if(step)pan(step[0],step[1]);
    else return;
    event.preventDefault();
  });
  viewport.addEventListener('wheel',function(event){
    event.preventDefault();
    var box=viewport.getBoundingClientRect();
    zoomAt(Math.exp(-event.deltaY*0.0015),event.clientX-box.left,event.clientY-box.top);
  },{passive:false});
  // ポインターが一つならドラッグで移動し、二つならその距離の比で拡大率を変える。
  var pinch=function(){
    var points=Object.keys(pointers).map(function(id){return pointers[id];});
    return points.length===2?{distance:Math.hypot(points[0].x-points[1].x,points[0].y-points[1].y),x:(points[0].x+points[1].x)/2,y:(points[0].y+points[1].y)/2}:null;
  };
  viewport.addEventListener('pointerdown',function(event){
    viewport.setPointerCapture(event.pointerId);
    pointers[event.pointerId]={x:event.clientX,y:event.clientY};
    viewport.classList.add('dragging');
  });
  viewport.addEventListener('pointermove',function(event){
    var previous=pointers[event.pointerId];
    if(!previous)return;
    var before=pinch();
    pointers[event.pointerId]={x:event.clientX,y:event.clientY};
    var after=pinch();
    if(before&&after){
      var box=viewport.getBoundingClientRect();
      zoomAt(after.distance/before.distance,after.x-box.left,after.y-box.top);
      pan(after.x-before.x,after.y-before.y);
    }else if(!after){
      pan(event.clientX-previous.x,event.clientY-previous.y);
    }
  });
  var release=function(event){
    delete pointers[event.pointerId];
    if(Object.keys(pointers).length===0)viewport.classList.remove('dragging');
  };
  viewport.addEventListener('pointerup',release);
  viewport.addEventListener('pointercancel',release);
  dialog.addEventListener('close',function(){
    if(!shown)return;
    shown.parent.insertBefore(shown.svg,shown.next);
    ['width','height','style'].forEach(function(name){
      if(shown.attributes[name]===null)shown.svg.removeAttribute(name);
      else shown.svg.setAttribute(name,shown.attributes[name]);
    });
    shown.button.focus();
    shown=null;
    pointers={};
  });
  return{open:function(svg,button){
    var box=(svg.getAttribute('viewBox')||'').split(/[\\s,]+/).map(Number);
    var rendered=svg.getBoundingClientRect();
    view.width=box[2]||rendered.width||1;
    view.height=box[3]||rendered.height||1;
    shown={svg:svg,button:button,parent:svg.parentNode,next:svg.nextSibling,attributes:{}};
    ['width','height','style'].forEach(function(name){shown.attributes[name]=svg.getAttribute(name);});
    // Mermaid は本文の幅へ縮める max-width を属性で与えるので、表示中だけ元の大きさに戻す。
    svg.removeAttribute('style');
    svg.setAttribute('width',String(view.width));
    svg.setAttribute('height',String(view.height));
    canvas.appendChild(svg);
    dialog.showModal();
    fit();
  }};
}
window.addEventListener('DOMContentLoaded',function(){
  if(window.mermaid&&document.querySelector('.mermaid')){
    var dark=window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches;
    window.mermaid.initialize({startOnLoad:false,securityLevel:'strict',layout:'dagre',look:'classic',theme:'base',themeVariables:dark?{background:'#1a1e25',primaryColor:'#1e2740',primaryTextColor:'#e7ebf1',primaryBorderColor:'#b9c8ff',lineColor:'#b9c8ff',textColor:'#e7ebf1',edgeLabelBackground:'#14171c',tertiaryColor:'#1a1e25'}:{background:'#f6f7f9',primaryColor:'#eef2ff',primaryTextColor:'#1c2024',primaryBorderColor:'#294cba',lineColor:'#294cba',textColor:'#1c2024',edgeLabelBackground:'#fff',tertiaryColor:'#f6f7f9'}});
    window.mermaid.run({querySelector:'.mermaid'});
  }
  var viewer=null;
  document.querySelectorAll('[data-diagram-zoom]').forEach(function(button){
    button.addEventListener('click',function(){
      var svg=button.parentNode.querySelector('svg');
      if(!svg)return;
      if(!viewer)viewer=createDiagramViewer();
      viewer.open(svg,button);
    });
  });
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
  contextTags?: Record<string, string[]>
  sourcePaths?: string[]
  contextAliases?: Record<string, string>
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
  const files: Record<string, string> = {
    'api/index.html': apiPage(args.openapi, args.openapiFileName, documents),
    'models/index.html': modelIndex(args.models, documents),
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
      ],
      documents,
    }),
  }
  const rules = declaredRules(documents)
  for (const document of documents) {
    const tags = (document.context ? args.contextTags?.[document.context] : undefined) ?? []
    const reference =
      document.category === 'context'
        ? featureMap(
            document,
            documents,
            rules,
            args.sourcePaths ?? [],
            args.contextAliases ?? {},
          ) +
          contextReference({
            document,
            tags,
            operations: openapi.operations.filter((operation) =>
              operation.tags.some((tag) => tags.includes(tag)),
            ),
            models: args.models.filter((model) => model.context === document.context),
          })
        : document.category === 'feature'
          ? featureRuleIndex(document, rules)
          : ''
    files[document.outputPath] = insertAfterHeading(
      documentPage(document, documents, markdown, reference),
      `${document.id}-${slug('操作')}`,
      document.category === 'feature' ? featureOperationIndex(document, documents, rules) : '',
    )
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

import { posix } from 'node:path'
import { moduleFlow } from './ui-flow.ts'
import type { UiCatalog } from './ui-pages.ts'

type UiPage = UiCatalog['pages'][number]
type UiField = UiPage['parameters'][number]
const root = 'docs/design/application/pages'
const areas: Record<string, string> = {
  authentication: '認証',
  account: 'アカウントポータル',
  admin: '管理コンソール',
  system: 'システムコンソール',
}

function cell(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('|', '&#124;')
    .replaceAll('\n', ' ')
}

function pagePath(page: UiPage): string {
  return `docs/modules/${page.module}/pages/${page.id}.md`
}

function link(from: string, target: string, label: string): string {
  return `[${cell(label)}](${posix.relative(posix.dirname(from), target)})`
}

function introduction(title: string, purpose: string, preview: boolean): string[] {
  return [
    `# ${title}`,
    '',
    `この文書は、${purpose}を示す。`,
    ...(preview
      ? ['', '> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。']
      : []),
    '',
    'TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。',
    '',
  ]
}

function fieldsTable(fields: readonly UiField[]): string[] {
  return [
    '| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |',
    '| --- | --- | --- | --- | --- | --- | --- |',
    ...fields.map(
      (field) =>
        `| ${cell(field.label)}${field.repeated ? '（繰り返し）' : ''} | ${field.id} | ${cell(field.control)} | ${cell(field.type)} | ${cell(field.initial)} | ${cell(field.condition)} | ${cell([field.binding, field.mapping].filter(Boolean).join('：'))} |`,
    ),
    '',
  ]
}

function summary(pages: readonly UiPage[], path: string): string[] {
  return [
    '| ページ | URL | 目的 | 入力項目数 |',
    '| --- | --- | --- | --- |',
    ...pages.map(
      (page) =>
        `| ${link(path, pagePath(page), page.name)} | \`${page.path}\` | ${cell(page.purpose)} | ${page.forms.reduce((sum, form) => sum + form.fields.length, 0)} |`,
    ),
    '',
  ]
}

function detail(page: UiPage, catalog: UiCatalog, preview: boolean): string {
  const path = pagePath(page)
  const categoryPath = `${root}/${page.area}/${page.category}/README.md`
  const modulePath = `docs/modules/${page.module}/pages/README.md`
  const moduleName = catalog.moduleNames[page.module] ?? page.module
  const lines = introduction(page.name, `${page.name}の操作、入力項目と状態`, preview)
  const destinationLink = (id: string) => {
    const target = catalog.pages.find((one) => one.id === id)
    return target
      ? link(path, pagePath(target), target.name)
      : (catalog.destinations.find((one) => one.id === id)?.name ?? id)
  }
  lines.push(
    `${link(path, `docs/modules/${page.module}/README.md`, moduleName)} / ${link(path, modulePath, 'ページ一覧')} / ${page.name}`,
    '',
    `分類：${link(path, `${root}/${page.area}/README.md`, areas[page.area]!)} / ${link(path, categoryPath, page.categoryName)}`,
    '',
    `\`${page.id}\`　\`${page.path}\``,
    '',
    page.purpose,
    '',
    '| 項目 | 内容 |',
    '| --- | --- |',
    `| 主担当 | ${link(path, `docs/modules/${page.module}/README.md`, moduleName)} |`,
    `| 利用者 | ${cell(page.actor)} |`,
    `| 入口 | ${cell(page.entry)} |`,
    `| 利用条件 | ${cell(page.context)} |`,
    `| 表示内容 | ${cell(page.content.join('、'))} |`,
    '',
    `利用手順は${link(path, modulePath, `${moduleName} のページ一覧`)}にまとめる。`,
    '',
    '## フォームと入力項目',
    '',
    '項目ごとのバリデーション規則は別の作業で定義する。',
    '',
  )
  if (!page.forms.length) lines.push('入力フォームはない。', '')
  for (const form of page.forms) {
    lines.push(
      `### ${form.name}`,
      '',
      `識別子：${page.id}/${form.id}。表示条件：${form.condition}。確定：${form.commit}。`,
      '',
    )
    lines.push(...(form.fields.length ? fieldsTable(form.fields) : ['追加の入力項目はない。', '']))
  }
  lines.push('## パラメーター', '')
  lines.push(
    ...(page.parameters.length
      ? fieldsTable(page.parameters)
      : ['ページ固有のパラメーターはない。', '']),
  )
  lines.push(
    '## 操作と結果の詳細',
    '',
    '| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |',
    '| --- | --- | --- | --- | --- | --- | --- | --- |',
  )
  for (const action of page.actions)
    lines.push(
      `| ${action.id} | ${action.from} | ${cell(action.event)} | ${cell(action.condition)} | ${cell(action.operations.join('、') || 'UI 内の操作')} | ${cell(action.result)} | ${destinationLink(action.destination)} / ${action.toState} | ${cell(action.data)} |`,
    )
  lines.push('', '## 表示状態', '', '| 状態 | 表示 |', '| --- | --- |')
  for (const [state, description] of Object.entries(page.states))
    lines.push(`| ${state} | ${cell(description)} |`)
  const applicationDirectory = posix.relative(posix.dirname(path), 'docs/design/application')
  const references = page.references.map((reference) =>
    reference.replaceAll('](user-interface.md', `](${applicationDirectory}/user-interface.md`),
  )
  lines.push(
    '',
    '## 関連する設計',
    '',
    `- ${link(path, page.source, 'TypeSpec の定義')}`,
    `- ${link(path, 'docs/design/application/user-interface.md', '共通の UI 設計')}`,
    ...references.map((reference) => `- ${reference}`),
    '',
    `${link(path, modulePath, `${moduleName}のページ一覧へ戻る`)}`,
    '',
  )
  return lines.join('\n')
}

/** UI の作業カテゴリで一覧を分け、詳細は一つのページを一つの文書にする。 */
export function renderUiDocuments(
  catalog: UiCatalog,
  options: { preview?: boolean } = {},
): Map<string, string> {
  const preview = options.preview ?? false
  const documents = new Map<string, string>()
  for (const page of catalog.pages) {
    if (
      ![page.id, page.module, page.area, page.category].every((value) =>
        /^[a-z][a-z0-9-]*$/.test(value),
      )
    )
      throw new Error(`UI の文書パスに使えない識別子: ${page.id}`)
  }
  const indexPath = `${root}/README.md`
  const index = introduction('UI ページ', '利用者の作業で分類したページ設計の入口', preview)
  index.push('## 主担当のモジュール', '', '| モジュール | ページ数 |', '| --- | --- |')
  for (const module of new Set(catalog.pages.map((page) => page.module))) {
    const pages = catalog.pages.filter((page) => page.module === module)
    const moduleName = catalog.moduleNames[module] ?? module
    index.push(
      `| ${link(indexPath, `docs/modules/${module}/pages/README.md`, moduleName)} | ${pages.length} |`,
    )
  }
  index.push('')
  index.push('## 領域', '', '| 領域 | カテゴリ | ページ数 |', '| --- | --- | --- |')
  for (const [area, name] of Object.entries(areas)) {
    const pages = catalog.pages.filter((page) => page.area === area)
    if (!pages.length) continue
    const areaPath = `${root}/${area}/README.md`
    const categories = new Map(pages.map((page) => [page.category, page.categoryName]))
    index.push(
      `| ${link(indexPath, areaPath, name)} | ${[...categories.values()].map(cell).join('、')} | ${pages.length} |`,
    )
    const areaLines = introduction(name, `${name}の作業カテゴリ`, preview)
    areaLines.push(
      `${link(areaPath, indexPath, 'UI ページへ戻る')}`,
      '',
      '## 作業カテゴリ',
      '',
      '| カテゴリ | ページ数 |',
      '| --- | --- |',
    )
    for (const [category, categoryName] of categories) {
      const categoryPath = `${root}/${area}/${category}/README.md`
      const categoryPages = pages.filter((page) => page.category === category)
      areaLines.push(`| ${link(areaPath, categoryPath, categoryName)} | ${categoryPages.length} |`)
      const categoryLines = introduction(categoryName, `${categoryName}のページ一覧`, preview)
      categoryLines.push(
        `${link(categoryPath, indexPath, 'UI ページ')} / ${link(categoryPath, areaPath, name)}`,
        '',
        '## ページ一覧',
        '',
        ...summary(categoryPages, categoryPath),
      )
      documents.set(categoryPath, categoryLines.join('\n'))
      for (const page of categoryPages)
        documents.set(pagePath(page), detail(page, catalog, preview))
    }
    documents.set(areaPath, `${areaLines.join('\n')}\n`)
  }
  index.push(
    '',
    '## 共通の設計',
    '',
    `- ${link(indexPath, 'docs/design/application/user-interface.md', 'UI の共通規則')}`,
    `- ${link(indexPath, 'docs/design/application/frontend.md', 'フロントエンド設計')}`,
    `- ${link(indexPath, 'docs/modules/frontend/pages/shared.md', '共通表示と外部の到達先')}`,
    '',
  )
  documents.set(indexPath, index.join('\n'))
  const shared = introduction(
    '共通表示と外部の到達先',
    'ページを数える際に区別する共有要素と外部システム',
    preview,
  )
  shared.push(
    '[Frontend のページへ戻る](README.md)',
    '',
    '## 共通表示と外部の到達先',
    '',
    '| 名前 | 識別子 | 区分 | 役割 |',
    '| --- | --- | --- | --- |',
  )
  for (const one of catalog.destinations)
    shared.push(
      `| ${cell(one.name)} | ${one.id} | ${one.kind === 'shared' ? '共通表示' : '外部'} | ${cell(one.description)} |`,
    )
  shared.push(
    '',
    '## ページを描画しない経路',
    '',
    '| 経路 ID | 区分 | 理由 |',
    '| --- | --- | --- |',
  )
  for (const note of catalog.routeNotes)
    shared.push(`| \`${note.route}\` | ${note.kind} | ${cell(note.reason)} |`)
  documents.set('docs/modules/frontend/pages/shared.md', `${shared.join('\n')}\n`)
  for (const module of new Set([...catalog.pages.map((page) => page.module), 'frontend'])) {
    const path = `docs/modules/${module}/pages/README.md`
    const name = catalog.moduleNames[module] ?? module
    const pages = catalog.pages.filter((page) => page.module === module)
    const lines = introduction(`${name} のページ`, `${name} が主担当となるページの設計`, preview)
    lines.push(
      `${link(path, `docs/modules/${module}/README.md`, `${name} のモジュール設計`)} / ${link(path, indexPath, 'UI 全体の索引')}`,
      '',
    )
    const flows = catalog.flows.filter((flow) => flow.module === module)
    if (flows.length)
      lines.push(
        '## 主な利用手順',
        '',
        '枠は利用者に見えるページや表示、矢印は次へ進む操作を示す。破線は戻り道、枠の破線はシステム外の表示である。',
        '',
      )
    for (const flow of flows) {
      lines.push(
        `### ${flow.name}`,
        '',
        flow.goal,
        '',
        '```mermaid',
        moduleFlow(flow, catalog),
        '```',
        '',
      )
      for (const note of flow.notes) lines.push(`- ${note}`)
      lines.push('')
      const pagesInFlow = new Set(flow.views.map((view) => view.page))
      lines.push(
        `ページの詳細：${pages
          .filter((page) => pagesInFlow.has(page.id))
          .map((page) => link(path, pagePath(page), page.name))
          .join('、')}`,
        '',
      )
    }
    for (const [category, categoryName] of new Map(
      pages.map((page) => [page.category, page.categoryName]),
    )) {
      lines.push(
        `## ${categoryName}`,
        '',
        ...summary(
          pages.filter((page) => page.category === category),
          path,
        ),
      )
    }
    if (module === 'frontend')
      lines.push('## 共通表示', '', '[共通表示と外部の到達先](shared.md)', '')
    documents.set(path, lines.join('\n'))
  }
  return documents
}

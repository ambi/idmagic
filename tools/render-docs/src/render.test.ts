import { describe, expect, it } from 'bun:test'
import { renderDocumentationSite } from './render.ts'
import type { CatalogSymbol } from './typespec-catalog.ts'

const rootDocument = {
  path: 'docs/README.md',
  source: `# Whole-System Specification

The whole system.

## Context Map

\`\`\`mermaid
flowchart LR
  Demo --> Other
\`\`\`

## Documents

| File | Content |
|---|---|
| [domain/demo/README.md](domain/demo/README.md) | The demo context |
`,
}

const rootProductOverviewDocument = {
  path: 'docs/requirements/product-overview.md',
  source: '# プロダクト概要\n\nプロダクトの目的。\n',
}

const rootGlossaryDocument = {
  path: 'docs/domain/glossary.md',
  source: `# 用語集

| Term | Definition |
|---|---|
| InterfaceStability | 外部契約としての安定性の区分。 |
`,
}

const rootStandardsDocument = {
  path: 'docs/domain/standards.md',
  source: '# 全体の標準仕様\n\n採用する標準仕様。\n',
}

const rootStructureDocument = {
  path: 'docs/domain/structure.md',
  source: '# 構造\n\nディレクトリの配置。\n',
}

const rootScenariosDocument = {
  path: 'docs/domain/scenarios.feature.md',
  source: '# Feature: Cross-Context Scenarios\n',
}

const requirementsIndexDocument = {
  path: 'docs/requirements/README.md',
  source: '# 要求\n\n要求の分類。\n',
}

const qualityDocument = {
  path: 'docs/requirements/quality.md',
  source: '# 品質要求\n\nシステム品質の目標。\n',
}

const designDocument = {
  path: 'docs/design/README.md',
  source: '# 設計\n\n設計の索引。\n',
}

const applicationDesignDocument = {
  path: 'docs/design/application/README.md',
  source: '# アプリケーション\n\nアプリケーション設計の索引。\n',
}

const apiGuidelinesDocument = {
  path: 'docs/design/application/api-guidelines.md',
  source: '# API ガイドライン\n\nAPI の設計の観点。\n',
}

const designGuidelinesDocument = {
  path: 'docs/design/application/design-guidelines.md',
  source: '# 設計ガイドライン\n\n設計の観点。\n',
}

const contextDocument = {
  path: 'docs/domain/demo/README.md',
  source: `# Demo

The demo context.

| File | Content |
|---|---|
| [states.md](states.md) | 状態と遷移 |
| [scenarios.feature.md](scenarios.feature.md) | 受け入れシナリオ |
`,
}

const statesDocument = {
  path: 'docs/domain/demo/states.md',
  source: `# Demo State Transitions

## DemoLifecycle

| State | Kind | Meaning |
|---|---|---|
| Ready | initial | 受理直後 |
| Running | — | 実行中 |
| Done | terminal | 完了 |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Ready | Start | allowed \\| privileged | Running | emit Started |
| Running | Finish | complete | Done | emit Completed |
| Done | Reset | "" | Ready | emit Reset |
`,
}

const scenariosDocument = {
  path: 'docs/domain/demo/scenarios.feature.md',
  source: `# Feature: Demo

## Rule: REQ-DEMO-001 a demo runs

Primary actor: \`Developer\`

### Example: EX-DEMO-001-01 a ready demo starts

- Given a ready demo
- When the developer starts the demo
- Then \`demo:run\` keeps the demo running

### Example: EX-DEMO-001-02 a forbidden start changes nothing

- Given a ready demo
- When the developer starts the demo
- But start is forbidden
- Then the demo stays ready
`,
}

const glossaryDocument = {
  path: 'docs/domain/demo/glossary.md',
  source: `# Demo Glossary

| Term | Definition |
|---|---|
| DemoRun | 1 回の実行。 |
`,
}

const domainIndexDocument = {
  path: 'docs/domain/README.md',
  source: '# ドメイン設計文書\n\nドメイン設計文書の入口。\n',
}

const operationsIndexDocument = {
  path: 'docs/operations/README.md',
  source: '# 運用\n\n運用文書の入口。\n',
}

const serviceManagementDocument = {
  path: 'docs/operations/service-management.md',
  source: '# サービス管理\n\n平常時の管理。\n',
}

const runbookDocument = {
  path: 'docs/runbooks/async-jobs.md',
  source: '# 非同期ジョブのランブック\n\n停滞したときの手順。\n',
}

const documentationGuideDocument = {
  path: 'DOCUMENTATION_GUIDE.md',
  source: '# 文書ガイド\n\n文書体系と配置を定める。\n',
}

const guideDocument = {
  path: 'WORK_ITEM_FORMAT.md',
  source: `# Work Item Format

When the work is complete, set the status.

- When an item enters \`in_progress\`, add the evidence policy.
- Then move the file to the done directory.
`,
}

const developmentDocument = {
  path: 'docs/development/README.md',
  source: `# 開発文書

| 文書 | 内容 |
| --- | --- |
| [release.md](release.md) | リリース手順 |
`,
}

const releaseDocument = {
  path: 'docs/development/release.md',
  source: '# リリース\n\n段階的に展開する。\n',
}

const models: CatalogSymbol[] = [
  {
    kind: 'model',
    name: 'Example.Demo.InternalRecord',
    namespace: 'Example.Demo',
    shortName: 'InternalRecord',
    doc: 'A specification model that is not exposed by HTTP.',
    apiExposed: false,
    context: 'demo',
    properties: [
      {
        name: 'id',
        type: 'string',
        optional: false,
        doc: 'Stable identifier.',
        constraints: ['minLength: 3'],
        references: [],
      },
    ],
    members: [],
    references: [],
  },
]

/** The canonical order is glossary before states before scenarios, not the alphabet. */
const site = () =>
  renderDocumentationSite({
    documents: [
      rootDocument,
      rootProductOverviewDocument,
      rootGlossaryDocument,
      rootStandardsDocument,
      rootStructureDocument,
      rootScenariosDocument,
      requirementsIndexDocument,
      qualityDocument,
      designDocument,
      domainIndexDocument,
      operationsIndexDocument,
      serviceManagementDocument,
      runbookDocument,
      developmentDocument,
      releaseDocument,
      contextDocument,
      statesDocument,
      glossaryDocument,
      scenariosDocument,
      documentationGuideDocument,
      guideDocument,
    ],
    repositoryRoot: '/repo',
    outputDirectory: '/repo/site',
    openapiFileName: 'example.openapi.json',
    openapi: {
      info: { title: 'Demo API', version: '1.0.0' },
      paths: {
        '/things': {
          get: { operationId: 'ListThings', summary: 'List things', tags: ['Demo'] },
        },
      },
    },
    models,
    contextTags: { demo: ['Demo'] },
    traces: [
      { id: 'REQ-DEMO-001', sources: [], workItems: ['work-items/wi-demo.md'] },
      { id: 'EX-DEMO-001-01', sources: ['backend/demo/demo_test.go'], workItems: [] },
      {
        id: 'EX-DEMO-001-02',
        sources: [],
        workItems: [],
        debt: '対応する拒否テストを確認していないため',
      },
    ],
  })

/** Every page carries the navigation twice, once for the sidebar and once for the mobile header. */
const sidebar = (html: string | undefined) =>
  (html ?? '').slice((html ?? '').indexOf('<aside class="sidebar">'))

const childLabels = (html: string | undefined) => {
  return [...sidebar(html).matchAll(/class="nav-link nav-context-child"[^>]*>([^<]+)</g)].map(
    (match) => match[1],
  )
}

describe('renderDocumentationSite', () => {
  it('renders development documents as their own navigable plane', () => {
    const result = renderDocumentationSite({
      documents: [rootDocument, developmentDocument, releaseDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: {},
      models: [],
    })

    expect(result.files['development/index.html']).toContain('href="release.html"')
    expect(result.files['development/release.html']).toContain('リリース')
    expect(sidebar(result.files['development/index.html'])).toContain(
      '<summary><a data-site-link class="nav-section-link" aria-current="page" href="#">開発文書</a></summary>',
    )
    expect(sidebar(result.files['development/index.html'])).toContain(
      '<summary><a data-site-link class="nav-section-link" aria-current="page" href="#">開発文書</a></summary><ul class="nav-tree">',
    )
    expect(
      sidebar(result.files['development/index.html']).indexOf('>設計文書</a></summary>'),
    ).toBeLessThan(
      sidebar(result.files['development/index.html']).indexOf('>開発文書</a></summary>'),
    )
    expect(sidebar(result.files['development/index.html'])).toContain('>リリース</a>')
  })

  it('renders a linked multi-page documentation site', () => {
    const result = site()

    expect(Object.keys(result.files).sort()).toEqual([
      'api/index.html',
      'development/index.html',
      'development/release.html',
      'docs/design/index.html',
      'docs/requirements/index.html',
      'docs/requirements/product-overview.html',
      'docs/requirements/quality.html',
      'domain/demo/glossary.html',
      'domain/demo/index.html',
      'domain/demo/scenarios.html',
      'domain/demo/states.html',
      'domain/glossary.html',
      'domain/index.html',
      'domain/scenarios.html',
      'domain/standards.html',
      'domain/structure.html',
      'format/documentation-guide.html',
      'format/index.html',
      'format/work-item-format.html',
      'index.html',
      'models/example-demo-internalrecord.html',
      'models/index.html',
      'operations/index.html',
      'operations/runbooks/async-jobs.html',
      'operations/service-management.html',
      'reference/index.html',
      'traceability/index.html',
    ])
    // トップページは docs/README.md そのものである。同じ案内をレンダラー側に書かない。
    expect(result.files['index.html']).toContain(
      '<h2 id="whole-system-context-map">Context Map</h2>',
    )
    expect(result.files['index.html']).not.toContain('目的から探す')
    expect(result.files['index.html']).not.toContain('class="card"')
    expect(sidebar(result.files['index.html'])).toContain('href="domain/demo/index.html"')
    expect(result.files['domain/demo/index.html']).toContain('href="states.html"')
    expect(result.files['domain/demo/states.html']).toContain('stateDiagram-v2')
    expect(result.files['domain/demo/states.html']).toContain('state_3 --&gt; state_1: Reset')
    expect(result.files['domain/demo/states.html']).not.toContain('Reset [')
    expect(result.files['domain/demo/scenarios.html']).toContain('class="scenario-keyword when"')
    expect(result.files['domain/demo/scenarios.html']).toContain('class="scenario-keyword but"')
    expect(result.files['traceability/index.html']).toContain('EX-DEMO-001-01')
    expect(result.files['traceability/index.html']).toContain('backend/demo/demo_test.go')
    expect(result.files['traceability/index.html']).toContain('規則／例')
    expect(result.files['traceability/index.html']).toContain('作業項目')
    expect(result.files['traceability/index.html']).not.toContain('Rule / Example')
    expect(result.files['traceability/index.html']).not.toContain('work item')
    expect(result.files['traceability/index.html']).toContain(
      '負債: 対応する拒否テストを確認していないため',
    )
    expect(result.files['index.html']).toContain('class="mermaid"')
    expect(result.files['api/index.html']).toContain('swagger-ui-bundle.js')
    expect(result.files['api/index.html']).toContain('class="swagger-shell"')
    expect(result.files['api/index.html']).toContain('SwaggerUIBundle({spec:specification')
    expect(result.files['api/index.html']).toContain('<h1>API リファレンス</h1>')
    expect(result.files['api/index.html']).toContain('レスポンスボディ')
    expect(result.files['api/index.html']).toContain('リクエストボディ')
    expect(result.files['api/index.html']).not.toContain('URL.createObjectURL(new Blob(')
    expect(result.files['api/index.html']).toContain('href="../openapi/example.openapi.json"')
    expect(result.files['api/index.html']).toContain('../openapi/example.openapi.json')
    expect(result.files['models/index.html']).toContain('InternalRecord')
    expect(result.files['models/index.html']).toContain('data-model-search')
    expect(result.files['models/index.html']).toContain('assets/site.js')
    expect(result.assets['site.css']).toContain('--diagram-line:#b9c8ff')
    expect(result.assets['site.js']).toContain("layout:'dagre',look:'classic'")
    expect(result.files['models/example-demo-internalrecord.html']).toContain('API 非公開')
    expect(result.files['models/example-demo-internalrecord.html']).toContain('minLength: 3')
  })

  // 印は文書と位置で絞る。方法論文書の英語本文にも Gherkin と同じ語が現れる。
  it('marks every scenario step, the actor, and nothing outside a scenario document', () => {
    const scenarios = site().files['domain/demo/scenarios.html'] ?? ''

    // 残りがコード片から始まるステップでも印が付く。
    expect(scenarios).toContain(
      '<span class="scenario-keyword then">Then</span> <code>demo:run</code>',
    )
    expect(scenarios).toContain('<span class="scenario-actor">Primary actor</span>')
    expect([...scenarios.matchAll(/class="scenario-keyword /g)].length).toBe(7)

    // WORK_ITEM_FORMAT.md の "When an item enters ..." は本文であってステップではない。
    expect(site().files['format/work-item-format.html']).not.toContain('scenario-keyword')
  })

  it('names context children by content and lists them in canonical order', () => {
    const page = site().files['domain/demo/index.html']

    expect(childLabels(page)).toEqual(['Glossary', 'State Transitions', 'シナリオ'])
    expect(page).not.toContain('>glossary.md<')
  })

  it('keeps a parent page distinct from its children and removes Japanese possession', () => {
    const result = renderDocumentationSite({
      documents: [
        rootDocument,
        developmentDocument,
        contextDocument,
        {
          path: 'docs/domain/demo/glossary.md',
          source: '# Demo の用語集\n\n| 用語 | 定義 |\n|---|---|\n| 用語 | 定義 |\n',
        },
      ],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: {},
      models: [],
    })
    const page = result.files['development/index.html'] ?? ''

    expect(sidebar(page)).toContain(
      '<summary><a data-site-link class="nav-section-link" aria-current="page" href="#">開発文書</a></summary>',
    )
    expect(sidebar(result.files['domain/demo/index.html'])).toContain(
      '<details class="nav-section" open><summary>ドメイン設計文書</summary>',
    )
    expect(sidebar(result.files['domain/demo/index.html'])).toContain('>用語集</a>')
    expect(sidebar(result.files['domain/demo/index.html'])).not.toContain('>の用語集</a>')
  })

  it('renders system documents as a directory tree', () => {
    const result = renderDocumentationSite({
      documents: [
        // 設計の索引はサイドバーから外れるので、文書体系の表からたどる。
        { path: 'docs/README.md', source: '# システム文書\n\n[設計](design/)を読む。\n' },
        designDocument,
        applicationDesignDocument,
        apiGuidelinesDocument,
        designGuidelinesDocument,
      ],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: {},
      models: [],
    })
    const page = sidebar(result.files['docs/design/application/api-guidelines.html'])

    // 「システム設計」は 7 領域の索引にすぎないので、枝を廃して領域を区分の直下へ上げる。
    expect(page).not.toContain('>設計</a>')
    expect(page).toContain('>アプリケーション</a></summary><ul><li class="nav-item">')
    expect(page).not.toContain('>概要</a>')
    expect(page).toContain('>API ガイドライン</a>')
    expect(page).toContain('>設計ガイドライン</a>')
    expect(page).not.toContain('nav-child')
  })

  /**
   * `docs/domain/` は `docs/design/` と別のディレクトリなので、区分も別にする。区分の中は
   * ディレクトリの写しで、`docs/domain/*.md` が葉、Bounded Context のディレクトリが枝になる。
   */
  it('gives the domain documents their own division beside the design documents', () => {
    const page = sidebar(site().files['domain/glossary.html'])
    const design = page.slice(
      page.indexOf('>設計文書</a></summary>'),
      page.indexOf('>ドメイン設計文書</a></summary>'),
    )
    const domain = page.slice(
      page.indexOf('>ドメイン設計文書</a></summary>'),
      page.indexOf('>開発文書</a></summary>'),
    )

    expect(
      [...design.matchAll(/class="nav-link"[^>]*>([^<]+)/g)].map((match) => match[1]).slice(0, 2),
    ).toEqual(['要求', 'プロダクト概要'])
    expect(design).not.toContain('ドメイン設計')
    expect(
      [...domain.matchAll(/class="nav-link"[^>]*>([^<]+)/g)].map((match) => match[1]).slice(0, 5),
    ).toEqual(['用語集', '全体の標準仕様', '構造', 'システム横断シナリオ', 'Demo'])
  })

  /**
   * 子は常に HTML へ載せ、開くかどうかだけを現在位置で決める。到達させるために枝を
   * 開いたままにすると、無関係なページを開いただけで一覧がすべて展開される。
   */
  it('opens a branch only while the current page is inside it', () => {
    const away = sidebar(site().files['docs/design/index.html'])
    const inside = sidebar(site().files['docs/requirements/quality.html'])

    expect(away).toContain('<details class="nav-directory"><summary>')
    expect(away).not.toContain('<details class="nav-directory" open>')
    expect(away).toContain('>品質要求</a>')
    expect(inside).toContain('<details class="nav-directory" open>')
  })

  it('carries the operations plane down to the runbooks', () => {
    const result = site()
    const page = sidebar(result.files['operations/index.html'])

    expect(result.files['operations/service-management.html']).toContain('平常時の管理')
    expect(result.files['operations/runbooks/async-jobs.html']).toContain('停滞したときの手順')
    expect(page).toContain('<details class="nav-section" open><summary><a')
    expect(page).toContain('>サービス管理</a>')
    expect(page).toContain('<summary><span class="nav-label">運用手順</span></summary>')
    // 枝の名札が種類を言うので、子の名札は題名の末尾の種類名を繰り返さない。
    expect(page).toContain('>非同期ジョブ</a>')
    expect(page).not.toContain('>非同期ジョブのランブック</a>')
    expect(result.files['docs/operations/index.html']).toBeUndefined()
  })

  /** 名札、パンくず、`<title>` は地の文なので、表題の記法を読み手へ出さない。 */
  it('reads a title as plain text wherever it is not the heading itself', () => {
    const result = renderDocumentationSite({
      documents: [
        rootDocument,
        operationsIndexDocument,
        {
          path: 'docs/runbooks/token-endpoint-error-rate.md',
          source: '# `/token` の **エラー率** と [復旧](../operations/README.md)\n\n手順。\n',
        },
      ],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: {},
      models: [],
    })
    const page = result.files['operations/runbooks/token-endpoint-error-rate.html'] ?? ''

    expect(sidebar(page)).toContain('>/token の エラー率 と 復旧</a>')
    expect(page).toContain('<title>/token の エラー率 と 復旧 · IdMagic ドキュメント</title>')
    expect(sidebar(page)).not.toContain('`')
    expect(sidebar(page)).not.toContain('**')
    // 本文の見出しは Markdown として組む。地の文にするのは派生した表示だけである。
    expect(page).toContain('<code>/token</code>')
  })

  it('lists a document own headings beside its body', () => {
    const page = site().files['domain/demo/scenarios.html'] ?? ''

    expect(page).toContain('<nav class="page-toc" aria-label="このページの内容">')
    expect(page).toContain('href="#context-demo-scenarios-rule-req-demo-001-a-demo-runs"')
    expect(page).toContain(
      'class="page-toc-h3"><a data-site-link href="#context-demo-scenarios-example-ex-demo-001-01-a-ready-demo-starts"',
    )
    // 見出しが一つしかないページに目次を出しても、本文を繰り返すだけである。
    expect(site().files['docs/requirements/product-overview.html']).not.toContain('page-toc')
  })

  it('names the complete site IdMagic ドキュメント', () => {
    const result = site()

    expect(result.files['index.html']).toContain('<title>IdMagic ドキュメント</title>')
    // トップページは docs/README.md そのものなので、見出しはその文書の H1 である。
    expect(result.files['index.html']).toContain('>Whole-System Specification</h1>')
    expect(result.files['index.html']).not.toContain('class="breadcrumbs"')
    expect(result.files['domain/demo/index.html']).toContain(
      '<title>Demo · IdMagic ドキュメント</title>',
    )
    expect(result.files['domain/demo/index.html']).toContain('>IdMagic ドキュメント</a>')
  })

  it('closes top-level sections on the landing page and opens only the current section elsewhere', () => {
    const result = site()
    const top = sidebar(result.files['index.html'])

    expect(
      [...top.matchAll(/<details class="nav-section"><summary>(?:<a[^>]*>)?([^<]+)/g)].map(
        (match) => match[1],
      ),
    ).toEqual([
      '設計文書',
      'ドメイン設計文書',
      '開発文書',
      '運用文書',
      'リファレンス',
      'フォーマット',
    ])
    expect(top).not.toContain('<details class="nav-section" open>')

    const context = sidebar(result.files['domain/demo/index.html'])
    expect(context.match(/<details class="nav-section" open>/g)).toHaveLength(1)
    expect(context).toContain('<details class="nav-section" open><summary>')
  })

  // 地の文を主役に置く体裁なので、枠と影で本文を囲まず、階層は字下げで示す。
  it('sets the body on the page background and shows hierarchy by indent alone', () => {
    const css = site().assets['site.css'] ?? ''

    expect(css).toContain('--measure:')
    expect(css).not.toMatch(/\.document[^{]*\{[^}]*box-shadow/)
    expect(css).not.toMatch(/\.nav-tree\s*(?:ul)?\{[^}]*border-left/)
    expect(css).toContain('.nav-tree ul{margin:0;padding:0 0 0 18px;list-style:none}')
    // 葉の頭は、兄弟の枝が持つ開閉記号の幅だけ下げてそろえる。
    expect(css).toContain('.nav-item>.nav-link{padding-left:26px}')
    // 子を持つかどうかは文書の性質ではないので、名札の色と太さには出さない。
    expect(css).not.toMatch(/\.nav-branch>\.nav-link[^{]*\{[^}]*(?:color|font-weight)/)
    expect(css).toContain('.page-toc')
    expect(css).toContain('.page-toc a[aria-current=location]')
  })

  it('uses the full reference width without Swagger UI wrapper padding', () => {
    const css = site().assets['site.css']

    expect(css).toContain('.swagger-shell .swagger-ui .wrapper{max-width:none;padding-inline:0}')
    expect(css).toMatch(/main:has\(\.swagger-shell\)\{[^}]*max-width:none/)
  })

  /**
   * Swagger UI 5.x は内部参照でも `baseDoc` の URI を取得して解決するため、`file://` で開いた
   * 生成物では解決できない。参照を残さなければ取得が起きないので、組み立て時に展開して渡す。
   */
  it('hands Swagger UI a document with no reference left to resolve', () => {
    const result = renderDocumentationSite({
      documents: [rootDocument, contextDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: {
        paths: {
          '/things': {
            get: {
              operationId: 'ListThings',
              tags: ['Demo'],
              responses: { '200': { schema: { $ref: '#/components/schemas/Thing' } } },
            } as never,
          },
        },
        components: {
          schemas: {
            Thing: {
              type: 'object',
              properties: {
                child: { $ref: '#/components/schemas/Thing' },
                name: { type: 'string' },
              },
            },
          },
        },
      },
      models: [],
      contextTags: { demo: ['Demo'] },
    })
    const page = result.files['api/index.html'] ?? ''

    expect(page).toContain('SwaggerUIBundle({spec:specification')
    expect(page).not.toContain('URL.createObjectURL(new Blob(')
    expect(page).not.toContain('url:"../openapi/example.openapi.json"')
    // 展開した節は元のモデル名を `$$ref` で保つ。
    expect(page).toContain('"$$ref":"#/components/schemas/Thing"')
    // 自分自身へ戻る参照だけは打ち切る。展開できないまま残すと解決の取得が起きる。
    expect(page).toContain('Thing と同じ形の入れ子。')
    expect(page).not.toContain('"$ref":')
    // 生の JSON を読みたいときの導線は残す。
    expect(page).toContain('href="../openapi/example.openapi.json"')
  })

  it('keeps a glossary term on one line', () => {
    expect(site().files['domain/demo/glossary.html']).toContain('<table class="term-table">')
    expect(site().assets['site.css']).toContain('.term-table td:first-child{white-space:nowrap}')
  })

  it('leads from a context to its own operations and models', () => {
    const result = site()
    const page = result.files['domain/demo/index.html'] ?? ''

    expect(page).toContain('API とモデル')
    expect(page).toContain('>API</h3>')
    expect(page).toContain('<th scope="col">説明</th>')
    expect(page).toContain('href="../../api/index.html?tag=Demo"')
    expect(page).toContain('List things')
    expect(page).toContain('href="../../models/index.html#context-demo"')
    expect(page).toContain('href="../../models/example-demo-internalrecord.html"')
    expect(result.files['models/index.html']).toContain('<h2 id="context-demo">Demo</h2>')
  })

  it('uses an OpenAPI description before exposing an operation identifier', () => {
    const result = renderDocumentationSite({
      documents: [rootDocument, contextDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: {
        paths: {
          '/things': {
            get: {
              operationId: 'ListThings',
              description: '利用可能な Thing を一覧する。',
              tags: ['Demo'],
            },
          },
        },
      },
      models: [],
      contextTags: { demo: ['Demo'] },
    })
    const page = result.files['domain/demo/index.html'] ?? ''

    expect(page).toContain('利用可能な Thing を一覧する。')
    expect(page).not.toContain('<td>ListThings</td>')
  })

  /**
   * 見出しは日本語で書く。markdown-it はリンクの href を百分率符号化して渡すので、
   * 断片をそのまま綴りに直すと、見出し側の綴りと一致しない別名を指すことになる。
   */
  it('resolves a cross-document link whose fragment is a Japanese heading', () => {
    const result = renderDocumentationSite({
      documents: [
        rootDocument,
        {
          path: 'docs/domain/glossary.md',
          source:
            '# 用語集\n\n目標は [品質要求](../requirements/quality.md#可用性の目標) が定める。\n',
        },
        {
          path: 'docs/requirements/quality.md',
          source: '# 品質要求\n\n## 可用性の目標\n\n目標値。\n',
        },
      ],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: {},
      models: [],
    })

    expect(result.files['docs/requirements/quality.html']).toContain(
      'id="whole-system-requirements-quality-md-可用性の目標"',
    )
    expect(result.files['domain/glossary.html']).toContain(
      'href="../docs/requirements/quality.html#whole-system-requirements-quality-md-可用性の目標"',
    )
  })

  /** リポジトリでは読めるディレクトリへの参照が、生成サイトでは行き先を失う。 */
  it('sends a directory link to that directory index page', () => {
    const result = renderDocumentationSite({
      documents: [
        {
          path: 'docs/README.md',
          source: '# システム文書\n\n[設計](design/)と[要求](requirements/)を読む。\n',
        },
        designDocument,
        requirementsIndexDocument,
      ],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: {},
      models: [],
    })
    const page = result.files['index.html'] ?? ''

    expect(page).toContain('href="docs/design/index.html"')
    expect(page).toContain('href="docs/requirements/index.html"')
    expect(page).not.toContain('href="design/"')
  })

  it('renders doc comments as the Markdown they are written in', () => {
    const result = renderDocumentationSite({
      documents: [rootDocument, contextDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: { paths: {} },
      models: [
        {
          ...models[0]!,
          doc: 'Expires at `expires_at`.',
          properties: [
            {
              name: 'id',
              type: 'string',
              optional: false,
              doc: 'Paired with `expires_at`.',
              constraints: [],
              references: [],
            },
          ],
          members: [{ name: 'ready', value: '"ready"', doc: 'Set once `expires_at` passes.' }],
        },
      ],
    })
    const page = result.files['models/example-demo-internalrecord.html'] ?? ''

    expect(page).toContain('Expires at <code>expires_at</code>.')
    expect(page).toContain('Paired with <code>expires_at</code>.')
    expect(page).toContain('Set once <code>expires_at</code> passes.')
    expect(result.files['models/index.html']).toContain('Expires at <code>expires_at</code>.')
  })

  it('explains undocumented model properties from their established field meaning', () => {
    const result = renderDocumentationSite({
      documents: [rootDocument, contextDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: { paths: {} },
      models: [
        {
          ...models[0]!,
          properties: [
            {
              name: 'tenantId',
              type: 'string',
              optional: false,
              constraints: [],
              references: [],
            },
          ],
        },
      ],
    })
    const page = result.files['models/example-demo-internalrecord.html'] ?? ''

    expect(page).toContain('対象テナントの識別子。')
    expect(page).not.toContain('説明なし')
  })

  it('rejects unclassified operations', () => {
    expect(() =>
      renderDocumentationSite({
        documents: [rootDocument],
        repositoryRoot: '/repo',
        outputDirectory: '/repo/site',
        openapiFileName: 'example.openapi.json',
        openapi: { paths: { '/things': { get: { operationId: 'ListThings' } } } },
        models: [],
      }),
    ).toThrow('has no owning context tag')
  })

  it('escapes model and OpenAPI content', () => {
    const result = renderDocumentationSite({
      documents: [rootDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/site',
      openapiFileName: 'example.openapi.json',
      openapi: { info: { title: '</script><script>bad()</script>' }, paths: {} },
      models: [
        {
          ...models[0]!,
          name: 'Example.Demo.Escaped',
          shortName: 'Escaped',
          doc: '<img src=x onerror=bad()>',
        },
      ],
    })
    expect(result.files['api/index.html']).not.toContain('</script><script>bad()')
    expect(result.files['models/example-demo-escaped.html']).toContain(
      '&lt;img src=x onerror=bad()&gt;',
    )
  })
})

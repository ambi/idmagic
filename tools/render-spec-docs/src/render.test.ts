import { describe, expect, it } from 'bun:test'
import { renderSpecificationSite } from './render.ts'
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
| [contexts/demo/README.md](contexts/demo/README.md) | The demo context |
`,
}

const rootGlossaryDocument = {
  path: 'docs/glossary.md',
  source: `# Glossary

| Term | Definition |
|---|---|
| InterfaceStability | 外部契約としての安定性の区分。 |
`,
}

const rootStructureDocument = {
  path: 'docs/structure.md',
  source: '# Structure\n\nディレクトリの配置。\n',
}

const requirementsIndexDocument = {
  path: 'docs/requirements/README.md',
  source: '# 要求\n\n要求の分類。\n',
}

const qualityDocument = {
  path: 'docs/requirements/quality.md',
  source: '# 品質要求\n\nシステム品質の目標。\n',
}

const contextDocument = {
  path: 'docs/contexts/demo/README.md',
  source: `# Demo

The demo context.

| File | Content |
|---|---|
| [states.md](states.md) | 状態と遷移 |
| [scenarios.feature.md](scenarios.feature.md) | 受け入れシナリオ |
`,
}

const statesDocument = {
  path: 'docs/contexts/demo/states.md',
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
  path: 'docs/contexts/demo/scenarios.feature.md',
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
  path: 'docs/contexts/demo/glossary.md',
  source: `# Demo Glossary

| Term | Definition |
|---|---|
| DemoRun | 1 回の実行。 |
`,
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
  renderSpecificationSite({
    documents: [
      rootDocument,
      rootGlossaryDocument,
      rootStructureDocument,
      requirementsIndexDocument,
      qualityDocument,
      contextDocument,
      statesDocument,
      glossaryDocument,
      scenariosDocument,
      guideDocument,
    ],
    repositoryRoot: '/repo',
    outputDirectory: '/repo/spec/generated/docs',
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
  const contexts = sidebar(html).match(
    /<details class="nav-group"[^>]*><summary>コンテキスト別<\/summary>([\s\S]*?)<\/details>/,
  )?.[1]
  return [...(contexts ?? '').matchAll(/class="nav-child"[^>]*>([^<]+)</g)].map((match) => match[1])
}

/** 深さつきの子項目。`nav-child` が 1 段目、`nav-child-2` が 2 段目を表す。 */
const nestedLabels = (html: string | undefined) =>
  [...sidebar(html).matchAll(/class="nav-child(-\d)?"[^>]*>([^<]+)</g)].map(
    (match) => `${match[1] ? match[1].slice(1) : '1'}:${match[2]}`,
  )

describe('renderSpecificationSite', () => {
  it('renders development documents as their own navigable plane', () => {
    const result = renderSpecificationSite({
      documents: [rootDocument, developmentDocument, releaseDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/spec/generated/docs',
      openapiFileName: 'example.openapi.json',
      openapi: {},
      models: [],
    })

    expect(result.files['development/index.html']).toContain('href="release.html"')
    expect(result.files['development/release.html']).toContain('リリース')
    expect(sidebar(result.files['development/index.html'])).toContain('<summary>開発</summary>')
    expect(nestedLabels(result.files['development/index.html'])).toEqual([])
  })

  it('renders a linked multi-page specification site', () => {
    const result = site()

    expect(Object.keys(result.files).sort()).toEqual([
      'api/index.html',
      'contexts/demo/glossary.html',
      'contexts/demo/index.html',
      'contexts/demo/scenarios.html',
      'contexts/demo/states.html',
      'index.html',
      'method/work-item-format.html',
      'models/example-demo-internalrecord.html',
      'models/index.html',
      'specification/glossary.html',
      'specification/index.html',
      'specification/requirements/index.html',
      'specification/requirements/quality.html',
      'specification/structure.html',
      'traceability/index.html',
    ])
    expect(result.files['index.html']).toContain('Whole-System Specification')
    expect(result.files['index.html']).toContain('href="contexts/demo/index.html"')
    expect(result.files['contexts/demo/index.html']).toContain('href="states.html"')
    expect(result.files['contexts/demo/states.html']).toContain('stateDiagram-v2')
    expect(result.files['contexts/demo/states.html']).toContain('state_3 --&gt; state_1: Reset')
    expect(result.files['contexts/demo/states.html']).not.toContain('Reset [')
    expect(result.files['contexts/demo/scenarios.html']).toContain('class="scenario-keyword when"')
    expect(result.files['contexts/demo/scenarios.html']).toContain('class="scenario-keyword but"')
    expect(result.files['traceability/index.html']).toContain('EX-DEMO-001-01')
    expect(result.files['traceability/index.html']).toContain('backend/demo/demo_test.go')
    expect(result.files['traceability/index.html']).toContain('規則／例')
    expect(result.files['traceability/index.html']).toContain('作業項目')
    expect(result.files['traceability/index.html']).not.toContain('Rule / Example')
    expect(result.files['traceability/index.html']).not.toContain('work item')
    expect(result.files['traceability/index.html']).toContain(
      '負債: 対応する拒否テストを確認していないため',
    )
    expect(result.files['specification/index.html']).toContain('class="mermaid"')
    expect(result.files['api/index.html']).toContain('swagger-ui-bundle.js')
    expect(result.files['api/index.html']).toContain('class="swagger-shell"')
    expect(result.files['api/index.html']).toContain('"/things"')
    expect(result.files['api/index.html']).toContain('<h1>API リファレンス</h1>')
    expect(result.files['api/index.html']).toContain('レスポンスボディ')
    expect(result.files['api/index.html']).toContain('リクエストボディ')
    expect(result.files['api/index.html']).toContain('URL.createObjectURL(new Blob(')
    expect(result.files['api/index.html']).toContain('url:specificationUrl')
    expect(result.files['api/index.html']).not.toContain('SwaggerUIBundle({spec:')
    expect(result.files['api/index.html']).toContain('../../openapi/example.openapi.json')
    expect(result.files['models/index.html']).toContain('InternalRecord')
    expect(result.files['models/index.html']).toContain('data-model-search')
    expect(result.files['models/index.html']).toContain('assets/site.js')
    expect(result.assets['site.css']).toContain('--diagram-line:#b9c8ff')
    expect(result.files['models/example-demo-internalrecord.html']).toContain('API 非公開')
    expect(result.files['models/example-demo-internalrecord.html']).toContain('minLength: 3')
  })

  // 印は文書と位置で絞る。方法論文書の英語本文にも Gherkin と同じ語が現れる。
  it('marks every scenario step, the actor, and nothing outside a scenario document', () => {
    const scenarios = site().files['contexts/demo/scenarios.html'] ?? ''

    // 残りがコード片から始まるステップでも印が付く。
    expect(scenarios).toContain(
      '<span class="scenario-keyword then">Then</span> <code>demo:run</code>',
    )
    expect(scenarios).toContain('<span class="scenario-actor">Primary actor</span>')
    expect([...scenarios.matchAll(/class="scenario-keyword /g)].length).toBe(7)

    // WORK_ITEM_FORMAT.md の "When an item enters ..." は本文であってステップではない。
    expect(site().files['method/work-item-format.html']).not.toContain('scenario-keyword')
  })

  it('names context children by content and lists them in canonical order', () => {
    const page = site().files['contexts/demo/index.html']

    expect(childLabels(page)).toEqual(['Glossary', 'State Transitions', 'シナリオ'])
    expect(page).not.toContain('>glossary.md<')
  })

  /**
   * 上から下へ分解した体系は、一段の一覧では読めない。ディレクトリの索引を
   * 親に、その配下の文書を子に置き、名札は各文書自身の題名にする。英語の
   * ディレクトリ名を接頭辞として足すと、日本語の題名の前に別の語彙が並ぶ。
   */
  it('lists the root and directory indexes at one level without a single-child wrapper', () => {
    const result = site()
    expect(nestedLabels(result.files['specification/index.html'])).toEqual(['1:品質要求'])
    expect(sidebar(result.files['specification/index.html'])).toContain(
      '>Whole-System Specification</a>',
    )
    expect(sidebar(result.files['specification/index.html'])).toContain('>要求</a>')
    expect(sidebar(result.files['contexts/demo/index.html'])).toContain(
      'href="../../specification/structure.html"',
    )
  })

  it('folds every navigation group the same way, with method alone starting folded', () => {
    const result = site()
    const groups = (page: string) =>
      [
        ...sidebar(result.files[page]).matchAll(
          /<details class="nav-group"( open)?><summary>([^<]+)/g,
        ),
      ].map((match) => `${match[2]}${match[1] ? ' open' : ''}`)

    expect(groups('contexts/demo/index.html')).toEqual([
      '方法論',
      'システム',
      'コンテキスト別 open',
      '参照 open',
    ])
    expect(groups('specification/index.html')).toEqual([
      '方法論',
      'システム open',
      'コンテキスト別',
      '参照 open',
    ])
    expect(groups('method/work-item-format.html')[0]).toBe('方法論 open')
  })

  it('uses the full reference width without Swagger UI wrapper padding', () => {
    const css = site().assets['site.css']

    expect(css).toContain('main:has(.swagger-shell){width:calc(100% - 340px);max-width:none}')
    expect(css).toContain('.swagger-shell .swagger-ui .wrapper{max-width:none;padding-inline:0}')
    expect(css).toContain('main:has(.swagger-shell){width:auto}')
  })

  it('keeps a glossary term on one line', () => {
    expect(site().files['contexts/demo/glossary.html']).toContain('<table class="term-table">')
    expect(site().assets['site.css']).toContain('.term-table td:first-child{white-space:nowrap}')
  })

  it('leads from a context to its own operations and models', () => {
    const result = site()
    const page = result.files['contexts/demo/index.html'] ?? ''

    expect(page).toContain('API とモデル')
    expect(page).toContain('href="../../api/index.html?tag=Demo"')
    expect(page).toContain('List things')
    expect(page).toContain('href="../../models/index.html#context-demo"')
    expect(page).toContain('href="../../models/example-demo-internalrecord.html"')
    expect(result.files['models/index.html']).toContain('<h2 id="context-demo">Demo</h2>')
  })

  /**
   * 見出しは日本語で書く。markdown-it はリンクの href を百分率符号化して渡すので、
   * 断片をそのまま綴りに直すと、見出し側の綴りと一致しない別名を指すことになる。
   */
  it('resolves a cross-document link whose fragment is a Japanese heading', () => {
    const result = renderSpecificationSite({
      documents: [
        rootDocument,
        {
          path: 'docs/glossary.md',
          source:
            '# 用語集\n\n目標は [品質要求](requirements/quality.md#可用性の目標) が定める。\n',
        },
        {
          path: 'docs/requirements/quality.md',
          source: '# 品質要求\n\n## 可用性の目標\n\n目標値。\n',
        },
      ],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/spec/generated/docs',
      openapiFileName: 'example.openapi.json',
      openapi: {},
      models: [],
    })

    expect(result.files['specification/requirements/quality.html']).toContain(
      'id="whole-system-requirements-quality-md-可用性の目標"',
    )
    expect(result.files['specification/glossary.html']).toContain(
      'href="requirements/quality.html#whole-system-requirements-quality-md-可用性の目標"',
    )
  })

  it('renders doc comments as the Markdown they are written in', () => {
    const result = renderSpecificationSite({
      documents: [rootDocument, contextDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/spec/generated/docs',
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

  it('rejects unclassified operations', () => {
    expect(() =>
      renderSpecificationSite({
        documents: [rootDocument],
        repositoryRoot: '/repo',
        outputDirectory: '/repo/spec/generated/docs',
        openapiFileName: 'example.openapi.json',
        openapi: { paths: { '/things': { get: { operationId: 'ListThings' } } } },
        models: [],
      }),
    ).toThrow('has no owning context tag')
  })

  it('escapes model and OpenAPI content', () => {
    const result = renderSpecificationSite({
      documents: [rootDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/spec/generated/docs',
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

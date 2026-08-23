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

const contextDocument = {
  path: 'docs/contexts/demo/README.md',
  source: `# Demo

The demo context.

| File | Content |
|---|---|
| [states.md](states.md) | 状態と遷移 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
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
  path: 'docs/contexts/demo/scenarios.md',
  source: `# Demo Scenarios

### REQ-DEMO-001: a demo runs
- ACTOR Developer
- GIVEN a ready demo
- WHEN the demo starts
- THEN the demo is running
  - ALT start is forbidden → the demo stays ready
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
  source: '# Work Item Format\n\nEnglish method guidance.\n',
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
  })

/** Every page carries the navigation twice, once for the sidebar and once for the mobile header. */
const sidebar = (html: string | undefined) =>
  (html ?? '').slice((html ?? '').indexOf('<aside class="sidebar">'))

const childLabels = (html: string | undefined) =>
  [...sidebar(html).matchAll(/class="nav-child"[^>]*>([^<]+)</g)].map((match) => match[1])

describe('renderSpecificationSite', () => {
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
    expect(result.files['contexts/demo/scenarios.html']).toContain('class="scenario-keyword alt"')
    expect(result.files['specification/index.html']).toContain('class="mermaid"')
    expect(result.files['api/index.html']).toContain('swagger-ui-bundle.js')
    expect(result.files['api/index.html']).toContain('class="swagger-shell"')
    expect(result.files['api/index.html']).toContain('"/things"')
    expect(result.files['api/index.html']).toContain('../../openapi/example.openapi.json')
    expect(result.files['models/index.html']).toContain('InternalRecord')
    expect(result.files['models/index.html']).toContain('data-model-search')
    expect(result.files['models/index.html']).toContain('assets/site.js')
    expect(result.assets['site.css']).toContain('--diagram-line:#b9c8ff')
    expect(result.files['models/example-demo-internalrecord.html']).toContain('Not API-exposed')
    expect(result.files['models/example-demo-internalrecord.html']).toContain('minLength: 3')
  })

  it('names context children by content and lists them in canonical order', () => {
    const page = site().files['contexts/demo/index.html']

    expect(childLabels(page)).toEqual(['Glossary', 'State Transitions', 'Scenarios'])
    expect(page).not.toContain('>glossary.md<')
  })

  it('lists whole-system children in canonical order, only from inside', () => {
    const result = site()
    expect(childLabels(result.files['specification/index.html'])).toEqual(['Structure', 'Glossary'])
    expect(result.files['contexts/demo/index.html']).not.toContain('specification/structure.html')
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
      'Method',
      'Whole System open',
      'Contexts open',
      'References open',
    ])
    expect(groups('method/work-item-format.html')[0]).toBe('Method open')
  })

  it('keeps a glossary term on one line', () => {
    expect(site().files['contexts/demo/glossary.html']).toContain('<table class="term-table">')
    expect(site().assets['site.css']).toContain('.term-table td:first-child{white-space:nowrap}')
  })

  it('leads from a context to its own operations and models', () => {
    const result = site()
    const page = result.files['contexts/demo/index.html'] ?? ''

    expect(page).toContain('API and Models')
    expect(page).toContain('href="../../api/index.html?tag=Demo"')
    expect(page).toContain('List things')
    expect(page).toContain('href="../../models/index.html#context-demo"')
    expect(page).toContain('href="../../models/example-demo-internalrecord.html"')
    expect(result.files['models/index.html']).toContain('<h2 id="context-demo">Demo</h2>')
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

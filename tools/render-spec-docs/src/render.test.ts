import { describe, expect, it } from 'bun:test'
import { renderSpecificationSite } from './render.ts'
import type { CatalogSymbol } from './typespec-catalog.ts'

const rootDocument = {
  path: 'spec/SPECIFICATION.md',
  source: `---
context: repository
updated_at: 2026-08-11
---

# Whole-System Specification

## Overview

The whole system.

## Design

### Context Map

\`\`\`mermaid
flowchart LR
  Demo --> Other
\`\`\`
`,
}

const contextDocument = {
  path: 'spec/contexts/demo/SPECIFICATION.md',
  source: `---
context: demo
updated_at: 2026-08-11
---

# Demo Specification

## Overview

The demo context.

## State Transitions

### DemoLifecycle

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Ready | Start | allowed \\| privileged | Running | emit Started |
| Running | Finish | complete | Done | emit Completed |
| Done | Reset | "" | Ready | emit Reset |

## Scenarios

### REQ-DEMO-001: a demo runs
- ACTOR Developer
- GIVEN a ready demo
- WHEN the demo starts
- THEN the demo is running
  - ALT start is forbidden → the demo stays ready
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

describe('renderSpecificationSite', () => {
  it('renders a linked multi-page specification site', () => {
    const result = renderSpecificationSite({
      documents: [rootDocument, contextDocument, guideDocument],
      repositoryRoot: '/repo',
      outputDirectory: '/repo/spec/generated/docs',
      openapiFileName: 'example.openapi.json',
      openapi: {
        info: { title: 'Demo API', version: '1.0.0' },
        paths: {
          '/things': {
            get: { operationId: 'ListThings', tags: ['Demo'] },
          },
        },
      },
      models,
    })

    expect(Object.keys(result.files).sort()).toEqual([
      'api/index.html',
      'contexts/demo/index.html',
      'index.html',
      'method/work-item-format.html',
      'models/example-demo-internalrecord.html',
      'models/index.html',
      'specification/index.html',
    ])
    expect(result.files['index.html']).toContain('Whole-System Specification')
    expect(result.files['index.html']).toContain('href="contexts/demo/index.html"')
    expect(result.files['contexts/demo/index.html']).toContain('stateDiagram-v2')
    expect(result.files['contexts/demo/index.html']).toContain('state_3 --&gt; state_1: Reset')
    expect(result.files['contexts/demo/index.html']).not.toContain('Reset [')
    expect(result.files['contexts/demo/index.html']).toContain('class="scenario-keyword when"')
    expect(result.files['contexts/demo/index.html']).toContain('class="scenario-keyword alt"')
    expect(result.files['specification/index.html']).toContain('class="mermaid"')
    expect(result.files['api/index.html']).toContain('swagger-ui-bundle.js')
    expect(result.files['api/index.html']).toContain('class="swagger-shell"')
    expect(result.files['api/index.html']).toContain('"/things"')
    expect(result.files['api/index.html']).toContain('../../openapi/example.openapi.json')
    expect(result.files['models/index.html']).toContain('InternalRecord')
    expect(result.files['models/index.html']).toContain('data-model-search')
    expect(result.assets['site.css']).toContain('--diagram-line:#b9c8ff')
    expect(result.files['models/example-demo-internalrecord.html']).toContain('Not API-exposed')
    expect(result.files['models/example-demo-internalrecord.html']).toContain('minLength: 3')
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

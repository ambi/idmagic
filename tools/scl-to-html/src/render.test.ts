/**
 * End-to-end render smoke tests — feed each renderer a tiny fixture and
 * assert key anchors / labels / cross-refs appear in the output.
 */

import { describe, expect, it } from 'bun:test'
import { renderPage } from './page.ts'
import { renderChangesTab } from './render-changes.ts'
import { renderDecisionsTab } from './render-decisions.ts'
import { renderSclTab, sclTocItems } from './render-scl.ts'
import type { ChangeEntry, DecisionDoc, SclBundle, SclDocument, SiteInput } from './types.ts'

const sampleScl = (): SclDocument => ({
  system: 'demo',
  spec_version: '3.0',
  standards: {
    StandardA: {
      title: 'A',
      requirements: [
        { id: 'REQ', adoption: 'required', statement: 'A requirement', refs: ['interfaces.DoIt'] },
      ],
    },
    StandardB: {
      title: 'B',
      requirements: [{ id: 'REQ', adoption: 'required', statement: 'Another requirement' }],
    },
  },
  context_map: {
    Auth: { description: 'auth context', depends_on: { Directory: { via: 'published_language' } } },
    Directory: { description: 'directory context' },
  },
  glossary: {
    Foo: { definition: 'a thing' },
  },
  models: {
    Foo: {
      kind: 'entity',
      identity: ['a', 'b'],
      fields: { a: { type: 'String' }, b: { type: 'String' } },
      constraints: ['a != b'],
    },
    Bar: { kind: 'enum', values: ['X', 'Y'] },
    BarUpdated: { kind: 'event', payload: { id: { type: 'UUID' } } },
  },
  interfaces: {
    DoIt: {
      description: 'do it',
      steps: ['"{x}" を実行する'],
      input: { x: { type: 'String' } },
      output: { y: { type: 'Foo' } },
      emits: ['BarUpdated'],
      requires: ['input.x != ""'],
      ensures: ['output.y.a != ""'],
      access: { policies: ['P'], resource: { type: 'Foo', id: 'input.x' } },
      bindings: [
        { kind: 'http', method: 'POST', path: '/do' },
        { kind: 'schedule', every: '1m' },
      ],
    },
  },
  states: {
    FooLifecycle: {
      target: 'Foo',
      initial: 'Draft',
      terminal: ['Done'],
      transitions: [
        { from: 'Draft', event: 'Submit', to: 'Ready' },
        { from: 'Ready', event: 'Finish', to: 'Done' },
      ],
    },
  },
  authorization: {
    principals: {
      User: { type: 'Foo', matches: ['principal.a != ""'] },
    },
    policies: {
      P: { effect: 'permit', principal: 'User', when: 'resource.a != ""' },
    },
  },
  scenarios: {
    'demo の流れ': {
      actor: 'User',
      main_success: ['DoIt を呼ぶ', 'BarUpdated が発行される'],
    },
  },
  objectives: {
    O: {
      interface: 'DoIt',
      indicator: 'measurement.latency_ms < 200',
      target: 0.95,
      window: '30d',
    },
  },
  flows: {
    Demo: {
      entry: 'Login',
      views: {
        Login: {
          sees: 'ログイン画面(メールアドレス入力フォーム、パスワード入力フォーム)',
          does: [{ action: 'success', does: '入力して、ログインボタンをクリックする', to: 'Done', interface: 'DoIt' }],
        },
        Done: {
          sees: 'ログイン完了画面',
        },
      },
    },
  },
})

describe('renderSclTab', () => {
  const html = renderSclTab(sampleScl())

  it('contains every present-section anchor', () => {
    expect(html).toContain('id="glossary"')
    expect(html).toContain('id="models"')
    expect(html).toContain('id="interfaces"')
    expect(html).toContain('id="states"')
    expect(html).toContain('id="authorization"')
    expect(html).toContain('id="scenarios"')
    expect(html).toContain('id="objectives"')
    expect(html).toContain('id="flows"')
  })

  it('renders the spec version in the SCL tab header', () => {
    expect(html).toContain('spec 3.0')
  })

  it('renders composite identity as a comma-joined string', () => {
    expect(html).toContain('a, b')
  })

  it('linkifies known names inside scenario steps', () => {
    expect(html).toContain('href="#scl-demo/interface/DoIt"')
    expect(html).toContain('href="#scl-demo/model/BarUpdated"')
  })

  it('renders the http binding method as a badge', () => {
    expect(html).toContain('badge-method-post')
  })

  it('renders the schedule binding kind', () => {
    expect(html).toContain('every: 1m')
  })

  it('renders local model and interface contracts beside their owners', () => {
    expect(html).toContain('Model constraints')
    expect(html).toContain('a != b')
    expect(html).toContain('requires')
    expect(html).toContain('input.x != &quot;&quot;')
    expect(html).toContain('ensures')
    expect(html).toContain('output.y.a != &quot;&quot;')
  })

  it('renders protected access, authorization policy, and SLO details', () => {
    expect(html).toContain('access-protected')
    expect(html).toContain('id="scl-demo/authorization_policy/P"')
    expect(html).toContain('effect-permit')
    expect(html).toContain('measurement.latency_ms &lt; 200')
    expect(html).toContain('30d')
  })

  it('renders canonical references and derives collision-free requirement anchors', () => {
    expect(html).toContain('demo/model/Foo')
    expect(html).toContain('demo/interface/DoIt')
    expect(html).toContain('demo/standard_requirement/StandardA/REQ')
    expect(html).toContain('id="scl-demo/standard_requirement/StandardA/REQ"')
    expect(html).toContain('id="scl-demo/standard_requirement/StandardB/REQ"')
    expect(html).toContain('href="#scl-demo/interface/DoIt"')
  })

  it('uses a defined canonical target for every canonical internal href', () => {
    const ids = new Set([...html.matchAll(/\bid="([^"]+)"/g)].map((match) => match[1]))
    const hrefs = [...html.matchAll(/\bhref="#(scl-[^"]+)"/g)].map((match) => match[1])
    expect(hrefs.length).toBeGreaterThan(0)
    for (const href of hrefs) expect(ids.has(href ?? '')).toBe(true)
  })

  it('renders derived diagrams for context map, states, and flows', () => {
    expect(html).toContain('id="diagram-context-map"')
    expect(html).toContain('id="diagram-state-foolifecycle"')
    expect(html).toContain('id="diagram-flow-demo"')
    expect(html).toContain('data-diagram-svg')
    expect(html).toContain('published_language')
    expect(html).toContain('Submit')
    expect(html).toContain('success')
  })

  it('renders flow view sees/does content', () => {
    expect(html).toContain('ログイン画面(メールアドレス入力フォーム、パスワード入力フォーム)')
    expect(html).toContain('入力して、ログインボタンをクリックする')
    expect(html).toContain('ログイン完了画面')
  })

  it('lists section titles in TOC items', () => {
    const items = sclTocItems(sampleScl())
    expect(items.map((i) => i.id)).toContain('models')
    expect(items.map((i) => i.id)).toContain('scenarios')
  })
})

describe('renderSclTab canonical authorization anchors', () => {
  const html = renderSclTab({
    system: 'demo',
    spec_version: '3.0',
    models: {
      Subject: { kind: 'entity', identity: 'id', fields: { id: { type: 'UUID' } } },
    },
    authorization: {
      resources: { Same: { description: 'resource' } },
      principals: { Same: { type: 'Subject', matches: ['principal.id != ""'] } },
      policies: { Same: { effect: 'permit', principal: 'Same' } },
    },
  })

  it('keeps identically named authorization kinds distinct', () => {
    expect(html).toContain('id="scl-demo/authorization_resource/Same"')
    expect(html).toContain('id="scl-demo/authorization_principal/Same"')
    expect(html).toContain('id="scl-demo/authorization_policy/Same"')
  })
})

describe('renderSclTab with context documents', () => {
  const bundle: SclBundle = {
    root: {
      system: 'demo',
      spec_version: '3.0',
      context_map: {
        Application: { path: 'contexts/application.yaml' },
      },
    },
    contexts: [
      {
        name: 'Application',
        path: 'contexts/application.yaml',
        document: {
          system: 'demo',
          spec_version: '3.0',
          context: 'Application',
          models: {
            Application: {
              kind: 'entity',
              identity: 'id',
              fields: { id: { type: 'UUID' } },
            },
          },
          scenarios: {
            'application flow': {
              actor: 'Application',
              main_success: ['Application を読む'],
            },
          },
        },
      },
    ],
  }
  const html = renderSclTab(bundle)

  it('renders root context map and referenced context documents in one SCL tab', () => {
    expect(html).toContain('id="context_map"')
    expect(html).toContain('id="context-application"')
    expect(html).toContain('id="context-application-models"')
    expect(html).toContain('Bounded Context · contexts/application.yaml')
  })

  it('renders context documents behind second-level context tabs', () => {
    expect(html).toContain('class="context-tab-link active"')
    expect(html).toContain('data-scl-context-link="overview"')
    expect(html).toContain('data-scl-context-link="context-application"')
    expect(html).toContain('data-scl-context-pane="overview"')
    expect(html).toContain('data-scl-context-pane="context-application"')
    expect(html).toContain('href="#tab=scl&amp;sec=context-application"')
  })

  it('uses context-qualified element anchors to avoid cross-context collisions', () => {
    expect(html).toContain('id="scl-Application/model/Application"')
    expect(html).toContain('href="#scl-Application/model/Application"')
  })

  it('lists context sections in TOC items', () => {
    const items = sclTocItems(bundle)
    expect(items.map((i) => i.id)).toContain('context-application')
    expect(items.map((i) => i.id)).toContain('context-application-models')
  })
})

describe('renderDecisionsTab', () => {
  const docs: DecisionDoc[] = [
    {
      id: 'conception',
      title: 'Conception',
      kind: 'conception',
      filename: 'CONCEPTION.md',
      body: '## Goals\n\nA paragraph.',
    },
    {
      id: 'adr-001-foo',
      title: 'ADR-001: Foo',
      kind: 'adr',
      filename: 'ADR-001-foo.md',
      body: '## Context\n\nWhy.',
      number: 1,
    },
  ]
  const html = renderDecisionsTab(docs)

  it('emits one card per document', () => {
    expect(html).toContain('id="conception"')
    expect(html).toContain('id="adr-001-foo"')
  })

  it('renders ADR number badges and an ADR index', () => {
    expect(html).toContain('ADR-001')
    expect(html).toContain('adr-index-row')
  })

  it('renders markdown bodies through the .md container', () => {
    expect(html).toContain('class="md"')
    expect(html).toContain('Goals')
  })

  it('shows an empty state for zero docs', () => {
    const empty = renderDecisionsTab([])
    expect(empty).toContain('No CONCEPTION or ADR sources')
  })
})

describe('renderChangesTab', () => {
  const changes: ChangeEntry[] = [
    {
      id: 'wi-1-demo',
      work_item: {
        id: 'wi-1-demo',
        title: 'Demo',
        status: 'in_progress',
        risk: 'medium',
        depends_on: ['wi-2-done'],
        motivation: 'because',
        scope: { ui: ['screen A'] },
        out_of_scope: ['unrelated'],
        plan: 'Use existing renderer paths.',
        tasks: '- [ ] T001 [Tooling] Parse tasks.\n- [x] T002 [Verify] Render tasks.',
        affected_guarantees: ['rbac'],
        verification: [{ cmd: 'go test ./...' }],
        risk_notes: 'careful',
        target_state: {
          scl: ['new interface'],
          ui: ['screen B'],
        },
      },
    },
    {
      id: 'wi-2-done',
      work_item: {
        id: 'wi-2-done',
        title: 'Done thing',
        status: 'completed',
        risk: 'low',
        completion: {
          summary: 'finished',
          affected_guarantees_state: ['unchanged'],
          evidence: [
            {
              id: 'go-test',
              kind: 'test',
              result: 'passed',
              command: 'go test ./...',
              artifacts: [
                {
                  path: 'work-items/artifacts/wi-2-done/go-test.log',
                  sha256: '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
                  summary: 'test output',
                },
              ],
            },
          ],
          scl_changes: {
            models: ['Foo'],
          },
        },
      },
    },
  ]
  const html = renderChangesTab(changes)

  it('renders one card per change anchored by id', () => {
    expect(html).toContain('id="wi-1-demo"')
    expect(html).toContain('id="wi-2-done"')
  })

  it('shows status badges and an index row per change', () => {
    expect(html).toContain('badge-status-progress')
    expect(html).toContain('badge-status-done')
    expect(html).toContain('ch-index-row')
  })

  it('renders the completion under the work item when present', () => {
    expect(html).toContain('Completion')
    expect(html).toContain('finished')
  })

  it('renders plan and task progress for active work items', () => {
    expect(html).toContain('Plan')
    expect(html).toContain('Use existing renderer paths')
    expect(html).toContain('Tasks (1/2)')
    expect(html).toContain('1/2 tasks')
  })

  it('renders dependencies as links to their work items', () => {
    expect(html).toContain('Depends on')
    expect(html).toContain('href="#wi-2-done"')
  })

  it('renders completion evidence as a first-class block', () => {
    expect(html).toContain('Evidence')
    expect(html).toContain('go-test')
    expect(html).toContain('work-items/artifacts/wi-2-done/go-test.log')
  })

  it('renders schema-extension fields instead of dropping them', () => {
    expect(html).toContain('Target State')
    expect(html).toContain('new interface')
    expect(html).toContain('Scl Changes')
    expect(html).toContain('Foo')
  })

  it('orders in_progress before completed in the details list', () => {
    const first = html.indexOf('id="wi-1-demo"')
    const second = html.indexOf('id="wi-2-done"')
    expect(first).toBeGreaterThan(-1)
    expect(second).toBeGreaterThan(first)
  })

  it('emits an empty-state placeholder for zero changes', () => {
    expect(renderChangesTab([])).toContain('No work items')
  })
})

describe('renderPage (integration)', () => {
  const site: SiteInput = {
    scl: sampleScl(),
    decisions: [
      {
        id: 'adr-001-foo',
        title: 'Foo',
        kind: 'adr',
        filename: 'ADR-001-foo.md',
        body: 'body',
        number: 1,
      },
    ],
    work_items: [
      {
        id: 'wi-x',
        work_item: { id: 'wi-x', title: 'X', status: 'pending', risk: 'low' },
      },
    ],
    title: 'demo system',
  }
  const html = renderPage(site)

  it('produces a single self-contained HTML document', () => {
    expect(html.startsWith('<!doctype html>')).toBe(true)
    expect(html).toContain('<style>')
    expect(html).toContain('<script>')
    expect(html).toContain('data-diagram-zoom')
  })

  it('embeds available tabs with data-tab markers', () => {
    expect(html).toContain('data-tab="scl"')
    expect(html).toContain('data-tab="decisions"')
    expect(html).toContain('data-tab="work-items"')
    expect(html).not.toContain('data-tab="overview"')
  })

  it('omits Decisions and Work Items tabs when no sources are loaded', () => {
    const out = renderPage({ scl: sampleScl(), decisions: [], work_items: [] })
    expect(out).toContain('data-tab="scl"')
    expect(out).not.toContain('data-tab="overview"')
    expect(out).not.toContain('data-tab="decisions"')
    expect(out).not.toContain('data-tab="work-items"')
    expect(out).not.toContain('data-tab-link="decisions"')
    expect(out).not.toContain('data-tab-link="work-items"')
  })

  it('emits a tab bar with active SCL by default (server side)', () => {
    expect(html).toContain('tab-link active')
    expect(html).toContain('data-tab-link="scl"')
  })

  it('honours --title override in <title> and header', () => {
    expect(html).toContain('<title>demo system</title>')
    expect(html).toContain('>demo system<')
  })

  it('renders without injected XSS when SCL strings contain script tags', () => {
    const evil: SiteInput = {
      ...site,
      scl: { system: '<script>alert(1)</script>', spec_version: '1.0' },
    }
    const out = renderPage(evil)
    expect(out).not.toContain('<script>alert(1)</script>')
    expect(out).toContain('&lt;script&gt;alert(1)&lt;/script&gt;')
  })

  it('is deterministic on identical input', () => {
    expect(renderPage(site)).toBe(renderPage(site))
  })
})

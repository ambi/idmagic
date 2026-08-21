import { describe, expect, it } from 'bun:test'
import { type ReferenceEnvironment, verifyWorkItemReferences } from './work-item-references.ts'

const files: Record<string, string> = {
  'spec/contexts/demo/scenarios.md': [
    '## Standards',
    '',
    'RFC7644-PATCH is adopted.',
    '',
    '## Scenarios',
    '',
    '### REQ-DEMO-001: A valid request succeeds',
    '- ACTOR User',
  ].join('\n'),
  'spec/contexts/demo/main.tsp': 'op StartTask(): void;',
}
const directories = new Set(['backend/demo'])

const environment: ReferenceEnvironment = {
  exists: (path) => path in files || directories.has(path),
  read: (path) => files[path],
}

describe('verifyWorkItemReferences', () => {
  it('accepts scenario, standard, and symbol references that resolve', () => {
    const findings = verifyWorkItemReferences(
      {
        status: 'pending',
        affected_spec: [
          { path: 'spec/contexts/demo/scenarios.md', requirement: 'REQ-DEMO-001' },
          { path: 'spec/contexts/demo/scenarios.md', requirement: 'RFC7644-PATCH' },
          { path: 'spec/contexts/demo/main.tsp', symbol: 'Demo.Operations.StartTask' },
        ],
      },
      environment,
    )
    expect(findings).toEqual([])
  })

  it('rejects a requirement that is only mentioned, not declared as a scenario', () => {
    const findings = verifyWorkItemReferences(
      {
        status: 'pending',
        affected_spec: [{ path: 'spec/contexts/demo/scenarios.md', requirement: 'REQ-DEMO-002' }],
      },
      environment,
    )
    expect(findings).toEqual([
      'requirement does not resolve in spec/contexts/demo/scenarios.md: REQ-DEMO-002',
    ])
  })

  it('ignores the reading list until the item is in progress', () => {
    const record = { initial_context: { source: ['backend/removed'] } }
    expect(verifyWorkItemReferences({ ...record, status: 'pending' }, environment)).toEqual([])
    expect(verifyWorkItemReferences({ ...record, status: 'in_progress' }, environment)).toEqual([
      'initial_context source does not exist: backend/removed',
    ])
  })

  it('resolves a reading-list scenario reference to its declaring document', () => {
    const started = (specification: string[]) =>
      verifyWorkItemReferences(
        { status: 'in_progress', initial_context: { specification } },
        environment,
      )
    expect(started(['spec/contexts/demo/scenarios.md#REQ-DEMO-001'])).toEqual([])
    expect(started(['spec/contexts/demo/scenarios.md#REQ-DEMO-404'])).toEqual([
      'initial_context specification does not resolve: spec/contexts/demo/scenarios.md#REQ-DEMO-404',
    ])
    expect(started(['spec/contexts/gone/scenarios.md#REQ-GONE-001'])).toEqual([
      'initial_context specification path does not exist: spec/contexts/gone/scenarios.md#REQ-GONE-001',
    ])
  })

  it('reports a legacy reference only while the item is active', () => {
    const record = { affected_spec: [{ context: 'Demo', kind: 'model', element: 'User' }] }
    expect(verifyWorkItemReferences({ ...record, status: 'completed' }, environment)).toEqual([])
    expect(verifyWorkItemReferences({ ...record, status: 'in_progress' }, environment)).toEqual([
      'active work item contains a legacy specification reference',
    ])
  })
})

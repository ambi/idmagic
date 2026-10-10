import { describe, expect, it } from 'bun:test'
import {
  type ReferenceEnvironment,
  relocatedSpecPaths,
  verifyWorkItemReferences,
} from './work-item-references.ts'

const files: Record<string, string> = {
  'docs/modules/demo/scenarios.feature.md': [
    '# Feature: Demo',
    '',
    'RFC7644-PATCH is adopted.',
    '',
    '## Rule: REQ-DEMO-001 A valid request succeeds',
    '',
    '### Example: EX-DEMO-001-01 valid request',
    '',
    '- When the user submits a request',
    '- Then the request succeeds',
  ].join('\n'),
  'spec/modules/demo/main.tsp': 'op StartTask(): void;',
}
const directories = new Set(['backend/demo'])

const environment: ReferenceEnvironment = {
  exists: (path) => path in files || directories.has(path),
  read: (path) => files[path],
}

describe('verifyWorkItemReferences with the feature layout', () => {
  const moved: ReferenceEnvironment = {
    exists: (path) => path === 'docs/modules/demo/work/task/README.md',
    read: (path) =>
      path === 'docs/modules/demo/work/task/README.md'
        ? '# Task\n\n## 操作\n\n#### REQ-DEMO-001 A valid request succeeds\n'
        : undefined,
    relocated: (path) =>
      path === 'docs/modules/demo/scenarios.feature.md'
        ? ['docs/modules/demo/work/task/README.md']
        : undefined,
  }

  it('resolves a rule declared by a feature specification heading', () => {
    const findings = verifyWorkItemReferences(
      {
        status: 'pending',
        affected_spec: [
          { path: 'docs/modules/demo/work/task/README.md', requirement: 'REQ-DEMO-001' },
        ],
      },
      moved,
    )
    expect(findings).toEqual([])
  })

  it('resolves a completed record through the relocation table', () => {
    const findings = verifyWorkItemReferences(
      {
        status: 'completed',
        affected_spec: [
          { path: 'docs/modules/demo/scenarios.feature.md', requirement: 'REQ-DEMO-001' },
          { path: 'docs/modules/demo/scenarios.feature.md', requirement: 'REQ-DEMO-002' },
        ],
      },
      moved,
    )
    expect(findings).toEqual([
      'requirement does not resolve where docs/modules/demo/scenarios.feature.md moved: REQ-DEMO-002',
    ])
  })

  it('keeps an active record on the current path', () => {
    const findings = verifyWorkItemReferences(
      {
        status: 'pending',
        affected_spec: [
          { path: 'docs/modules/demo/scenarios.feature.md', requirement: 'REQ-DEMO-001' },
        ],
      },
      moved,
    )
    expect(findings).toEqual([
      'affected_spec path does not exist: docs/modules/demo/scenarios.feature.md',
    ])
  })
})

describe('verifyWorkItemReferences', () => {
  it('accepts scenario, standard, and symbol references that resolve', () => {
    const findings = verifyWorkItemReferences(
      {
        status: 'pending',
        affected_spec: [
          { path: 'docs/modules/demo/scenarios.feature.md', requirement: 'REQ-DEMO-001' },
          { path: 'docs/modules/demo/scenarios.feature.md', requirement: 'RFC7644-PATCH' },
          { path: 'spec/modules/demo/main.tsp', symbol: 'Demo.Operations.StartTask' },
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
        affected_spec: [
          { path: 'docs/modules/demo/scenarios.feature.md', requirement: 'REQ-DEMO-002' },
        ],
      },
      environment,
    )
    expect(findings).toEqual([
      'requirement does not resolve in docs/modules/demo/scenarios.feature.md: REQ-DEMO-002',
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
    expect(started(['docs/modules/demo/scenarios.feature.md#REQ-DEMO-001'])).toEqual([])
    expect(started(['docs/modules/demo/scenarios.feature.md#REQ-DEMO-404'])).toEqual([
      'initial_context specification does not resolve: docs/modules/demo/scenarios.feature.md#REQ-DEMO-404',
    ])
    expect(started(['docs/modules/gone/scenarios.feature.md#REQ-GONE-001'])).toEqual([
      'initial_context specification path does not exist: docs/modules/gone/scenarios.feature.md#REQ-GONE-001',
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

describe('relocatedSpecPaths', () => {
  const table = {
    'docs/domain/demo/scenarios.feature.md': ['docs/modules/demo/work/task/README.md'],
    'docs/domain/glossary.md': ['docs/requirements/glossary.md'],
    'docs/domain/': ['docs/modules/'],
  }

  it('prefers the entry that names the moved file', () => {
    expect(relocatedSpecPaths(table, 'docs/domain/demo/scenarios.feature.md')).toEqual([
      'docs/modules/demo/work/task/README.md',
    ])
    expect(relocatedSpecPaths(table, 'docs/domain/glossary.md')).toEqual([
      'docs/requirements/glossary.md',
    ])
  })

  it('moves a path under a renamed directory by its prefix', () => {
    expect(relocatedSpecPaths(table, 'docs/domain/demo/work/task/README.md')).toEqual([
      'docs/modules/demo/work/task/README.md',
    ])
  })

  it('leaves a path no entry covers unresolved', () => {
    expect(relocatedSpecPaths(table, 'docs/other/README.md')).toBeUndefined()
  })
})

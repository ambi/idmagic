import { describe, expect, it } from 'bun:test'
import { type Snapshot, diffSpecifications, formatSpecificationDiff } from './spec-diff.ts'

const document = (scenarios: string, transitions = ''): string =>
  [
    '# Demo Specification',
    '',
    '## Overview',
    '',
    'Demo behavior.',
    '',
    '## State Transitions',
    '',
    transitions,
    '',
    '## Scenarios',
    '',
    scenarios,
  ].join('\n')

const machine = (effect: string): string =>
  [
    '### Lifecycle',
    '',
    '| From | Event | Guard | To | Effects |',
    '|---|---|---|---|---|',
    `| Ready | Run | — | Done | ${effect} |`,
  ].join('\n')

const scenario = (id: string, result: string): string =>
  [
    `### ${id}: A request`,
    '- ACTOR User',
    '- WHEN the request is submitted',
    `- THEN ${result}`,
  ].join('\n')

const snapshot = (documentSource: string, tsp = 'op StartTask(): void;'): Snapshot =>
  new Map([
    ['spec/contexts/demo/SPECIFICATION.md', documentSource],
    ['spec/contexts/demo/main.tsp', tsp],
  ])

describe('diffSpecifications', () => {
  it('reports nothing when the normative content is unchanged', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Done')))
    const diff = diffSpecifications(
      base,
      snapshot(document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Done'))),
    )
    expect(diff).toEqual({
      addedScenarios: [],
      removedScenarios: [],
      changedScenarios: [],
      changedTransitions: [],
      addedDeclarations: [],
      removedDeclarations: [],
    })
    expect(formatSpecificationDiff(diff, 'main')).toBe(
      'no normative specification change against main',
    )
  })

  it('separates added, removed, and changed scenarios', () => {
    const base = snapshot(
      document(
        [scenario('REQ-DEMO-001', 'it succeeds'), scenario('REQ-DEMO-002', 'it stops')].join(
          '\n\n',
        ),
      ),
    )
    const head = snapshot(
      document(
        [scenario('REQ-DEMO-001', 'it is accepted'), scenario('REQ-DEMO-003', 'it retries')].join(
          '\n\n',
        ),
      ),
    )
    const diff = diffSpecifications(base, head)
    expect(diff.addedScenarios).toEqual(['REQ-DEMO-003'])
    expect(diff.removedScenarios).toEqual(['REQ-DEMO-002'])
    expect(diff.changedScenarios).toEqual(['REQ-DEMO-001'])
  })

  it('detects a changed transition row and ignores reformatting', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Done')))
    const reformatted = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds'), `${machine('emit Done')}   `),
    )
    expect(diffSpecifications(base, reformatted).changedTransitions).toEqual([])

    const changed = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds'), machine('emit Completed')),
    )
    expect(diffSpecifications(base, changed).changedTransitions).toEqual([
      'spec/contexts/demo/SPECIFICATION.md#Lifecycle',
    ])
  })

  it('tracks TypeSpec declarations coming and going', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')), 'model Task {}')
    const head = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      'op StartTask(): void;',
    )
    const diff = diffSpecifications(base, head)
    expect(diff.addedDeclarations).toEqual(['spec/contexts/demo/main.tsp:StartTask'])
    expect(diff.removedDeclarations).toEqual(['spec/contexts/demo/main.tsp:Task'])
  })

  it('leaves per-operation transport wrappers out of the declaration list', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')), '')
    const head = snapshot(
      document(scenario('REQ-DEMO-001', 'it succeeds')),
      [
        'op StartTask(): void;',
        'model StartTaskError403Body {}',
        'model StartTaskHttpRequest {}',
        'model StartTaskSuccess_200 {}',
      ].join('\n'),
    )
    expect(diffSpecifications(base, head).addedDeclarations).toEqual([
      'spec/contexts/demo/main.tsp:StartTask',
    ])
  })

  it('formats only the groups that have entries', () => {
    const base = snapshot(document(scenario('REQ-DEMO-001', 'it succeeds')))
    const head = snapshot(
      document(
        [scenario('REQ-DEMO-001', 'it succeeds'), scenario('REQ-DEMO-002', 'it retries')].join(
          '\n\n',
        ),
      ),
    )
    const text = formatSpecificationDiff(diffSpecifications(base, head), 'HEAD')
    expect(text).toContain('added scenarios:\n  REQ-DEMO-002')
    expect(text).not.toContain('removed scenarios')
  })
})

import { describe, expect, it } from 'bun:test'
import {
  extractDeclaration,
  initialContextDraft,
  partitionSources,
  sourceDirectories,
} from './brief.ts'

const SCENARIOS = [
  '# Feature: Demo Scenarios',
  '',
  '## Rule: REQ-DEMO-001 The task starts',
  '',
  'Primary actor: `Operator`',
  '',
  '### Example: EX-DEMO-001-01 the ordinary route',
  '',
  '- When the operator submits the form',
  '- Then the task runs',
  '',
  '## Rule: REQ-DEMO-002 The task stops',
  '',
  '- Then the task stops',
  '',
].join('\n')

const STANDARDS = [
  '# Demo Standards',
  '',
  '## The Demo Protocol',
  '',
  'RFC 9999 — https://example.test/rfc9999',
  '',
  '| ID | Adoption | Strength | Statement |',
  '|---|---|---|---|',
  '| RFC9999-START | required | MUST | Starting a task is answered with its identifier. |',
  '| RFC9999-STOP | optional | MAY | Stopping a task is offered. |',
  '',
].join('\n')

describe('extractDeclaration', () => {
  /**
   * The brief exists so that reading a requirement does not mean opening a
   * 2,000-line context. It therefore returns the block that declares the
   * identifier, and nothing that follows it.
   */
  it('returns the rule block and stops at the next rule', () => {
    const declaration = extractDeclaration(SCENARIOS, 'REQ-DEMO-001')
    expect(declaration).toContain('## Rule: REQ-DEMO-001 The task starts')
    expect(declaration).toContain('### Example: EX-DEMO-001-01 the ordinary route')
    expect(declaration).not.toContain('REQ-DEMO-002')
  })

  it('returns the last rule of a document', () => {
    expect(extractDeclaration(SCENARIOS, 'REQ-DEMO-002')).toContain('- Then the task stops')
  })

  /**
   * A standards requirement is a table row, not a heading. Returning the row
   * without its header would hand back four unlabelled cells.
   */
  it('returns a standards row with the heading and column names that give it meaning', () => {
    const declaration = extractDeclaration(STANDARDS, 'RFC9999-START') ?? ''
    expect(declaration).toContain('## The Demo Protocol')
    expect(declaration).toContain('| ID | Adoption | Strength | Statement |')
    expect(declaration).toContain('| RFC9999-START | required | MUST |')
    expect(declaration).not.toContain('RFC9999-STOP')
  })

  it('reports nothing for an identifier the document does not declare', () => {
    expect(extractDeclaration(SCENARIOS, 'REQ-DEMO-404')).toBeUndefined()
  })

  /** A mention inside another rule's steps is not a declaration. */
  it('does not mistake a mention for a declaration', () => {
    const source = [
      '## Rule: REQ-DEMO-003 Something',
      '',
      '- Then REQ-DEMO-004 also holds',
      '',
    ].join('\n')
    expect(extractDeclaration(source, 'REQ-DEMO-004')).toBeUndefined()
  })
})

describe('partitionSources', () => {
  it('separates tests from implementation', () => {
    expect(
      partitionSources([
        'backend/system/start.go',
        'backend/system/start_test.go',
        'frontend/tests/e2e/start.spec.ts',
        'frontend/src/start.tsx',
      ]),
    ).toEqual({
      implementation: ['backend/system/start.go', 'frontend/src/start.tsx'],
      tests: ['backend/system/start_test.go', 'frontend/tests/e2e/start.spec.ts'],
    })
  })
})

describe('sourceDirectories', () => {
  it('collapses naming files to the packages that hold them', () => {
    expect(
      sourceDirectories([
        'backend/system/usecases/start_test.go',
        'backend/system/usecases/stop_test.go',
        'backend/system/domain/task.go',
      ]),
    ).toEqual(['backend/system/domain', 'backend/system/usecases'])
  })

  it('keeps a path with no directory as it is', () => {
    expect(sourceDirectories(['CONFIGURATION.md'])).toEqual(['CONFIGURATION.md'])
  })
})

describe('initialContextDraft', () => {
  const draft = initialContextDraft({
    specification: ['docs/contexts/demo/scenarios.feature.md#REQ-DEMO-001'],
    typespec: ['Product.Demo.Operations.StartTask'],
    implementation: ['backend/demo/usecases/start.go', 'backend/demo/domain'],
    tests: ['backend/demo/usecases/start_test.go'],
  })

  /** The point of the draft is that it can be pasted, not retyped. */
  it('emits the frontmatter block a work item can take as written', () => {
    expect(draft).toBe(
      [
        'initial_context:',
        '  specification: [docs/contexts/demo/scenarios.feature.md#REQ-DEMO-001]',
        '  typespec: [Product.Demo.Operations.StartTask]',
        '  source:',
        '    - backend/demo/usecases/start.go',
        '    - backend/demo/domain',
        '  tests: [backend/demo/usecases/start_test.go]',
        '  stop_before_reading: []',
        '',
      ].join('\n'),
    )
  })

  it('emits an empty list rather than an absent key', () => {
    const empty = initialContextDraft({
      specification: [],
      typespec: [],
      implementation: [],
      tests: [],
    })
    expect(empty).toContain('  source: []')
    expect(empty).toContain('  tests: []')
  })
})

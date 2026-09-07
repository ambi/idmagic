import { describe, expect, it } from 'bun:test'
import { documentKind, validateDocument } from './specification-doc.ts'

const SCENARIOS = 'docs/contexts/demo/scenarios.feature.md'
const STATES = 'docs/contexts/demo/states.md'
const STANDARDS = 'docs/contexts/demo/standards.md'

const scenarios = `# Feature: Demo

## Rule: REQ-DEMO-001 A valid request succeeds

### Example: EX-DEMO-001-01 A valid request

- Given a valid request exists
- When the user submits the request
- Then the request succeeds
`

const messages = (path: string, source: string) =>
  validateDocument(path, source).findings.map((finding) => finding.message)

describe('documentKind', () => {
  it('names the grammar of each canonical document', () => {
    expect(documentKind('docs/contexts/demo/states.md')).toBe('states')
    expect(documentKind('docs/contexts/demo/scenarios.feature.md')).toBe('scenarios')
    expect(documentKind('docs/contexts/demo/decisions.md')).toBe('prose')
    expect(documentKind('docs/standards.md')).toBe('standards')
    expect(documentKind('docs/design/security/authorization.md')).toBe('prose')
    expect(documentKind('docs/design/security/threat-model.md')).toBe('prose')
    expect(documentKind('docs/design/application/design-rules.md')).toBe('prose')
  })

  it('names the grammar of the top-down system document tree', () => {
    expect(documentKind('docs/requirements/quality.md')).toBe('prose')
    expect(documentKind('docs/architecture/deployment.md')).toBe('prose')
    expect(documentKind('docs/design/security/threat-model.md')).toBe('prose')
    expect(documentKind('docs/design/observability/logging.md')).toBe('prose')
    expect(documentKind('docs/verification/system-acceptance.md')).toBe('prose')
    expect(documentKind('docs/operations/service-management.md')).toBe('prose')
  })

  it('rejects a name the layout does not define, and a context-only name at the root', () => {
    expect(documentKind('docs/contexts/demo/notes.md')).toBeUndefined()
    expect(documentKind('docs/states.md')).toBeUndefined()
    expect(documentKind('docs/authorization.md')).toBeUndefined()
    expect(documentKind('docs/contexts/demo/user/scenarios.feature.md')).toBeUndefined()
    expect(documentKind('frontend/README.md')).toBeUndefined()
    expect(documentKind('docs/design/security/network.md')).toBeUndefined()
  })

  it('no longer recognizes the single canonical document', () => {
    expect(documentKind('docs/SPECIFICATION.md')).toBeUndefined()
    expect(documentKind('docs/contexts/demo/SPECIFICATION.md')).toBeUndefined()
  })

  it('rejects a path the canonical layout does not define', () => {
    expect(messages('docs/contexts/demo/notes.md', '# Notes\n')).toEqual([
      'not a canonical specification document',
    ])
  })
})

describe('scenarios.feature.md', () => {
  it('accepts a Gherkin example and reports its rule and example ids', () => {
    const result = validateDocument(SCENARIOS, scenarios)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.map((scenario) => scenario.id)).toEqual(['REQ-DEMO-001'])
    expect(result.exampleIds).toEqual([{ id: 'EX-DEMO-001-01', line: 5, parentId: 'REQ-DEMO-001' }])
  })

  it('accepts a retired rule without examples and reports its successor', () => {
    const source = `${scenarios}
## Rule: REQ-DEMO-002 An old behavior (superseded by REQ-DEMO-001)

Replaced by the valid request scenario.
`
    const result = validateDocument(SCENARIOS, source)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.at(-1)).toMatchObject({
      id: 'REQ-DEMO-002',
      supersededBy: 'REQ-DEMO-001',
    })
  })

  it('still requires examples in a rule that is not retired', () => {
    const source = `${scenarios}
## Rule: REQ-DEMO-002 An old behavior

Replaced by the valid request scenario.
`
    expect(messages(SCENARIOS, source)).toContain('REQ-DEMO-002 must contain at least one example')
  })

  it('rejects an example without a trigger', () => {
    const source = scenarios.replace(
      '- When the user submits the request',
      '- Given the user submits the request',
    )
    expect(messages(SCENARIOS, source)).toContain('example must contain at least one When step')
  })

  it('accepts multiple triggers in a multi-operation flow', () => {
    const source = scenarios.replace(
      '- Then the request succeeds',
      '- Then the request succeeds\n- When the user retrieves the result\n- Then the result is returned',
    )
    expect(validateDocument(SCENARIOS, source).findings).toEqual([])
  })

  it('rejects links from a canonical document to decisions/', () => {
    const source = scenarios.replace(
      '# Feature: Demo',
      '# Feature: Demo\n\nSee [old choice](../decisions/old-choice.md).',
    )
    expect(messages(SCENARIOS, source)).toContain(
      'current specification must be self-contained and must not link to decisions/',
    )
  })

  it('requires exactly one H1', () => {
    expect(messages(SCENARIOS, scenarios.replace('# Feature: Demo\n', ''))).toContain(
      'document must contain exactly one H1',
    )
  })

  it('keeps normative rules out of the other documents', () => {
    const source = `# Demo Decisions

## Rule: REQ-DEMO-002 A behavior
`
    const result = validateDocument('docs/contexts/demo/decisions.md', source)
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'REQ-DEMO-002 must be declared in scenarios.feature.md',
    ])
    expect(result.scenarioIds).toEqual([])
  })
})

const states = `# Demo State Transitions

## Lifecycle

| State | Kind | Meaning |
|---|---|---|
| Ready | initial | 受理直後 |
| Done | terminal | 完了 |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Ready | Run | — | Done | Completed |
`

describe('states.md', () => {
  it('accepts a machine that declares its states before its transitions', () => {
    expect(validateDocument(STATES, states).findings).toEqual([])
  })

  it('requires the state table', () => {
    const source = states.replace(
      '| State | Kind | Meaning |\n|---|---|---|\n| Ready | initial | 受理直後 |\n| Done | terminal | 完了 |\n\n',
      '',
    )
    expect(messages(STATES, source)).toEqual([
      'state machine must declare its states with | State | Kind | Meaning |',
    ])
  })

  it('requires the transition table', () => {
    const source = states.slice(0, states.indexOf('| From | Event |'))
    expect(messages(STATES, source)).toContain(
      'state transition must use From | Event | Guard | To | Effects',
    )
  })

  it('rejects a transition into a state the state table does not declare', () => {
    const source = states.replace('| Ready | Run | — | Done |', '| Ready | Run | — | Gone |')
    expect(messages(STATES, source)).toEqual([
      'transition names Gone, which the state table does not declare',
    ])
  })

  it('rejects a Kind outside the vocabulary and a machine without one initial state', () => {
    const source = states.replace('| Ready | initial |', '| Ready | 初期 |')
    expect(messages(STATES, source)).toEqual([
      'state Ready has Kind "初期"; use one of initial, terminal, —',
      'state machine must declare exactly one initial state, found 0',
    ])
  })

  it('rejects an empty-string guard', () => {
    const source = states.replace('| Ready | Run | — |', '| Ready | Run | "" |')
    expect(messages(STATES, source)).toContain(
      'unconditional state transition guard must use — instead of an empty string',
    )
  })

  it('keeps an escaped pipe inside a guard rather than ending the cell', () => {
    const source = states.replace(
      '| Ready | Run | — | Done | Completed |',
      '| Ready | Run | input.purge \\|\\| expired | Done | Completed |',
    )
    expect(validateDocument(STATES, source).findings).toEqual([])
  })
})

const withStandards = (rows: string) => `# Demo Standards

## Demo Protocol

https://example.invalid/demo

| ID | Adoption | Strength | Statement |
|---|---|---|---|
${rows}
`

describe('standards.md', () => {
  it('accepts closed vocabularies on both axes, including optional with MUST', () => {
    const source = withStandards(
      [
        '| DEMO-CORE | required | MUST | The product answers a demo request. |',
        '| DEMO-EXTRA | optional | MUST | When the extension is offered, its rules are honored. |',
        '| DEMO-LEGACY | excluded | MAY | The legacy transport is not offered. |',
      ].join('\n'),
    )
    expect(validateDocument(STANDARDS, source).findings).toEqual([])
  })

  it('rejects a standard without the canonical table', () => {
    const source = '# Demo Standards\n\n## Demo Protocol\n\nAdopted in full.\n'
    expect(messages(STANDARDS, source)).toContain(
      'standard must use | ID | Adoption | Strength | Statement |',
    )
  })

  it('rejects an adoption outside the vocabulary', () => {
    const source = withStandards('| DEMO-CORE | planned | MUST | Someday. |')
    expect(messages(STANDARDS, source)).toContain(
      'DEMO-CORE has Adoption "planned"; use one of required, optional, partial, excluded',
    )
  })

  it('rejects a strength outside the vocabulary', () => {
    const source = withStandards('| DEMO-CORE | required | SHALL | The product answers. |')
    expect(messages(STANDARDS, source)).toContain(
      'DEMO-CORE has Strength "SHALL"; use one of MUST, MUST NOT, SHOULD, MAY',
    )
  })

  it('rejects an obligation on an excluded capability', () => {
    const source = withStandards(
      '| DEMO-LEGACY | excluded | MUST | The legacy transport is required. |',
    )
    expect(messages(STANDARDS, source)).toContain(
      'DEMO-LEGACY is excluded, so it cannot carry the obligation "MUST"',
    )
  })

  it('rejects a duplicate standard id across two standards', () => {
    const source = `${withStandards('| DEMO-CORE | required | MUST | The product answers. |')}
## Other Protocol

https://example.invalid/other

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| DEMO-CORE | optional | MAY | A second row claiming the same id. |
`
    expect(messages(STANDARDS, source)).toContain(
      'duplicate standard id DEMO-CORE (first seen on line 9)',
    )
  })

  it('collects the id of every row so the coverage check can ask for a test', () => {
    const source = withStandards(
      [
        '| DEMO-CORE | required | MUST | The product answers a demo request. |',
        '| DEMO-LEGACY | excluded | MAY | The legacy transport is not offered. |',
      ].join('\n'),
    )
    expect(validateDocument(STANDARDS, source).standardIds).toEqual([
      { id: 'DEMO-CORE', line: 9 },
      { id: 'DEMO-LEGACY', line: 10 },
    ])
  })

  it('collects no standard id from a document of another kind', () => {
    expect(validateDocument(SCENARIOS, scenarios).standardIds).toEqual([])
  })
})

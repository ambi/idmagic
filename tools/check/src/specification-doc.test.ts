import { describe, expect, it } from 'bun:test'
import { documentKind, validateDocument } from './specification-doc.ts'

const SCENARIOS = 'docs/contexts/demo/scenarios.md'
const STATES = 'docs/contexts/demo/states.md'
const STANDARDS = 'docs/contexts/demo/standards.md'

const scenarios = `# Demo Scenarios

### REQ-DEMO-001: A valid request succeeds
- ACTOR User
- GIVEN a valid request
- WHEN the request is submitted
- THEN the request succeeds
  - ALT the request is invalid → the request is rejected
`

const messages = (path: string, source: string) =>
  validateDocument(path, source).findings.map((finding) => finding.message)

describe('documentKind', () => {
  it('names the grammar of each canonical document', () => {
    expect(documentKind('docs/contexts/demo/states.md')).toBe('states')
    expect(documentKind('docs/contexts/demo/scenarios.md')).toBe('scenarios')
    expect(documentKind('docs/contexts/demo/decisions.md')).toBe('prose')
    expect(documentKind('docs/standards.md')).toBe('standards')
    expect(documentKind('docs/authorization.md')).toBe('prose')
    expect(documentKind('docs/threat-model.md')).toBe('prose')
    expect(documentKind('docs/design-rules.md')).toBe('prose')
  })

  it('rejects a name the layout does not define, and a context-only name at the root', () => {
    expect(documentKind('docs/contexts/demo/notes.md')).toBeUndefined()
    expect(documentKind('docs/states.md')).toBeUndefined()
    expect(documentKind('docs/contexts/demo/user/scenarios.md')).toBeUndefined()
    expect(documentKind('frontend/README.md')).toBeUndefined()
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

describe('scenarios.md', () => {
  it('accepts a scenario with a nested alternative and reports its id', () => {
    const result = validateDocument(SCENARIOS, scenarios)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.map((scenario) => scenario.id)).toEqual(['REQ-DEMO-001'])
  })

  it('accepts a retired scenario without steps and reports its successor', () => {
    const source = `${scenarios}
### REQ-DEMO-002: An old behavior (superseded by REQ-DEMO-001)
Replaced by the valid request scenario.
`
    const result = validateDocument(SCENARIOS, source)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.at(-1)).toMatchObject({
      id: 'REQ-DEMO-002',
      supersededBy: 'REQ-DEMO-001',
    })
  })

  it('still requires steps in a scenario that is not retired', () => {
    const source = `${scenarios}
### REQ-DEMO-002: An old behavior
Replaced by the valid request scenario.
`
    expect(messages(SCENARIOS, source)).toContain('scenario must contain at least one WHEN line')
  })

  it('rejects a top-level alternative and old scenario syntax', () => {
    const source = scenarios.replace('- ACTOR User', '- Actor: User').replace('  - ALT', '- ALT')
    expect(messages(SCENARIOS, source)).toContain(
      'scenario keyword Actor must be uppercase and must not use a colon',
    )
    expect(messages(SCENARIOS, source)).toContain(
      'ALT must be a two-space-indented child of a WHEN or THEN step',
    )
  })

  it('rejects a scenario without a trigger', () => {
    const source = scenarios.replace(
      '- WHEN the request is submitted',
      '- GIVEN the request is submitted',
    )
    expect(messages(SCENARIOS, source)).toContain('scenario must contain at least one WHEN line')
  })

  it('rejects an alternative nested below GIVEN', () => {
    const source = scenarios.replace(
      '- GIVEN a valid request',
      '- GIVEN a valid request\n  - ALT it is invalid → reject it',
    )
    expect(messages(SCENARIOS, source)).toContain(
      'ALT must be nested immediately below a WHEN or THEN step',
    )
  })

  it('rejects numbered behavior steps', () => {
    const source = scenarios.replace(
      '- WHEN the request is submitted',
      '- WHEN (1) the request is submitted',
    )
    expect(messages(SCENARIOS, source)).toContain('WHEN and THEN must not use local step numbers')
  })

  it('accepts multiple triggers in a multi-operation flow', () => {
    const source = scenarios.replace(
      '- THEN the request succeeds',
      '- THEN the request succeeds\n- WHEN the result is retrieved\n- THEN the result is returned',
    )
    expect(validateDocument(SCENARIOS, source).findings).toEqual([])
  })

  it('rejects GIVEN after behavior starts', () => {
    const source = scenarios.replace(
      '- THEN the request succeeds',
      '- THEN the request succeeds\n- GIVEN a late precondition',
    )
    expect(messages(SCENARIOS, source)).toContain('GIVEN must appear before WHEN and THEN clauses')
  })

  it('rejects links from a canonical document to decisions/', () => {
    const source = scenarios.replace(
      '# Demo Scenarios',
      '# Demo Scenarios\n\nSee [old choice](../decisions/old-choice.md).',
    )
    expect(messages(SCENARIOS, source)).toContain(
      'current specification must be self-contained and must not link to decisions/',
    )
  })

  it('requires exactly one H1', () => {
    expect(messages(SCENARIOS, scenarios.replace('# Demo Scenarios\n', ''))).toContain(
      'document must contain exactly one H1',
    )
  })

  it('keeps normative scenarios out of the other documents', () => {
    const source = `# Demo Decisions

### REQ-DEMO-002: A behavior
- ACTOR User
- WHEN it happens
- THEN it holds
`
    const result = validateDocument('docs/contexts/demo/decisions.md', source)
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'REQ-DEMO-002 must be declared in scenarios.md',
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
})

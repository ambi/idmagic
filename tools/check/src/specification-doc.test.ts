import { describe, expect, it } from 'bun:test'
import { documentKind, validateDocument, validateSpecification } from './specification-doc.ts'

const valid = `---
context: demo
updated_at: 2026-08-11
---

# Demo Specification

## Overview

Demo behavior.

## Glossary

| Term | Definition |
|---|---|
| Demo | A sample. |

## State Transitions

### Lifecycle

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Ready | Run | allowed | Done | emit Completed |

## Scenarios

### REQ-DEMO-001: A valid request succeeds
- ACTOR User
- GIVEN a valid request
- WHEN the request is submitted
- THEN the request succeeds
  - ALT the request is invalid → the request is rejected
`

describe('validateSpecification', () => {
  it('accepts canonical sections and nested alternatives', () => {
    const result = validateSpecification(valid)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.map((scenario) => scenario.id)).toEqual(['REQ-DEMO-001'])
  })

  it('accepts a retired scenario without steps and reports its successor', () => {
    const source = `${valid}
### REQ-DEMO-002: An old behavior (superseded by REQ-DEMO-001)
Replaced by the valid request scenario.
`
    const result = validateSpecification(source)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.at(-1)).toMatchObject({
      id: 'REQ-DEMO-002',
      supersededBy: 'REQ-DEMO-001',
    })
  })

  it('still requires steps in a scenario that is not retired', () => {
    const source = `${valid}
### REQ-DEMO-002: An old behavior
Replaced by the valid request scenario.
`
    const result = validateSpecification(source)
    expect(result.findings.map((finding) => finding.message)).toContain(
      'scenario must contain at least one WHEN line',
    )
  })

  it('rejects a top-level alternative and old scenario syntax', () => {
    const result = validateSpecification(
      valid.replace('- ACTOR User', '- Actor: User').replace('  - ALT', '- ALT'),
    )
    expect(result.findings.map((finding) => finding.message)).toContain(
      'scenario keyword Actor must be uppercase and must not use a colon',
    )
    expect(result.findings.map((finding) => finding.message)).toContain(
      'ALT must be a two-space-indented child of a WHEN or THEN step',
    )
  })

  it('rejects a scenario without a trigger', () => {
    const result = validateSpecification(
      valid.replace('- WHEN the request is submitted', '- GIVEN the request is submitted'),
    )
    expect(result.findings.map((finding) => finding.message)).toContain(
      'scenario must contain at least one WHEN line',
    )
  })

  it('rejects an alternative nested below GIVEN', () => {
    const source = valid.replace(
      '- GIVEN a valid request',
      '- GIVEN a valid request\n  - ALT it is invalid → reject it',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'ALT must be nested immediately below a WHEN or THEN step',
    )
  })

  it('rejects numbered behavior steps', () => {
    const source = valid.replace(
      '- WHEN the request is submitted',
      '- WHEN (1) the request is submitted',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'WHEN and THEN must not use local step numbers',
    )
  })

  it('accepts multiple triggers in a multi-operation flow', () => {
    const source = valid.replace(
      '- THEN the request succeeds',
      '- THEN the request succeeds\n- WHEN the result is retrieved\n- THEN the result is returned',
    )
    expect(validateSpecification(source).findings).toEqual([])
  })

  it('rejects GIVEN after behavior starts', () => {
    const source = valid.replace(
      '- THEN the request succeeds',
      '- THEN the request succeeds\n- GIVEN a late precondition',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'GIVEN must appear before WHEN and THEN clauses',
    )
  })

  it('rejects scenarios placed before standards', () => {
    const source = valid.replace(
      '## Scenarios',
      '## Scenarios\n\n### REQ-DEMO-003: First\n- ACTOR User\n- WHEN something happens\n- THEN done\n\n## Standards',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'Standards is out of canonical section order',
    )
  })

  it('rejects links from current specifications to decisions/', () => {
    const source = valid.replace(
      'Demo behavior.',
      'Demo behavior. See [old choice](../decisions/old-choice.md).',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'current specification must be self-contained and must not link to decisions/',
    )
  })

  it('rejects an empty-string state transition guard', () => {
    const source = valid.replace('| Ready | Run | allowed | Done |', '| Ready | Run | "" | Done |')
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'unconditional state transition guard must use — instead of an empty string',
    )
  })

  it('rejects the retired Authorization Boundary section', () => {
    const source = valid.replace(
      '## State Transitions',
      '## Authorization Boundary\n\nOnly an administrator.\n\n## State Transitions',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'unknown top-level section Authorization Boundary',
    )
  })
})

const withStandards = (rows: string) =>
  valid.replace(
    '## State Transitions',
    `## Standards

### Demo Protocol

https://example.invalid/demo

| ID | Adoption | Strength | Statement |
|---|---|---|---|
${rows}

## State Transitions`,
  )

describe('validateSpecification standards', () => {
  it('accepts closed vocabularies on both axes, including optional with MUST', () => {
    const source = withStandards(
      [
        '| DEMO-CORE | required | MUST | The product answers a demo request. |',
        '| DEMO-EXTRA | optional | MUST | When the extension is offered, its rules are honored. |',
        '| DEMO-LEGACY | excluded | MAY | The legacy transport is not offered. |',
      ].join('\n'),
    )
    expect(validateSpecification(source).findings).toEqual([])
  })

  it('rejects a standard without the canonical table', () => {
    const source = valid.replace(
      '## State Transitions',
      '## Standards\n\n### Demo Protocol\n\nAdopted in full.\n\n## State Transitions',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'standard must use | ID | Adoption | Strength | Statement |',
    )
  })

  it('rejects an adoption outside the vocabulary', () => {
    const source = withStandards('| DEMO-CORE | planned | MUST | Someday. |')
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'DEMO-CORE has Adoption "planned"; use one of required, optional, partial, excluded',
    )
  })

  it('rejects a strength outside the vocabulary', () => {
    const source = withStandards('| DEMO-CORE | required | SHALL | The product answers. |')
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'DEMO-CORE has Strength "SHALL"; use one of MUST, MUST NOT, SHOULD, MAY',
    )
  })

  it('rejects an obligation on an excluded capability', () => {
    const source = withStandards(
      '| DEMO-LEGACY | excluded | MUST | The legacy transport is required. |',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'DEMO-LEGACY is excluded, so it cannot carry the obligation "MUST"',
    )
  })

  it('rejects a duplicate standard id across two standards', () => {
    const source = withStandards('| DEMO-CORE | required | MUST | The product answers. |').replace(
      '## State Transitions',
      `### Other Protocol

https://example.invalid/other

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| DEMO-CORE | optional | MAY | A second row claiming the same id. |

## State Transitions`,
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'duplicate standard id DEMO-CORE (first seen on line 26)',
    )
  })
})

describe('documentKind', () => {
  it('names the grammar of each split-layout document', () => {
    expect(documentKind('spec/contexts/demo/states.md')).toBe('states')
    expect(documentKind('spec/contexts/demo/scenarios.md')).toBe('scenarios')
    expect(documentKind('spec/contexts/demo/decisions.md')).toBe('prose')
    expect(documentKind('spec/standards.md')).toBe('standards')
    expect(documentKind('spec/authorization.md')).toBe('prose')
    expect(documentKind('spec/SPECIFICATION.md')).toBe('legacy')
  })

  it('rejects a name the layout does not define, and a context-only name at the root', () => {
    expect(documentKind('spec/contexts/demo/notes.md')).toBeUndefined()
    expect(documentKind('spec/states.md')).toBeUndefined()
    expect(documentKind('spec/contexts/demo/user/scenarios.md')).toBeUndefined()
    expect(documentKind('frontend/README.md')).toBeUndefined()
  })
})

const states = `# Demo の状態遷移

## Lifecycle

| State | Kind | Meaning |
|---|---|---|
| Ready | initial | 受理直後 |
| Done | terminal | 完了 |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Ready | Run | — | Done | Completed |
`

describe('validateDocument', () => {
  it('accepts a states.md that declares its states before its transitions', () => {
    const result = validateDocument('spec/contexts/demo/states.md', states)
    expect(result.findings).toEqual([])
  })

  it('requires the state table', () => {
    const source = states.replace(
      /\| State \| Kind \| Meaning \|\n\|---\|---\|---\|\n\| Ready \| initial \| 受理直後 \|\n\| Done \| terminal \| 完了 \|\n\n/,
      '',
    )
    const result = validateDocument('spec/contexts/demo/states.md', source)
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'state machine must declare its states with | State | Kind | Meaning |',
    ])
  })

  it('rejects a transition into a state the state table does not declare', () => {
    const source = states.replace('| Ready | Run | — | Done |', '| Ready | Run | — | Gone |')
    const result = validateDocument('spec/contexts/demo/states.md', source)
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'transition names Gone, which the state table does not declare',
    ])
  })

  it('rejects a Kind outside the vocabulary and a machine without one initial state', () => {
    const source = states.replace('| Ready | initial |', '| Ready | 初期 |')
    const result = validateDocument('spec/contexts/demo/states.md', source)
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'state Ready has Kind "初期"; use one of initial, terminal, —',
      'state machine must declare exactly one initial state, found 0',
    ])
  })

  it('accepts scenarios.md and reports the ids it declares', () => {
    const source = `# Demo のシナリオ

### REQ-DEMO-001: A valid request succeeds
- ACTOR User
- GIVEN a valid request
- WHEN the request is submitted
- THEN the request succeeds
`
    const result = validateDocument('spec/contexts/demo/scenarios.md', source)
    expect(result.findings).toEqual([])
    expect(result.scenarioIds.map((scenario) => scenario.id)).toEqual(['REQ-DEMO-001'])
  })

  it('keeps normative scenarios out of the other documents', () => {
    const source = `# Demo の設計判断

### REQ-DEMO-002: A behavior
- ACTOR User
- WHEN it happens
- THEN it holds
`
    const result = validateDocument('spec/contexts/demo/decisions.md', source)
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'REQ-DEMO-002 must be declared in scenarios.md',
    ])
    expect(result.scenarioIds).toEqual([])
  })

  it('rejects a path the canonical layout does not define', () => {
    const result = validateDocument('spec/contexts/demo/notes.md', '# Notes\n')
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'not a canonical specification document',
    ])
  })
})

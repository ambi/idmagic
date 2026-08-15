import { describe, expect, it } from 'bun:test'
import { validateSpecification } from './specification-doc.ts'

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

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

  it('rejects links from current specifications to the historical ADR archive', () => {
    const source = valid.replace(
      'Demo behavior.',
      'Demo behavior. See [ADR-001](../decisions/ADR-001-old-choice.md).',
    )
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'current specification must not link to the historical ADR archive',
    )
  })

  it('rejects an empty-string state transition guard', () => {
    const source = valid.replace('| Ready | Run | allowed | Done |', '| Ready | Run | "" | Done |')
    expect(validateSpecification(source).findings.map((finding) => finding.message)).toContain(
      'unconditional state transition guard must use — instead of an empty string',
    )
  })
})

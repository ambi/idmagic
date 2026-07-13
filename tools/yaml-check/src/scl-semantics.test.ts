import { describe, expect, it } from 'bun:test'
import brokenExtension from './fixtures/scl-v3/invalid/broken-extension.json' with { type: 'json' }
import brokenFlow from './fixtures/scl-v3/invalid/broken-flow.json' with { type: 'json' }
import invalidCelScope from './fixtures/scl-v3/invalid/invalid-cel-scope.json' with { type: 'json' }
import legacySection from './fixtures/scl-v3/invalid/legacy-section.json' with { type: 'json' }
import missingAccess from './fixtures/scl-v3/invalid/missing-access.json' with { type: 'json' }
import unresolvedAuthorization from './fixtures/scl-v3/invalid/unresolved-authorization.json' with { type: 'json' }
import tenancy from './fixtures/scl-v3/valid/tenancy.json' with { type: 'json' }
import { validateAgainstSchema } from './lib.ts'
import { verifySclSemantics } from './scl-semantics.ts'

describe('SCL 3.0 validation', () => {
  it('accepts the representative Tenancy fixture', () => {
    expect(validateAgainstSchema('scl', tenancy, '')).toEqual([])
    expect(verifySclSemantics(tenancy)).toEqual([])
  })

  it('rejects removed SCL 2.0 sections', () => {
    const findings = validateAgainstSchema('scl', legacySection, '')
    expect(findings.some((finding) => finding.message.includes('invariants'))).toBe(true)
  })

  it('requires every interface to classify access', () => {
    const findings = validateAgainstSchema('scl', missingAccess, '')
    expect(findings.some((finding) => finding.message.includes('access'))).toBe(true)
  })

  it('resolves policy, principal, and protected resource references', () => {
    const messages = verifySclSemantics(unresolvedAuthorization).map((finding) => finding.message)
    expect(messages.some((message) => message.includes('MissingPolicy'))).toBe(true)
    expect(messages.some((message) => message.includes('MissingPrincipal'))).toBe(true)
    expect(messages.some((message) => message.includes('MissingResource'))).toBe(true)
  })

  it('rejects CEL roots outside the position-specific binding', () => {
    const findings = verifySclSemantics(invalidCelScope)
    expect(findings.some((finding) => finding.message.includes("binding 'output'"))).toBe(true)
  })

  it('rejects unreachable flow views and unresolved flow interfaces', () => {
    const messages = verifySclSemantics(brokenFlow).map((finding) => finding.message)
    expect(messages.some((message) => message.includes('unreachable'))).toBe(true)
    expect(messages.some((message) => message.includes("unknown interface 'Missing'"))).toBe(true)
  })

  it('rejects scenario extensions outside the main success path', () => {
    const findings = verifySclSemantics(brokenExtension)
    expect(findings.some((finding) => finding.message.includes('past main_success'))).toBe(true)
  })

  it('resolves model, interface, state, objective, and standard references', () => {
    const invalid = {
      system: 'demo',
      spec_version: '3.0',
      standards: {
        Standard: {
          title: 'Standard',
          url: 'https://example.com',
          requirements: [
            {
              id: 'REQ-1',
              strength: 'MUST',
              adoption: 'required',
              statement: 'required',
              refs: ['interfaces.Missing'],
            },
          ],
        },
      },
      models: {
        Status: { kind: 'enum', values: ['Active', 'Deleted'] },
        Tenant: {
          kind: 'entity',
          identity: 'missing_id',
          fields: { status: { type: 'Status' }, owner: { type: 'MissingModel' } },
        },
        Changed: { kind: 'event' },
      },
      interfaces: { Read: { access: 'public', errors: ['MissingError'], emits: ['Changed'] } },
      states: {
        Lifecycle: {
          target: 'Tenant',
          initial: 'Unknown',
          terminal: ['Deleted'],
          transitions: [{ from: 'Deleted', event: 'MissingEvent', to: 'Active' }],
        },
      },
      objectives: {
        Latency: {
          interface: 'Missing',
          indicator: 'measurement.ok',
          target: 0.99,
          window: '30d',
        },
      },
    }
    const messages = verifySclSemantics(invalid).map((finding) => finding.message)
    expect(messages.some((message) => message.includes("unknown field 'missing_id'"))).toBe(true)
    expect(messages.some((message) => message.includes("unknown type 'MissingModel'"))).toBe(true)
    expect(messages.some((message) => message.includes("unknown error model 'MissingError'"))).toBe(true)
    expect(messages.some((message) => message.includes("unknown event model 'MissingEvent'"))).toBe(true)
    expect(messages.some((message) => message.includes("unknown state value 'Unknown'"))).toBe(true)
    expect(messages.some((message) => message.includes('transition from terminal state'))).toBe(true)
    expect(messages.some((message) => message.includes("unknown interface 'Missing'"))).toBe(true)
    expect(messages.some((message) => message.includes("unknown SCL element 'interfaces.Missing'"))).toBe(
      true,
    )
  })

  it('requires timeslice budgeting to declare a slice', () => {
    const invalid = {
      system: 'demo',
      spec_version: '3.0',
      objectives: {
        Availability: {
          indicator: 'measurement.healthy',
          target: 0.999,
          window: '30d',
          budgeting: 'timeslices',
        },
      },
    }
    const findings = validateAgainstSchema('scl', invalid, '')
    expect(findings.some((finding) => finding.message.includes('slice'))).toBe(true)
  })
})

import { describe, expect, it } from 'bun:test'
import type { AdminAgent, AdminAuditEvent, WorkloadTrustBundle } from '../../types'
import { adminWorkloadIdentityDictionary as dict } from './AdminWorkloadIdentityPage.i18n'
import {
  attestationRejections,
  bundleConfirmation,
  bindingConfirmation,
  displayNameForID,
  formatDateTime,
  jwksSource,
  multipleCredentialBindings,
  parseAudiences,
  parseInlineJwks,
  rejectionReasonLabel,
  rejectionsForTrustBundle,
} from './presentation'

const bundle: WorkloadTrustBundle = {
  id: 'bundle-1',
  tenant_id: 'tenant-a',
  name: 'prod-cluster',
  trust_domain: 'example.org',
  issuer: 'https://issuer.example',
  jwks_uri: 'https://issuer.example/keys',
  has_inline_jwks: false,
  accepted_audiences: ['idmagic'],
  max_subject_token_ttl_seconds: 3600,
  status: 'enabled',
  created_at: '2026-08-16T00:00:00Z',
}

function auditEvent(payload: Record<string, unknown>, id = 'event-1'): AdminAuditEvent {
  return {
    id,
    tenant_id: 'tenant-a',
    type: 'WorkloadAttestationRejected',
    occurred_at: '2026-08-16T01:00:00Z',
    payload,
  }
}

describe('rejectionReasonLabel', () => {
  // 仕様 (WorkloadAttestationRejected.reason) が列挙する 11 個をすべて辞書へ写せていること。
  const reasons: [string, keyof (typeof dict)['en']][] = [
    ['unregistered_issuer', 'reasonUnregisteredIssuer'],
    ['trust_bundle_disabled', 'reasonTrustBundleDisabled'],
    ['jwks_unavailable', 'reasonJwksUnavailable'],
    ['invalid_signature', 'reasonInvalidSignature'],
    ['expired', 'reasonExpired'],
    ['audience_mismatch', 'reasonAudienceMismatch'],
    ['ttl_exceeded', 'reasonTtlExceeded'],
    ['no_binding_match', 'reasonNoBindingMatch'],
    ['ambiguous_match', 'reasonAmbiguousMatch'],
    ['agent_not_active', 'reasonAgentNotActive'],
    ['agent_unbound', 'reasonAgentUnbound'],
  ]

  it.each(reasons)('translates %s in both locales', (reason, key) => {
    expect(rejectionReasonLabel(reason, dict.ja)).toBe(dict.ja[key])
    expect(rejectionReasonLabel(reason, dict.en)).toBe(dict.en[key])
  })

  it('falls back to the raw identifier for a reason the dictionary does not know', () => {
    expect(rejectionReasonLabel('reason_from_a_newer_server', dict.en)).toBe(
      'reason_from_a_newer_server',
    )
  })
})

describe('attestationRejections', () => {
  it('normalizes the audit payload into presentable rows', () => {
    const rows = attestationRejections([
      auditEvent({ tenantId: 'tenant-a', reason: 'expired', trustBundleId: 'bundle-1' }),
    ])
    expect(rows).toEqual([
      {
        id: 'event-1',
        occurredAt: '2026-08-16T01:00:00Z',
        reason: 'expired',
        trustBundleId: 'bundle-1',
      },
    ])
  })

  it('keeps a rejection that carries no trustBundleId (unregistered issuer)', () => {
    const rows = attestationRejections([
      auditEvent({ tenantId: 'tenant-a', reason: 'unregistered_issuer' }),
    ])
    expect(rows[0]?.trustBundleId).toBeUndefined()
  })

  it('drops rows whose payload has no reason, because they cannot be presented', () => {
    expect(attestationRejections([auditEvent({ tenantId: 'tenant-a' })])).toEqual([])
    expect(attestationRejections([auditEvent({ reason: 42 })])).toEqual([])
  })
})

describe('rejectionsForTrustBundle', () => {
  it('keeps only the rejections attributed to the bundle', () => {
    const rows = attestationRejections([
      auditEvent({ reason: 'expired', trustBundleId: 'bundle-1' }, 'a'),
      auditEvent({ reason: 'expired', trustBundleId: 'bundle-2' }, 'b'),
      auditEvent({ reason: 'unregistered_issuer' }, 'c'),
    ])
    expect(rejectionsForTrustBundle(rows, 'bundle-1').map((row) => row.id)).toEqual(['a'])
  })
})

describe('jwksSource', () => {
  it('shows the URI when one is configured', () => {
    expect(jwksSource(bundle, dict.en)).toBe('https://issuer.example/keys')
  })

  it('names the inline JWKS when no URI is configured', () => {
    expect(jwksSource({ ...bundle, jwks_uri: undefined, has_inline_jwks: true }, dict.ja)).toBe(
      dict.ja.jwksSourceInline,
    )
  })

  it('reports that no key source is configured at all', () => {
    expect(jwksSource({ ...bundle, jwks_uri: undefined, has_inline_jwks: false }, dict.en)).toBe(
      dict.en.jwksSourceNone,
    )
  })
})

describe('formatDateTime', () => {
  it('renders an em dash when the timestamp is absent', () => {
    expect(formatDateTime(undefined, 'ja')).toBe('—')
  })

  it('renders a parseable timestamp in the requested locale', () => {
    expect(formatDateTime('2026-08-16T01:00:00Z', 'en')).not.toBe('—')
  })

  it('returns the raw value when it cannot be parsed', () => {
    expect(formatDateTime('not-a-timestamp', 'en')).toBe('not-a-timestamp')
  })
})

describe('bundleConfirmation', () => {
  it('states the cascade impact of a delete with the binding count', () => {
    expect(bundleConfirmation('delete', bundle, 3, dict.ja)).toEqual({
      title: dict.ja.deleteBundleTitle,
      message: dict.ja.deleteBundleMessage
        .replace('{name}', 'prod-cluster')
        .replace('{count}', '3'),
    })
  })

  it('drops the binding clause when the bundle has none', () => {
    expect(bundleConfirmation('delete', bundle, 0, dict.en).message).toBe(
      dict.en.deleteBundleMessageNoBindings.replace('{name}', 'prod-cluster'),
    )
  })

  it('describes disabling as reversible, unlike deletion', () => {
    expect(bundleConfirmation('disable', bundle, 2, dict.en)).toEqual({
      title: dict.en.disableBundleTitle,
      message: dict.en.disableBundleMessage
        .replace('{name}', 'prod-cluster')
        .replace('{count}', '2'),
    })
  })
})

describe('bindingConfirmation', () => {
  it('names the pattern being removed', () => {
    expect(bindingConfirmation('delete', 'spiffe://example.org/ns/prod/sa/*', dict.ja)).toEqual({
      title: dict.ja.deleteBindingTitle,
      message: dict.ja.deleteBindingMessage.replace(
        '{pattern}',
        'spiffe://example.org/ns/prod/sa/*',
      ),
    })
  })

  it('names the pattern being disabled', () => {
    expect(bindingConfirmation('disable', 'system:serviceaccount:prod:*', dict.en).title).toBe(
      dict.en.disableBindingTitle,
    )
  })
})

const agent: AdminAgent = {
  id: 'agent-1',
  tenant_id: 'tenant-a',
  name: 'checkout-bot',
  kind: 'autonomous',
  owner_user_id: 'user-1',
  status: 'active',
  roles: [],
  client_ids: ['client-1'],
  created_at: '2026-08-16T00:00:00Z',
}

describe('displayNameForID', () => {
  it('resolves the id to the name, for agents and trust bundles alike', () => {
    expect(displayNameForID('agent-1', [agent])).toBe('checkout-bot')
    expect(displayNameForID('bundle-1', [bundle])).toBe('prod-cluster')
  })

  it('falls back to the raw id when it is not in the page the screen holds', () => {
    expect(displayNameForID('agent-missing', [agent])).toBe('agent-missing')
  })
})

describe('multipleCredentialBindings', () => {
  it('flags an agent whose credential binding is not unique', () => {
    expect(multipleCredentialBindings('agent-1', [{ ...agent, client_ids: ['a', 'b'] }])).toBe(true)
  })

  it('stays quiet for a single binding, none at all, or an unknown agent', () => {
    expect(multipleCredentialBindings('agent-1', [agent])).toBe(false)
    expect(multipleCredentialBindings('agent-1', [{ ...agent, client_ids: [] }])).toBe(false)
    expect(multipleCredentialBindings('agent-missing', [agent])).toBe(false)
  })
})

describe('parseAudiences', () => {
  it('splits on commas and whitespace and drops empties', () => {
    expect(parseAudiences(' idmagic,  https://idmagic.example ')).toEqual([
      'idmagic',
      'https://idmagic.example',
    ])
  })

  it('returns an empty list for blank input', () => {
    expect(parseAudiences('   ')).toEqual([])
  })
})

describe('parseInlineJwks', () => {
  it('treats blank input as "no inline JWKS"', () => {
    expect(parseInlineJwks('  ')).toEqual({ ok: true, value: undefined })
  })

  it('parses a JWKS object', () => {
    expect(parseInlineJwks('{"keys": []}')).toEqual({ ok: true, value: { keys: [] } })
  })

  it('rejects JSON that is not an object', () => {
    expect(parseInlineJwks('[1, 2]')).toEqual({ ok: false })
    expect(parseInlineJwks('nope')).toEqual({ ok: false })
  })
})

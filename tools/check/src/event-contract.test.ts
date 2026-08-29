import { describe, expect, it } from 'bun:test'
import {
  collectConsumedEventFields,
  collectDeclaredEventFields,
  diffEventFieldVocabulary,
} from './event-contract.ts'

const DECLARATION = [
  'namespace IdMagic.Contract {',
  '',
  '@doc("The envelope every domain event is serialized into.")',
  'model DomainEventEnvelope {',
  '  type: string;',
  '  occurredAt: utcDateTime;',
  '  payload: DomainEventPayload;',
  '}',
  '',
  '@doc("The cross-context published fields of a domain event payload.")',
  'model DomainEventPayload {',
  '  @doc("The tenant the event belongs to.")',
  '  tenantId?: string;',
  '  @doc("The subject the event is about.")',
  '  userId?: string;',
  '  @doc("How many steps of delegation the token carried.")',
  '  delegationDepth?: int32;',
  '  @doc("The participants of the delegation chain.")',
  '  actorChain?: string[];',
  '  ...Record<unknown>;',
  '}',
  '',
  '}',
].join('\n')

const EXTRACTOR = [
  'set("actor.id", payloadString(rec.Payload, "actorUserId"))',
  'set("client.ip", payloadString(rec.Payload, "ip"))',
  'setAll("delegation.actor", payloadStrings(rec.Payload, "actorChain"))',
  'set("workflow_step.id", payloadNumberString(rec.Payload, "stepIndex"))',
].join('\n')

const DISPATCHER = [
  'tenantID := stringField(payload, "tenantId")',
  'userAgent := stringField(payload, "userAgent")',
  'reason, _ := payload["reason"].(string)',
  '"SessionImpersonationStarted": {',
  '\tRecipientField: "targetUserId",',
  '},',
].join('\n')

describe('collectDeclaredEventFields', () => {
  it('collects the property names of the payload model', () => {
    expect(collectDeclaredEventFields(DECLARATION)).toEqual(
      new Set(['tenantId', 'userId', 'delegationDepth', 'actorChain']),
    )
  })

  it('does not collect the envelope properties, which are not payload fields', () => {
    const declared = collectDeclaredEventFields(DECLARATION)
    expect(declared.has('occurredAt')).toBe(false)
    expect(declared.has('payload')).toBe(false)
  })

  it('returns an empty set when the payload model is absent', () => {
    expect(collectDeclaredEventFields('model Unrelated {\n  id: string;\n}')).toEqual(new Set())
  })

  it('stops at the closing brace rather than absorbing the next model', () => {
    const followed = [
      'model DomainEventPayload {',
      '  tenantId?: string;',
      '}',
      '',
      'model SomethingElse {',
      '  unrelatedField?: string;',
      '}',
    ].join('\n')
    expect(collectDeclaredEventFields(followed)).toEqual(new Set(['tenantId']))
  })
})

describe('collectConsumedEventFields', () => {
  it('reads the audit extractor payload accessors', () => {
    expect(collectConsumedEventFields(new Map([['extractor.go', EXTRACTOR]]))).toEqual(
      new Set(['actorUserId', 'ip', 'actorChain', 'stepIndex']),
    )
  })

  it('reads the notification dispatcher accessors, guards, and recipient fields', () => {
    expect(collectConsumedEventFields(new Map([['dispatch.go', DISPATCHER]]))).toEqual(
      new Set(['tenantId', 'userAgent', 'reason', 'targetUserId']),
    )
  })

  it('ignores a map key that is not a payload accessor', () => {
    expect(collectConsumedEventFields(new Map([['x.go', 'attrs["actor.id"] = values']]))).toEqual(
      new Set(),
    )
  })
})

describe('diffEventFieldVocabulary', () => {
  it('reports a consumed field the declaration does not carry', () => {
    const diff = diffEventFieldVocabulary(new Set(['tenantId']), new Set(['tenantId', 'userId']))
    expect(diff).toEqual({ missing: ['userId'], undeclared: [] })
  })

  it('reports a declared field nothing consumes', () => {
    const diff = diffEventFieldVocabulary(new Set(['tenantId', 'stale']), new Set(['tenantId']))
    expect(diff).toEqual({ missing: [], undeclared: ['stale'] })
  })

  it('sorts each side so the failure reads the same on every run', () => {
    const diff = diffEventFieldVocabulary(new Set(), new Set(['userId', 'agentId', 'tenantId']))
    expect(diff.missing).toEqual(['agentId', 'tenantId', 'userId'])
  })

  it('finds no difference when the two vocabularies agree', () => {
    const same = new Set(['tenantId', 'userId'])
    expect(diffEventFieldVocabulary(same, new Set(same))).toEqual({ missing: [], undeclared: [] })
  })
})

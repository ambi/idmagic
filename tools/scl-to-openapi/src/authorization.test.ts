import { describe, expect, it } from 'bun:test'
import type { SclDocument } from '../../scl-to-html/src/types.ts'
import { buildAuthorizationMetadata } from './authorization.ts'

const document = (): SclDocument => ({
  system: 'demo',
  spec_version: '3.0',
  models: {
    User: { kind: 'entity', identity: 'id', fields: { id: { type: 'UUID' } } },
    Tenant: { kind: 'entity', identity: 'id', fields: { id: { type: 'UUID' } } },
  },
  interfaces: {
    UpdateTenant: {
      access: { policies: ['Write', 'TenantMember'], resource: { type: 'Tenant', id: 'input.id' } },
    },
    ReadTenant: {
      access: { policies: ['TenantMember'], resource: { type: 'Tenant', id: 'input.id' } },
    },
    DeleteTenant: {
      access: { policies: ['TenantMember', 'Write'], resource: { type: 'Tenant', id: 'input.id' } },
    },
    Health: { access: 'public' },
  },
  authorization: {
    principals: {
      Member: { type: 'User', matches: ['principal.id != ""'] },
    },
    policies: {
      TenantMember: { effect: 'permit', principal: 'Member' },
      Write: { effect: 'permit', principal: 'Member' },
    },
  },
})

describe('buildAuthorizationMetadata', () => {
  it('groups actions by sorted policy set and excludes public operations', () => {
    const metadata = buildAuthorizationMetadata(document())
    expect(metadata.groups.map((group) => group.name)).toEqual([
      'policy:TenantMember',
      'policy:TenantMember+Write',
    ])
    expect(metadata.groups[1]?.actions.map((action) => action.name)).toEqual([
      'DeleteTenant',
      'UpdateTenant',
    ])
  })

  it('derives Cedar appliesTo types through policy principals and resources', () => {
    const action = buildAuthorizationMetadata(document()).groups[0]?.actions[0]
    expect(action?.appliesTo).toEqual({ principalTypes: ['User'], resourceTypes: ['Tenant'] })
    expect(action?.resourceId).toBe('input.id')
  })

  it('is deterministic for repeated inputs', () => {
    expect(buildAuthorizationMetadata(document())).toEqual(buildAuthorizationMetadata(document()))
  })
})

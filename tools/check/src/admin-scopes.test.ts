import { describe, expect, it } from 'bun:test'
import {
  collectAdminOperations,
  isAdminScope,
  parseApiTokenScopes,
  verifyAdminScopes,
} from './admin-scopes.ts'

const document = {
  paths: {
    '/api/admin/v1/things': {
      get: { operationId: 'ListThings', 'x-api-token-scopes': ['things:read'] },
      post: { operationId: 'CreateThing', 'x-api-token-scopes': ['things:write'] },
      parameters: [],
    },
    '/api/account/v1/profile': {
      get: { operationId: 'GetProfile' },
    },
  },
}

describe('collectAdminOperations', () => {
  it('takes only admin operations and their declared scopes', () => {
    expect(collectAdminOperations(document)).toEqual([
      {
        name: 'CreateThing',
        path: '/api/admin/v1/things',
        method: 'POST',
        scopes: ['things:write'],
      },
      { name: 'ListThings', path: '/api/admin/v1/things', method: 'GET', scopes: ['things:read'] },
    ])
  })

  it('reports an undeclared operation as having no scopes', () => {
    const operations = collectAdminOperations({
      paths: { '/api/admin/v1/things': { get: { operationId: 'ListThings' } } },
    })
    expect(operations[0]?.scopes).toEqual([])
  })
})

describe('parseApiTokenScopes', () => {
  it('reads the enum values out of the TypeSpec source', () => {
    const source = 'enum ApiTokenScope {\n  a_read: "a:read",\n  b_write: "b:write",\n}\n'
    expect(parseApiTokenScopes(source)).toEqual(['a:read', 'b:write'])
  })

  it('fails loudly when the enum is absent', () => {
    expect(() => parseApiTokenScopes('enum Other {}')).toThrow()
  })
})

describe('isAdminScope', () => {
  it('excludes the vocabularies enforced by the account and SCIM APIs', () => {
    expect(isAdminScope('users:read')).toBe(true)
    expect(isAdminScope('account:read')).toBe(false)
    expect(isAdminScope('scim:users:read')).toBe(false)
  })
})

describe('verifyAdminScopes', () => {
  const operations = collectAdminOperations(document)

  it('accepts a fully declared surface', () => {
    expect(verifyAdminScopes(operations, ['things:read', 'things:write'])).toEqual([])
  })

  it('reports an operation that declares nothing', () => {
    const undeclared = collectAdminOperations({
      paths: { '/api/admin/v1/things': { get: { operationId: 'ListThings' } } },
    })
    expect(verifyAdminScopes(undeclared, []).join('\n')).toContain('declares no x-api-token-scopes')
  })

  it('reports a scope the vocabulary does not define', () => {
    expect(verifyAdminScopes(operations, ['things:read']).join('\n')).toContain(
      'ApiTokenScope does not define',
    )
  })

  it('reports a vocabulary entry no operation requires', () => {
    expect(
      verifyAdminScopes(operations, ['things:read', 'things:write', 'unused:read']).join('\n'),
    ).toContain('no admin operation requires it')
  })

  it('accepts interactive_session without it being part of the vocabulary', () => {
    const sessionOnly = collectAdminOperations({
      paths: {
        '/api/admin/v1/things': {
          get: { operationId: 'ListThings', 'x-api-token-scopes': ['interactive_session'] },
        },
      },
    })
    expect(verifyAdminScopes(sessionOnly, [])).toEqual([])
  })

  it('rejects mixing interactive_session with a granular scope', () => {
    const mixed = collectAdminOperations({
      paths: {
        '/api/admin/v1/things': {
          get: {
            operationId: 'ListThings',
            'x-api-token-scopes': ['interactive_session', 'things:read'],
          },
        },
      },
    })
    expect(verifyAdminScopes(mixed, ['things:read']).join('\n')).toContain('mixes')
  })
})

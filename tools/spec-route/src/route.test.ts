import { describe, expect, it } from 'bun:test'
import {
  joinableFacts,
  parseDeclaredErrors,
  parseDeclaredOperations,
  parseGeneratedContract,
  rankCandidates,
  ruleOf,
} from './route.ts'

describe('ruleOf', () => {
  it('reads the rule out of an example id', () => {
    expect(ruleOf('EX-OAUTH2-003-02')).toBe('REQ-OAUTH2-003')
  })

  it('leaves a rule id alone', () => {
    expect(ruleOf('REQ-OAUTH2-003')).toBe('REQ-OAUTH2-003')
  })
})

describe('joinableFacts', () => {
  it('reads the error type an outcome step names', () => {
    const join = joinableFacts(['Then 操作は AccessDeniedError で拒否される'])
    expect([...join.errorTypes]).toEqual(['AccessDeniedError'])
  })

  // 通常経路の具体例はエラー型を 1 つも名指さない。エラー型だけで絞る実装では、
  // その具体例に対して Context の全 operation が同点で並ぶ。
  it('reads the endpoint a normal-path step names', () => {
    const join = joinableFacts(['When クライアントがグラントを `/token` で交換する'])
    expect([...join.paths]).toEqual(['/token'])
    expect([...join.errorTypes]).toEqual([])
  })

  it('reads the granular scope a step names', () => {
    const join = joinableFacts(['But oauth-clients:read だけで変更を要求する'])
    expect([...join.scopes]).toEqual(['oauth-clients:read'])
  })
})

describe('rankCandidates', () => {
  const operations = [
    { name: 'Authorize', method: 'GET', path: '/authorize', scopes: [] },
    {
      name: 'CreateAdminOAuth2Client',
      method: 'POST',
      path: '/api/admin/v1/clients',
      scopes: ['oauth-clients:write'],
    },
    {
      name: 'ListAdminOAuth2Clients',
      method: 'GET',
      path: '/api/admin/v1/clients',
      scopes: ['oauth-clients:read'],
    },
  ]
  const errors = new Map([
    ['Authorize', new Map([['403', ['AccessDeniedError']]])],
    ['CreateAdminOAuth2Client', new Map([['403', ['AccessDeniedError']]])],
    ['ListAdminOAuth2Clients', new Map([['403', ['AccessDeniedError']]])],
  ])

  // これが scope で結合する理由である。AccessDeniedError は 3 つとも答えるので、
  // 型だけでは順位がつかない。
  it('puts the operation whose scope the example names first', () => {
    const ranked = rankCandidates(
      operations,
      errors,
      joinableFacts([
        'But oauth-clients:read だけで変更を要求する',
        'Then AccessDeniedError で拒否',
      ]),
    )
    expect(ranked[0]?.operation.name).toBe('ListAdminOAuth2Clients')
    expect(ranked[0]?.matched).toEqual(['403 AccessDeniedError', 'scope oauth-clients:read'])
    expect(ranked[1]?.matched).toEqual(['403 AccessDeniedError'])
  })

  it('returns every operation even when nothing joins', () => {
    const ranked = rankCandidates(operations, errors, joinableFacts(['When 何も名指さない']))
    expect(ranked).toHaveLength(3)
    expect(ranked.every((candidate) => candidate.matched.length === 0)).toBe(true)
  })

  it('orders operations with equal matches by name, so the output is stable', () => {
    const ranked = rankCandidates(operations, errors, joinableFacts(['Then AccessDeniedError']))
    expect(ranked.map((candidate) => candidate.operation.name)).toEqual([
      'Authorize',
      'CreateAdminOAuth2Client',
      'ListAdminOAuth2Clients',
    ])
  })
})

describe('parseGeneratedContract', () => {
  it('reads the method, path and scopes of a row', () => {
    const source =
      '\t"CreateAdminOAuth2Client": {Method: "POST", Path: "/api/admin/v1/clients", ' +
      'Deprecated: false, ApiTokenScopes: []string{"oauth-clients:write"}},\n'
    expect(parseGeneratedContract(source)).toEqual([
      {
        name: 'CreateAdminOAuth2Client',
        method: 'POST',
        path: '/api/admin/v1/clients',
        scopes: ['oauth-clients:write'],
      },
    ])
  })

  // 粒度スコープを宣言しない operation は nil を持つ。行を落とすと、ブラウザーの
  // エンドポイントが候補から消える。
  it('reads a row that declares no scope', () => {
    const source =
      '\t"Token": {Method: "POST", Path: "/token", Deprecated: false, ApiTokenScopes: nil},\n'
    expect(parseGeneratedContract(source)).toEqual([
      { name: 'Token', method: 'POST', path: '/token', scopes: [] },
    ])
  })
})

describe('parseDeclaredErrors and parseDeclaredOperations', () => {
  const typespec = [
    'union AuthorizeError403Body {',
    '  AccessDeniedError,',
    '  ConsentRequiredError,',
    '}',
    '@get',
    'op Authorize(',
    '  @query client_id: string,',
    '): AuthorizeOk | AuthorizeError403;',
  ].join('\n')

  it('reads the error types a status answers with', () => {
    expect([...(parseDeclaredErrors(typespec).get('Authorize')?.get('403') ?? [])]).toEqual([
      'AccessDeniedError',
      'ConsentRequiredError',
    ])
  })

  it('reads the operation names, and not the ops named inside a return type', () => {
    expect([...parseDeclaredOperations(typespec)]).toEqual(['Authorize'])
  })
})

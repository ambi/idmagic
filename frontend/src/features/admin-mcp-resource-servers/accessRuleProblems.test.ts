import { describe, expect, it } from 'bun:test'
import type { ResourceAccessRule } from '../../types'
import { accessRuleProblems, constraintsRepresentable, hasProblems } from './accessRuleProblems'

const base: ResourceAccessRule = {
  resource_type: 'idmagic_mcp_operation',
  relation: 'invoke',
  scopes: ['mcp'],
  tool: { name: 'delete_document', display_name: 'Delete' },
}

describe('accessRuleProblems', () => {
  //spec:covers REQ-OAUTH2-090: 人の承認を求める行に認可詳細の種類がなければ、その欄に不足を示す。
  it('requires a detail type only while approval is required', () => {
    expect(accessRuleProblems({ ...base, requires_approved_operation: true })).toMatchObject({
      authorizationDetailType: 'detailTypeRequired',
    })
    expect(
      accessRuleProblems({
        ...base,
        requires_approved_operation: true,
        authorization_detail_type: '',
      }).authorizationDetailType,
    ).toBe('detailTypeRequired')
    const approved = {
      ...base,
      requires_approved_operation: true,
      authorization_detail_type: 'doc',
    }
    expect(hasProblems(accessRuleProblems(approved))).toBe(false)
    expect(hasProblems(accessRuleProblems({ ...base, requires_approved_operation: false }))).toBe(
      false,
    )
  })

  //spec:covers REQ-OAUTH2-090: 承認の条件と明細の種類の対、人の承認との併用、空の条件を欄ごとに示す。
  it('pairs the approval condition with its detail type', () => {
    expect(
      accessRuleProblems({ ...base, condition: { require_approval_when: 'true' } })
        .approvalDetailType,
    ).toBe('approvalDetailTypeRequired')
    expect(
      accessRuleProblems({
        ...base,
        condition: { deny_when: 'false', approval_detail_type: 'doc' },
      }).approvalDetailType,
    ).toBe('approvalDetailTypeUnused')
    expect(
      accessRuleProblems({
        ...base,
        requires_approved_operation: true,
        authorization_detail_type: 'doc',
        condition: { require_approval_when: 'true', approval_detail_type: 'doc' },
      }).requireApprovalWhen,
    ).toBe('approvalConditionConflict')
    expect(accessRuleProblems({ ...base, condition: { version: 'abc' } }).denyWhen).toBe(
      'conditionEmpty',
    )
    expect(
      hasProblems(
        accessRuleProblems({
          ...base,
          condition: { require_approval_when: 'true', approval_detail_type: 'doc' },
        }),
      ),
    ).toBe(false)
    expect(hasProblems(accessRuleProblems({ ...base, condition: { deny_when: 'false' } }))).toBe(
      false,
    )
  })

  //spec:covers REQ-OAUTH2-090: 交換先は https の URL だけを受け付け、承認を求める操作との併用と 9 件以上を示す。
  it('accepts https exchange resources only without approval', () => {
    const exchange = {
      ...base,
      token_exchange_resources: ['https://api.example', 'http://api.example', ''],
    }
    expect(accessRuleProblems(exchange).tokenExchangeEntries).toEqual([
      undefined,
      'exchangeHttps',
      'exchangeHttps',
    ])
    // サーバーは https と空でないホストだけを見るので、ポートとパスは受け付ける。
    expect(
      hasProblems(
        accessRuleProblems({ ...base, token_exchange_resources: ['HTTPS://api.example:8443/v1'] }),
      ),
    ).toBe(false)
    expect(
      accessRuleProblems({
        ...base,
        condition: { require_approval_when: 'true', approval_detail_type: 'doc' },
        token_exchange_resources: ['https://api.example'],
      }).tokenExchange,
    ).toBe('exchangeWithApproval')
    expect(
      accessRuleProblems({
        ...base,
        requires_approved_operation: true,
        authorization_detail_type: 'doc',
        token_exchange_resources: ['https://api.example'],
      }).tokenExchange,
    ).toBe('exchangeWithApproval')
    const nine = Array.from({ length: 9 }, (_, index) => `https://api${index}.example`)
    expect(accessRuleProblems({ ...base, token_exchange_resources: nine }).tokenExchange).toBe(
      'exchangeTooMany',
    )
    expect(
      hasProblems(accessRuleProblems({ ...base, token_exchange_resources: nine.slice(0, 8) })),
    ).toBe(false)
  })

  //spec:covers REQ-OAUTH2-090: 上限の回数と窓は範囲の両端を受け付け、外側、整数でない値、単位と窓の組の重複、9 件以上を示す。
  it('checks the limit ranges at both edges and duplicates', () => {
    const limit = (max_requests: number, window_seconds: number) => ({
      ...base,
      limits: [{ max_requests, window_seconds }],
    })
    expect(hasProblems(accessRuleProblems(limit(1, 1)))).toBe(false)
    expect(hasProblems(accessRuleProblems(limit(1_000_000_000, 2_592_000)))).toBe(false)
    expect(accessRuleProblems(limit(0, 2_592_001)).limits).toEqual([
      { maxRequests: 'limitRequestsRange', windowSeconds: 'limitWindowRange' },
    ])
    expect(accessRuleProblems(limit(1_000_000_001, 1.5)).limits).toEqual([
      { maxRequests: 'limitRequestsRange', windowSeconds: 'limitWindowRange' },
    ])
    const duplicated = accessRuleProblems({
      ...base,
      limits: [
        { by: ['user', 'agent'], max_requests: 1, window_seconds: 60 },
        { by: ['agent', 'user'], max_requests: 5, window_seconds: 60 },
        { by: ['agent'], max_requests: 5, window_seconds: 60 },
      ],
    })
    expect(duplicated.limits.map((problem) => problem.windowSeconds)).toEqual([
      undefined,
      'limitDuplicate',
      undefined,
    ])
    const nine = Array.from({ length: 9 }, (_, index) => ({
      max_requests: 1,
      window_seconds: index + 1,
    }))
    expect(accessRuleProblems({ ...base, limits: nine }).limitsGroup).toBe('limitsTooMany')
  })
})

describe('constraintsRepresentable', () => {
  //spec:covers REQ-OAUTH2-090: 入力欄で表せない型の制約を見分け、読み取り専用にする規則を選ぶ。
  it('accepts contract-shaped constraints and rejects other types', () => {
    expect(constraintsRepresentable(base)).toBe(true)
    expect(
      constraintsRepresentable({
        ...base,
        requires_approved_operation: true,
        authorization_detail_type: 'doc',
        condition: { deny_when: 'false', version: 'abc' },
        limits: [{ by: ['agent', 'resource'], max_requests: 0, window_seconds: 1.5 }],
        token_exchange_resources: ['not a url'],
      }),
    ).toBe(true)
    const odd = (change: Record<string, unknown>) =>
      constraintsRepresentable({ ...base, ...change } as ResourceAccessRule)
    expect(odd({ requires_approved_operation: 'yes' })).toBe(false)
    expect(odd({ authorization_detail_type: 1 })).toBe(false)
    expect(odd({ condition: 'true' })).toBe(false)
    expect(odd({ condition: [] })).toBe(false)
    expect(odd({ condition: { deny_when: true } })).toBe(false)
    expect(odd({ limits: {} })).toBe(false)
    expect(odd({ limits: [{ max_requests: '1', window_seconds: 1 }] })).toBe(false)
    expect(odd({ limits: [{ by: ['tenant'], max_requests: 1, window_seconds: 1 }] })).toBe(false)
    expect(odd({ limits: [{ by: ['user', 'user'], max_requests: 1, window_seconds: 1 }] })).toBe(
      false,
    )
    expect(odd({ token_exchange_resources: 'https://api.example' })).toBe(false)
    expect(odd({ token_exchange_resources: [1] })).toBe(false)
  })
})

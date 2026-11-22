import { expect, it } from 'bun:test'
import type {
  AgentAccessSummary,
  AdminAgent,
  AgentGuardrailPolicySet,
  McpResourceServer,
  RelationTuple,
  ResourceAccessRule,
} from '../../types'
import {
  CONNECTION_CHECK_ITEMS,
  type CatalogResult,
  checkMcpConnections,
  type ConnectionCheckGap,
  type ConnectionCheckItem,
} from './mcpConnectionCheck'

const agent: AdminAgent = {
  id: 'agent-1',
  tenant_id: 't',
  name: 'Assistant',
  owner_user_id: 'u',
  kind: 'autonomous',
  status: 'active',
  client_ids: [],
  created_at: '',
}

const summary = (clients: AgentAccessSummary['clients']): AgentAccessSummary => ({
  as_of: '2026-11-13T00:00:00Z',
  clients,
  delegations: [],
})

const boundClient = { client_id: 'c1', scopes: ['read', 'write'], missing: false }

const toolRule = (
  name: string,
  assignmentID: string,
  scopes: string[],
  extra: Partial<ResourceAccessRule> = {},
): ResourceAccessRule => ({
  resource_type: 'idmagic_mcp_operation',
  relation: 'invoke',
  scopes,
  tool: { name, display_name: name, assignment_resource_id: assignmentID },
  ...extra,
})

const server = (overrides: Partial<McpResourceServer> = {}): McpResourceServer => ({
  id: 'srv-1',
  tenant_id: 't',
  name: 'Docs',
  resource: 'https://docs.example',
  scopes: ['read'],
  client_id: 'eval-client',
  state: 'active',
  created_at: '',
  updated_at: '',
  access_rules: [toolRule('read', 'op-1', ['read'])],
  ...overrides,
})

const invokeTuple = (resourceID: string): RelationTuple => ({
  resource_type: 'idmagic_mcp_operation',
  resource_id: resourceID,
  relation: 'invoke',
  subject_type: 'agent',
  subject_id: agent.id,
})

const catalogs: Record<string, CatalogResult> = {
  'srv-1': { tools: [], complete: true, received_at: '2026-11-13T00:00:00Z' },
}

function check(overrides: {
  agent?: AdminAgent
  summary?: AgentAccessSummary
  guardrails?: AgentGuardrailPolicySet
  servers?: McpResourceServer[]
  tuples?: RelationTuple[]
  catalogs?: Record<string, CatalogResult>
  now?: Date
}) {
  const result = checkMcpConnections({
    agent: overrides.agent ?? agent,
    summary: overrides.summary ?? summary([boundClient]),
    guardrails: overrides.guardrails ?? {},
    resourceServers: overrides.servers ?? [server()],
    tuples: overrides.tuples ?? [invokeTuple('op-1')],
    catalogs: overrides.catalogs ?? catalogs,
    now: overrides.now,
  })
  expect(result).toHaveLength(overrides.servers?.length ?? 1)
  return result[0]!
}

const gapsOf = (result: ReturnType<typeof check>, item: ConnectionCheckItem) => result.items[item]

//spec:covers REQ-OAUTH2-094: 登録、割当、トークンの上限、一覧の取得を保存値だけで判定する。
it('reports no gaps for a fully configured connection', () => {
  const result = check({})
  for (const item of CONNECTION_CHECK_ITEMS) {
    expect(gapsOf(result, item)).toEqual([])
  }
})

//spec:covers REQ-OAUTH2-094: 登録の不足は状態と評価用クライアントの紐付けで示す。
it('flags inactive registration and missing evaluation client separately', () => {
  const result = check({
    servers: [server({ state: 'disabled', client_id: undefined })],
  })
  expect(gapsOf(result, 'registration')).toEqual([
    'registration_inactive',
    'registration_no_client',
  ])
})

//spec:covers REQ-OAUTH2-094: 割当の不足は Agent の状態、資格情報、ツール、操作割当で示す。
it('flags each assignment gap independently', () => {
  const cases: Array<{
    name: string
    overrides: Parameters<typeof check>[0]
    expected: ConnectionCheckGap[]
  }> = [
    {
      name: 'agent not active',
      overrides: { agent: { ...agent, status: 'disabled' } },
      expected: ['assignment_agent_inactive'],
    },
    {
      name: 'no bound credentials',
      overrides: { summary: summary([{ client_id: 'gone', scopes: [], missing: true }]) },
      expected: ['assignment_no_credentials'],
    },
    {
      name: 'no published tools',
      overrides: { servers: [server({ access_rules: [] })] },
      expected: ['assignment_no_tools'],
    },
    {
      name: 'operation not assigned',
      overrides: { tuples: [] },
      expected: ['assignment_unassigned'],
    },
  ]
  for (const { name, overrides, expected } of cases) {
    expect({ name, gaps: gapsOf(check(overrides), 'assignment') }).toEqual({
      name,
      gaps: expected,
    })
  }
})

//spec:covers REQ-OAUTH2-094: 取得上限に達した材料では、割当がないと判定せず確かめられないことを示す。
it('does not claim missing assignments when the tuples reached the retrieval limit', () => {
  const unrelated = Array.from({ length: 1000 }, (_, index) => invokeTuple(`other-${index}`))
  expect(gapsOf(check({ tuples: unrelated }), 'assignment')).toEqual(['assignment_truncated'])
  expect(gapsOf(check({ tuples: [...unrelated, invokeTuple('op-1')] }), 'assignment')).toEqual([])
})

//spec:covers REQ-OAUTH2-094: トークンの上限は操作ごとのスコープ被覆とガードレールの deny で判定する。
it('requires a bound client covering each assigned operation scopes', () => {
  const result = check({
    summary: summary([{ client_id: 'c1', scopes: ['read'], missing: false }]),
    servers: [
      server({
        access_rules: [toolRule('read', 'op-1', ['read']), toolRule('write', 'op-2', ['write'])],
      }),
    ],
    tuples: [invokeTuple('op-1'), invokeTuple('op-2')],
  })
  expect(gapsOf(result, 'tokenLimits')).toEqual(['token_limits_scope'])
})

//spec:covers REQ-OAUTH2-094: require_approval への降格は発行の不足に数えない。
it('does not count require_approval demotions as token limit gaps', () => {
  const result = check({
    guardrails: {
      agent: {
        version: 'v',
        updated_at: '',
        allowed_resources: ['https://other.example'],
        on_violation: { allowed_resources: 'require_approval' },
      },
      tenant_default: { version: 'v', updated_at: '', require_approval: true },
    },
  })
  expect(gapsOf(result, 'tokenLimits')).toEqual([])
})

//spec:covers REQ-OAUTH2-094: ガードレールの deny を、資源、認可詳細の種類、期限切れと、拒否した層で区別して示す。
it('flags guardrail denials by kind and by the layer that denies', () => {
  const result = check({
    servers: [
      server({
        access_rules: [
          toolRule('read', 'op-1', ['read'], {
            authorization_detail_type: 'mcp_read',
            condition: { approval_detail_type: 'mcp_approval' },
          }),
        ],
      }),
    ],
    guardrails: {
      agent: {
        version: 'v',
        updated_at: '',
        allowed_authorization_detail_types: ['mcp_read'],
        expires_at: '2026-11-12T00:00:00Z',
      },
      tenant_default: {
        version: 'v',
        updated_at: '',
        allowed_resources: ['https://other.example'],
      },
    },
    now: new Date('2026-11-13T00:00:00Z'),
  })
  expect(gapsOf(result, 'tokenLimits')).toEqual([
    'agent_guardrail_detail_type',
    'agent_guardrail_expired',
    'tenant_guardrail_resource',
  ])
})

//spec:covers REQ-OAUTH2-094: カタログの未到着と取得の失敗を区別する。
it('distinguishes missing catalog from catalog fetch failure', () => {
  expect(
    gapsOf(check({ catalogs: { 'srv-1': { tools: [], complete: false } } }), 'catalog'),
  ).toEqual(['catalog_not_received'])
  expect(gapsOf(check({ catalogs: { 'srv-1': 'failed' } }), 'catalog')).toEqual([
    'catalog_fetch_failed',
  ])
})

import { RELATION_TUPLE_LIST_LIMIT } from '../../api/admin'
import { assignedOperationIDs, operationResourceID } from '../../lib/mcpAccess'
import type {
  AgentAccessSummary,
  AdminAgent,
  AgentGuardrailPolicy,
  AgentGuardrailPolicySet,
  McpResourceServer,
  RelationTuple,
  ResourceAccessRule,
  ToolCatalog,
} from '../../types'

// MCP 接続の確認項目 (REQ-OAUTH2-094)。IdMagic が保存値だけで判定できる四項目で、
// MCP サーバーへの到達を要する確認は含まない。
export type ConnectionCheckItem = 'registration' | 'assignment' | 'tokenLimits' | 'catalog'

export const CONNECTION_CHECK_ITEMS: ConnectionCheckItem[] = [
  'registration',
  'assignment',
  'tokenLimits',
  'catalog',
]

// ConnectionCheckGap は項目の不足の理由である。文言と修正先は画面の辞書が持つ。
// ガードレールの拒否は、修正先が異なるため Agent 別とテナントのデフォルトの層で分ける。
export type ConnectionCheckGap =
  | 'registration_inactive'
  | 'registration_no_client'
  | 'assignment_agent_inactive'
  | 'assignment_no_credentials'
  | 'assignment_no_tools'
  | 'assignment_unassigned'
  | 'assignment_truncated'
  | 'token_limits_scope'
  | 'agent_guardrail_resource'
  | 'agent_guardrail_detail_type'
  | 'agent_guardrail_expired'
  | 'tenant_guardrail_resource'
  | 'tenant_guardrail_detail_type'
  | 'tenant_guardrail_expired'
  | 'catalog_not_received'
  | 'catalog_fetch_failed'

// CatalogResult は接続先ごとのカタログで、取得に失敗した接続先は 'failed' を置く。
export type CatalogResult = ToolCatalog | 'failed'

// McpConnectionCheck は一つの接続先の確認結果である。不足のない項目は空の配列を持つ。
export type McpConnectionCheck = {
  server: McpResourceServer
  items: Record<ConnectionCheckItem, ConnectionCheckGap[]>
}

type ToolRule = ResourceAccessRule & { tool: NonNullable<ResourceAccessRule['tool']> }

function toolRules(server: McpResourceServer): ToolRule[] {
  return (server.access_rules ?? []).filter((rule): rule is ToolRule => rule.tool !== undefined)
}

function coversScopes(clientScopes: string[], required: string[]): boolean {
  return required.every((scope) => clientScopes.includes(scope))
}

const GUARDRAIL_GAPS = {
  agent: {
    resource: 'agent_guardrail_resource',
    detailType: 'agent_guardrail_detail_type',
    expired: 'agent_guardrail_expired',
  },
  tenant: {
    resource: 'tenant_guardrail_resource',
    detailType: 'tenant_guardrail_detail_type',
    expired: 'tenant_guardrail_expired',
  },
} as const satisfies Record<string, Record<string, ConnectionCheckGap>>

// guardrailDeniedGaps は、一つの層のガードレールが、この接続先の発行を deny で拒否する項目を返す。
// 違反時の扱いを省いた項目は deny である。require_approval への降格と、ポリシー全体の
// require_approval、接続に関係しない項目 (発行の計数、代行の段数、トークンの有効期間) は対象外である。
function guardrailDeniedGaps(
  layer: keyof typeof GUARDRAIL_GAPS,
  policy: AgentGuardrailPolicy | undefined,
  server: McpResourceServer,
  detailTypes: string[],
  now: Date,
): ConnectionCheckGap[] {
  if (!policy) return []
  const gaps = GUARDRAIL_GAPS[layer]
  const deny = (action: string | undefined) => (action ?? 'deny') === 'deny'
  const result: ConnectionCheckGap[] = []
  const allowedResources = policy.allowed_resources
  if (
    allowedResources !== undefined &&
    !allowedResources.includes(server.resource) &&
    deny(policy.on_violation?.allowed_resources)
  ) {
    result.push(gaps.resource)
  }
  const allowedTypes = policy.allowed_authorization_detail_types
  if (
    allowedTypes !== undefined &&
    detailTypes.some((type) => !allowedTypes.includes(type)) &&
    deny(policy.on_violation?.allowed_authorization_detail_types)
  ) {
    result.push(gaps.detailType)
  }
  if (
    policy.expires_at !== undefined &&
    now.getTime() >= new Date(policy.expires_at).getTime() &&
    deny(policy.on_violation?.expires_at)
  ) {
    result.push(gaps.expired)
  }
  return result
}

// requiredDetailTypes は、割当済みの規則がトークンに求める認可詳細の種類を集める。
function requiredDetailTypes(rules: ResourceAccessRule[]): string[] {
  const types = new Set<string>()
  for (const rule of rules) {
    if (rule.authorization_detail_type) types.add(rule.authorization_detail_type)
    if (rule.condition?.approval_detail_type) types.add(rule.condition.approval_detail_type)
  }
  return [...types]
}

// checkMcpConnections は、接続先ごとに四項目の状態を保存値から求める (REQ-OAUTH2-094)。
// tuples は Agent を主体とする操作の関係で、取得上限の件数に達していれば不完全とみなす。
export function checkMcpConnections({
  agent,
  summary,
  guardrails,
  resourceServers,
  tuples,
  catalogs,
  now = new Date(),
}: {
  agent: AdminAgent
  summary: AgentAccessSummary
  guardrails: AgentGuardrailPolicySet
  resourceServers: McpResourceServer[]
  tuples: RelationTuple[]
  catalogs: Record<string, CatalogResult>
  now?: Date
}): McpConnectionCheck[] {
  const assignedIDs = assignedOperationIDs(tuples, agent.id)
  const truncated = tuples.length >= RELATION_TUPLE_LIST_LIMIT
  const boundClients = summary.clients.filter((client) => !client.missing)
  return resourceServers.map((server) => {
    const registration: ConnectionCheckGap[] = []
    if (server.state !== 'Active') registration.push('registration_inactive')
    if (!server.client_id) registration.push('registration_no_client')

    const tools = toolRules(server)
    const assigned = tools.filter((rule) =>
      assignedIDs.has(operationResourceID(server, rule.tool.name)),
    )
    const assignment: ConnectionCheckGap[] = []
    if (agent.status !== 'active') assignment.push('assignment_agent_inactive')
    if (boundClients.length === 0) assignment.push('assignment_no_credentials')
    if (tools.length === 0) assignment.push('assignment_no_tools')
    if (tools.length > 0 && assigned.length === 0) {
      // 取得上限に達した材料では、割当がないことを確かめられない。
      assignment.push(truncated ? 'assignment_truncated' : 'assignment_unassigned')
    }

    const tokenLimits: ConnectionCheckGap[] = []
    if (
      assigned.some(
        (rule) => !boundClients.some((client) => coversScopes(client.scopes, rule.scopes)),
      )
    ) {
      tokenLimits.push('token_limits_scope')
    }
    const detailTypes = requiredDetailTypes(assigned)
    tokenLimits.push(
      ...guardrailDeniedGaps('agent', guardrails.agent, server, detailTypes, now),
      ...guardrailDeniedGaps('tenant', guardrails.tenant_default, server, detailTypes, now),
    )

    const catalog = catalogs[server.id]
    const catalogGaps: ConnectionCheckGap[] =
      catalog === 'failed'
        ? ['catalog_fetch_failed']
        : catalog?.received_at
          ? []
          : ['catalog_not_received']

    return { server, items: { registration, assignment, tokenLimits, catalog: catalogGaps } }
  })
}

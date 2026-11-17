import { MCP_OPERATION_TYPE, operationResourceID } from '../../lib/mcpAccess'
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
export type ConnectionCheckGap =
  | 'registration_inactive'
  | 'registration_no_client'
  | 'assignment_agent_inactive'
  | 'assignment_no_credentials'
  | 'assignment_no_tools'
  | 'assignment_unassigned'
  | 'token_limits_scope'
  | 'token_limits_resource'
  | 'token_limits_detail_type'
  | 'token_limits_expired'
  | 'catalog_not_received'
  | 'catalog_fetch_failed'

export type ConnectionCheckItemResult = {
  status: 'ok' | 'gap'
  gaps: ConnectionCheckGap[]
}

export type McpConnectionCheck = {
  serverId: string
  items: Record<ConnectionCheckItem, ConnectionCheckItemResult>
}

function itemResult(gaps: ConnectionCheckGap[]): ConnectionCheckItemResult {
  return { status: gaps.length > 0 ? 'gap' : 'ok', gaps }
}

function ownAgentTuples(tuples: RelationTuple[], agentID: string): RelationTuple[] {
  return tuples.filter(
    (tuple) =>
      tuple.resource_type === MCP_OPERATION_TYPE &&
      tuple.subject_type === 'agent' &&
      tuple.subject_id === agentID &&
      !tuple.subject_relation,
  )
}

function toolRules(server: McpResourceServer): Array<{ rule: ResourceAccessRule; id: string }> {
  return (server.access_rules ?? [])
    .filter((rule) => rule.tool)
    .map((rule) => ({ rule, id: operationResourceID(server, rule.tool!.name) }))
}

// assignedToolRules は、この Agent へ操作の割当 (invoke) があるツール規則を返す。
// 対象の範囲 (op/target、invoke_any_target) は操作の割当そのものではないので含めない。
function assignedToolRules(
  server: McpResourceServer,
  own: RelationTuple[],
): Array<{ rule: ResourceAccessRule; id: string }> {
  const assigned = new Set(
    own.filter((tuple) => tuple.relation === 'invoke').map((tuple) => tuple.resource_id),
  )
  return toolRules(server).filter(({ id }) => assigned.has(id))
}

function coversScopes(clientScopes: string[], required: string[]): boolean {
  return required.every((scope) => clientScopes.includes(scope))
}

// guardrailDeniedGaps は、一つの層のガードレールが、この接続先の発行を deny で拒否する項目を返す。
// require_approval への降格と、接続に関係しない項目 (発行の計数、代行の段数、トークンの有効期間) は対象外である。
function guardrailDeniedGaps(
  policy: AgentGuardrailPolicy | undefined,
  server: McpResourceServer,
  detailTypes: string[],
  now: Date,
): ConnectionCheckGap[] {
  if (!policy) return []
  const gaps = new Set<ConnectionCheckGap>()
  const deny = (action: string | undefined) => (action ?? 'deny') === 'deny'
  if (
    policy.allowed_resources !== undefined &&
    !policy.allowed_resources.includes(server.resource) &&
    deny(policy.on_violation?.allowed_resources)
  ) {
    gaps.add('token_limits_resource')
  }
  if (
    policy.allowed_authorization_detail_types !== undefined &&
    detailTypes.some((type) => !policy.allowed_authorization_detail_types!.includes(type)) &&
    deny(policy.on_violation?.allowed_authorization_detail_types)
  ) {
    gaps.add('token_limits_detail_type')
  }
  if (
    policy.expires_at !== undefined &&
    now.getTime() >= new Date(policy.expires_at).getTime() &&
    deny(policy.on_violation?.expires_at)
  ) {
    gaps.add('token_limits_expired')
  }
  return [...gaps]
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
// catalogs は接続先ごとのカタログで、取得に失敗した接続先は 'failed' を置く。
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
  catalogs: Record<string, ToolCatalog | 'failed'>
  now?: Date
}): McpConnectionCheck[] {
  const own = ownAgentTuples(tuples, agent.id)
  const boundClients = summary.clients.filter((client) => !client.missing)
  return resourceServers.map((server) => {
    const registrationGaps: ConnectionCheckGap[] = []
    if (server.state !== 'Active') registrationGaps.push('registration_inactive')
    if (!server.client_id) registrationGaps.push('registration_no_client')

    const tools = toolRules(server)
    const assigned = assignedToolRules(server, own)
    const assignmentGaps: ConnectionCheckGap[] = []
    if (agent.status !== 'active') assignmentGaps.push('assignment_agent_inactive')
    if (boundClients.length === 0) assignmentGaps.push('assignment_no_credentials')
    if (tools.length === 0) assignmentGaps.push('assignment_no_tools')
    if (tools.length > 0 && assigned.length === 0) assignmentGaps.push('assignment_unassigned')

    const tokenLimitGaps = new Set<ConnectionCheckGap>()
    for (const { rule } of assigned) {
      if (!boundClients.some((client) => coversScopes(client.scopes, rule.scopes))) {
        tokenLimitGaps.add('token_limits_scope')
      }
    }
    const detailTypes = requiredDetailTypes(assigned.map(({ rule }) => rule))
    for (const policy of [guardrails.agent, guardrails.tenant_default]) {
      for (const gap of guardrailDeniedGaps(policy, server, detailTypes, now)) {
        tokenLimitGaps.add(gap)
      }
    }

    const catalog = catalogs[server.id]
    const catalogGaps: ConnectionCheckGap[] =
      catalog === 'failed'
        ? ['catalog_fetch_failed']
        : catalog?.received_at
          ? []
          : ['catalog_not_received']

    return {
      serverId: server.id,
      items: {
        registration: itemResult(registrationGaps),
        assignment: itemResult(assignmentGaps),
        tokenLimits: itemResult([...tokenLimitGaps]),
        catalog: itemResult(catalogGaps),
      },
    }
  })
}

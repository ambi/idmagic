import { tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useLocale } from '../../lib/i18n'
import { MCP_OPERATION_TYPE, operationResourceID } from '../../lib/mcpAccess'
import type {
  AgentAccessSummary,
  AdminAgent,
  AgentGuardrailPolicy,
  AgentGuardrailPolicySet,
  McpResourceServer,
  RelationTuple,
  ToolCatalog,
} from '../../types'
import { agentAccessDictionary } from './AdminAgentAccessPage.i18n'
import {
  CONNECTION_CHECK_ITEMS,
  checkMcpConnections,
  type ConnectionCheckGap,
  type ConnectionCheckItem,
} from './mcpConnectionCheck'
import { isTargetScoped, savedRange } from './targetRange'

export function AdminAgentAccessPage({
  actorUsername,
  agent,
  summary,
  guardrails,
  resourceServers,
  tuples,
  catalogs,
}: {
  actorUsername?: string
  agent: AdminAgent
  summary: AgentAccessSummary
  guardrails: AgentGuardrailPolicySet
  resourceServers: McpResourceServer[]
  tuples: RelationTuple[]
  catalogs: Record<string, ToolCatalog | 'failed'>
}) {
  const t = useDictionary(agentAccessDictionary)
  const { locale } = useLocale()
  const date = (value: string) => new Date(value).toLocaleString(locale)
  const own = tuples.filter(
    (tuple) =>
      tuple.subject_type === 'agent' && tuple.subject_id === agent.id && !tuple.subject_relation,
  )
  const assigned = new Set(
    own
      .filter((tuple) => tuple.resource_type === MCP_OPERATION_TYPE && tuple.relation === 'invoke')
      .map((tuple) => tuple.resource_id),
  )
  const rules = resourceServers.flatMap((server) =>
    (server.access_rules ?? []).map((rule) => ({ server, rule })),
  )
  const checks = checkMcpConnections({
    agent,
    summary,
    guardrails,
    resourceServers,
    tuples,
    catalogs,
  })
  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.title}
      description={agent.name}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={tenantURL(`/admin/agents/${encodeURIComponent(agent.id)}`)} />}
        >
          {t.back}
        </Button>
      }
    >
      <div className="space-y-6">
        <p className="rounded-md border border-slate-200 bg-slate-50 p-4 text-sm">{t.help}</p>
        {tuples.length >= 1000 ? <Alert variant="destructive">{t.truncated}</Alert> : null}
        <Card className="space-y-3 p-6">
          <dl className="grid gap-3 sm:grid-cols-2">
            <div>
              <dt className="font-semibold">{t.asOf}</dt>
              <dd>{date(summary.as_of)}</dd>
            </div>
            <div>
              <dt className="font-semibold">{t.lastIssued}</dt>
              <dd>
                {summary.last_token_issued_at ? date(summary.last_token_issued_at) : t.noRecord}
              </dd>
            </div>
          </dl>
        </Card>
        <Card className="space-y-3 p-6">
          <h2 className="text-lg font-semibold">{t.clients}</h2>
          {summary.clients.length === 0 ? (
            <p>{t.noClients}</p>
          ) : (
            <ul className="space-y-3">
              {summary.clients.map((client) => (
                <li key={client.client_id}>
                  <a
                    className="underline"
                    href={tenantURL(`/admin/clients/${encodeURIComponent(client.client_id)}`)}
                  >
                    {client.name || client.client_id}
                  </a>
                  {client.name ? <p className="text-sm">{client.client_id}</p> : null}
                  {client.missing ? (
                    <p>{t.missing}</p>
                  ) : (
                    <p>{client.scopes.join(', ') || t.noValues}</p>
                  )}
                </li>
              ))}
            </ul>
          )}
        </Card>
        <Card className="space-y-3 overflow-x-auto p-6">
          <h2 className="text-lg font-semibold">{t.delegations}</h2>
          {summary.delegations.length === 0 ? (
            <p>{t.noDelegations}</p>
          ) : (
            <table className="w-full text-left text-sm">
              <thead>
                <tr>
                  {[t.user, t.scopes, t.state, t.expires].map((label) => (
                    <th key={label} className="p-2">
                      {label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {summary.delegations.map((grant) => (
                  <tr key={grant.user_id} className="border-t">
                    <td className="p-2">
                      <a
                        className="underline"
                        href={tenantURL(`/admin/users/${encodeURIComponent(grant.user_id)}`)}
                      >
                        {grant.user_id}
                      </a>
                    </td>
                    <td className="p-2">{grant.scopes.join(', ')}</td>
                    <td className="p-2">
                      {grant.state === 'revoked' ? t.revoked : grant.active ? t.active : t.expired}
                    </td>
                    <td className="p-2">{date(grant.expires_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
        <Card className="space-y-3 overflow-x-auto p-6">
          <h2 className="text-lg font-semibold">{t.rules}</h2>
          {rules.length === 0 ? (
            <p>{t.noRules}</p>
          ) : (
            <table className="w-full text-left text-sm">
              <thead>
                <tr>
                  {[t.server, t.operation, t.scopes, t.target, t.conditions, t.limits].map(
                    (label) => (
                      <th key={label} className="p-2">
                        {label}
                      </th>
                    ),
                  )}
                </tr>
              </thead>
              <tbody>
                {rules.map(({ server, rule }) => {
                  const id = rule.tool ? operationResourceID(server, rule.tool.name) : ''
                  const range = savedRange(own, id, agent.id)
                  return (
                    <tr
                      key={`${server.id}/${rule.tool?.name ?? `${rule.resource_type}#${rule.relation}`}`}
                      className="border-t align-top"
                    >
                      <td className="p-2">
                        <a
                          className="underline"
                          href={tenantURL(
                            `/admin/mcp-resource-servers/${encodeURIComponent(server.id)}/edit`,
                          )}
                        >
                          {server.name}
                        </a>
                        <p>{server.resource}</p>
                        <p>{server.state}</p>
                      </td>
                      <td className="p-2">
                        <p>{rule.tool?.display_name ?? `${rule.resource_type}#${rule.relation}`}</p>
                        <p>
                          {rule.tool ? (assigned.has(id) ? t.assigned : t.unassigned) : t.relation}
                        </p>
                      </td>
                      <td className="p-2">{rule.scopes.join(', ')}</td>
                      <td className="whitespace-pre-line p-2">
                        {isTargetScoped(rule)
                          ? range.mode === 'any'
                            ? t.anyTarget
                            : range.mode === 'specific'
                              ? range.targets
                              : t.noTargets
                          : rule.resource_type === MCP_OPERATION_TYPE
                            ? t.service
                            : `${t.relation}: ${rule.resource_type}#${rule.relation}`}
                      </td>
                      <td className="space-y-2 p-2">
                        {rule.condition?.deny_when ? (
                          <div>
                            {t.denyWhen}
                            <code className="block whitespace-pre-wrap">
                              {rule.condition.deny_when}
                            </code>
                          </div>
                        ) : null}
                        {rule.condition?.require_approval_when ? (
                          <div>
                            {t.approvalWhen}
                            <code className="block whitespace-pre-wrap">
                              {rule.condition.require_approval_when}
                            </code>
                            <p>{rule.condition.approval_detail_type}</p>
                          </div>
                        ) : null}
                        {rule.requires_approved_operation ? (
                          <p>
                            {t.approval}: {rule.authorization_detail_type}
                          </p>
                        ) : null}
                        {!rule.condition && !rule.requires_approved_operation
                          ? t.noConditions
                          : null}
                      </td>
                      <td className="p-2">
                        {rule.limits?.length
                          ? rule.limits.map((limit) => (
                              <p
                                key={`${limit.window_seconds}/${[...(limit.by ?? [])].sort().join(',')}`}
                              >
                                {t.limit
                                  .replace(
                                    '{by}',
                                    (limit.by ?? [])
                                      .map((by) =>
                                        by === 'agent'
                                          ? t.agent
                                          : by === 'user'
                                            ? t.user
                                            : t.resource,
                                      )
                                      .join(', ') || t.global,
                                  )
                                  .replace('{seconds}', String(limit.window_seconds))
                                  .replace('{count}', String(limit.max_requests))}
                              </p>
                            ))
                          : t.noLimits}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </Card>
        <McpConnectionCheckCard agent={agent} checks={checks} resourceServers={resourceServers} />
        <Card className="space-y-4 p-6">
          <h2 className="text-lg font-semibold">{t.guardrails}</h2>
          <GuardrailPolicy title={t.agentPolicy} policy={guardrails.agent} />
          <GuardrailPolicy title={t.tenantPolicy} policy={guardrails.tenant_default} />
        </Card>
      </div>
    </AdminShell>
  )
}

// McpConnectionCheckCard は接続先ごとの確認項目と不足の案内を示す (REQ-OAUTH2-094)。
// 判定は mcpConnectionCheck が保存値から行い、ここは結果の表示だけを担う。
function McpConnectionCheckCard({
  agent,
  checks,
  resourceServers,
}: {
  agent: AdminAgent
  checks: ReturnType<typeof checkMcpConnections>
  resourceServers: McpResourceServer[]
}) {
  const t = useDictionary(agentAccessDictionary)
  const itemLabels: Record<ConnectionCheckItem, string> = {
    registration: t.itemRegistration,
    assignment: t.itemAssignment,
    tokenLimits: t.itemTokenLimits,
    catalog: t.itemCatalog,
  }
  const gapTexts: Record<ConnectionCheckGap, string> = {
    registration_inactive: t.gapRegistrationInactive,
    registration_no_client: t.gapRegistrationNoClient,
    assignment_agent_inactive: t.gapAssignmentAgentInactive,
    assignment_no_credentials: t.gapAssignmentNoCredentials,
    assignment_no_tools: t.gapAssignmentNoTools,
    assignment_unassigned: t.gapAssignmentUnassigned,
    token_limits_scope: t.gapTokenLimitsScope,
    token_limits_resource: t.gapTokenLimitsResource,
    token_limits_detail_type: t.gapTokenLimitsDetailType,
    token_limits_expired: t.gapTokenLimitsExpired,
    catalog_not_received: t.gapCatalogNotReceived,
    catalog_fetch_failed: t.gapCatalogFetchFailed,
  }
  const gapHref = (gap: ConnectionCheckGap, serverID: string): string | undefined => {
    const serverEdit = `/admin/mcp-resource-servers/${encodeURIComponent(serverID)}/edit`
    switch (gap) {
      case 'registration_inactive':
      case 'registration_no_client':
      case 'assignment_no_tools':
      case 'catalog_not_received':
        return serverEdit
      case 'assignment_agent_inactive':
        return `/admin/agents/${encodeURIComponent(agent.id)}/edit`
      case 'assignment_no_credentials':
      case 'token_limits_scope':
        return `/admin/agents/${encodeURIComponent(agent.id)}/credentials`
      case 'assignment_unassigned':
        return `/admin/agents/${encodeURIComponent(agent.id)}/resources`
      case 'token_limits_resource':
      case 'token_limits_detail_type':
      case 'token_limits_expired':
        return `/admin/agents/${encodeURIComponent(agent.id)}/guardrail/edit`
      case 'catalog_fetch_failed':
        return undefined
    }
  }
  return (
    <Card className="space-y-3 p-6">
      <h2 className="text-lg font-semibold">{t.connectionCheck}</h2>
      <p className="text-sm text-slate-600">{t.connectionCheckHelp}</p>
      {checks.length === 0 ? (
        <p>{t.noConnections}</p>
      ) : (
        checks.map((check) => {
          const server = resourceServers.find((candidate) => candidate.id === check.serverId)
          return (
            <section
              key={check.serverId}
              className="space-y-2 border-t pt-3 first:border-t-0 first:pt-0"
            >
              <h3 className="font-semibold">
                {server ? (
                  <a
                    className="underline"
                    href={tenantURL(
                      `/admin/mcp-resource-servers/${encodeURIComponent(server.id)}/edit`,
                    )}
                  >
                    {server.name}
                  </a>
                ) : (
                  check.serverId
                )}
              </h3>
              <ul className="space-y-2">
                {CONNECTION_CHECK_ITEMS.map((item) => {
                  const result = check.items[item]
                  return (
                    <li key={item} className="text-sm">
                      <p>
                        <span className="font-medium">{itemLabels[item]}</span>{' '}
                        {result.status === 'ok' ? t.checkOk : t.checkGap}
                      </p>
                      {result.gaps.map((gap) => {
                        const href = gapHref(gap, check.serverId)
                        return href ? (
                          <p key={gap}>
                            <a className="underline" href={tenantURL(href)}>
                              {gapTexts[gap]}
                            </a>
                          </p>
                        ) : (
                          <p key={gap}>{gapTexts[gap]}</p>
                        )
                      })}
                    </li>
                  )
                })}
              </ul>
            </section>
          )
        })
      )}
    </Card>
  )
}

function GuardrailPolicy({ title, policy }: { title: string; policy?: AgentGuardrailPolicy }) {
  const t = useDictionary(agentAccessDictionary)
  const { locale } = useLocale()
  return (
    <section className="space-y-2">
      <h3 className="font-semibold">{title}</h3>
      {!policy ? (
        <p>{t.noPolicy}</p>
      ) : (
        <dl className="grid gap-2 sm:grid-cols-2">
          {(['allowed_resources', 'allowed_authorization_detail_types'] as const).map((key) =>
            policy[key] === undefined ? null : (
              <div key={key}>
                <dt>{t[key]}</dt>
                <dd>{policy[key]!.join(', ') || t.noValues}</dd>
              </div>
            ),
          )}
          {(['max_token_ttl_seconds', 'max_delegation_depth'] as const).map((key) =>
            policy[key] === undefined ? null : (
              <div key={key}>
                <dt>{t[key]}</dt>
                <dd>{policy[key]}</dd>
              </div>
            ),
          )}
          {policy.issuance_rate ? (
            <div>
              <dt>{t.issuance_rate}</dt>
              <dd>
                {t.issuanceRate
                  .replace('{count}', String(policy.issuance_rate.max_issuances))
                  .replace('{seconds}', String(policy.issuance_rate.window_seconds))}
              </dd>
            </div>
          ) : null}
          {policy.expires_at ? (
            <div>
              <dt>{t.expires_at}</dt>
              <dd>{new Date(policy.expires_at).toLocaleString(locale)}</dd>
            </div>
          ) : null}
          <div>
            <dt>{t.require_approval}</dt>
            <dd>{policy.require_approval ? t.yes : t.no}</dd>
          </div>
          {Object.entries(policy.on_violation ?? {}).map(([key, value]) => (
            <div key={key}>
              <dt>
                {t.on_violation}:{' '}
                {t[key as keyof NonNullable<AgentGuardrailPolicy['on_violation']>]}
              </dt>
              <dd>{value === 'deny' ? t.deny : t.require_approval_action}</dd>
            </div>
          ))}
        </dl>
      )}
    </section>
  )
}

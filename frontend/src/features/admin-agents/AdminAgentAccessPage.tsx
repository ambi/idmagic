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
} from '../../types'
import { agentAccessDictionary } from './AdminAgentAccessPage.i18n'
import { isTargetScoped, savedRange } from './targetRange'

export function AdminAgentAccessPage({
  actorUsername,
  agent,
  summary,
  guardrails,
  resourceServers,
  tuples,
}: {
  actorUsername?: string
  agent: AdminAgent
  summary: AgentAccessSummary
  guardrails: AgentGuardrailPolicySet
  resourceServers: McpResourceServer[]
  tuples: RelationTuple[]
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
        <Card className="space-y-4 p-6">
          <h2 className="text-lg font-semibold">{t.guardrails}</h2>
          <GuardrailPolicy title={t.agentPolicy} policy={guardrails.agent} />
          <GuardrailPolicy title={t.tenantPolicy} policy={guardrails.tenant_default} />
        </Card>
      </div>
    </AdminShell>
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

import { useState } from 'react'
import { tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { MCP_OPERATION_TYPE, operationResourceID } from '../../lib/mcpAccess'
import type { AdminAgent, McpResourceServer, RelationTuple } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'

export function AdminMcpAgentAccessPage({
  actorUsername,
  server,
  agents,
  tuples,
}: {
  actorUsername?: string
  server: McpResourceServer
  agents: AdminAgent[]
  tuples: RelationTuple[]
}) {
  const t = useDictionary(adminMcpResourceServersDictionary)
  const [agentID, setAgentID] = useState(
    agents.find((agent) => agent.status === 'active')?.id ?? '',
  )
  const byAgent = new Map<string, Set<string>>()
  for (const tuple of tuples) {
    if (
      tuple.resource_type !== MCP_OPERATION_TYPE ||
      tuple.relation !== 'invoke' ||
      tuple.subject_type !== 'agent' ||
      tuple.subject_relation
    )
      continue
    const resources = byAgent.get(tuple.subject_id) ?? new Set<string>()
    resources.add(tuple.resource_id)
    byAgent.set(tuple.subject_id, resources)
  }
  const assigned = agents
    .map((agent) => ({
      agent,
      operations: (server.access_rules ?? []).filter(
        (rule) =>
          rule.tool && byAgent.get(agent.id)?.has(operationResourceID(server, rule.tool.name)),
      ),
    }))
    .filter((item) => item.operations.length > 0)
  const editURL = (id: string) =>
    tenantURL(
      `/admin/agents/${encodeURIComponent(id)}/resources/edit?server=${encodeURIComponent(server.id)}`,
    )
  return (
    <AdminShell
      active="mcp-resource-servers"
      actorUsername={actorUsername}
      title={t.assignments}
      description={server.name}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={tenantURL('/admin/mcp-resource-servers')} />}
        >
          {t.backToList}
        </Button>
      }
    >
      <Card className="space-y-5 p-6">
        {tuples.length >= 1000 || agents.length >= 1000 ? (
          <Alert variant="destructive">{t.assignmentsTruncated}</Alert>
        ) : null}
        <div className="flex items-end gap-3">
          <div className="space-y-2">
            <Label htmlFor="assignment-agent">{t.selectAgent}</Label>
            <select
              id="assignment-agent"
              className="block rounded-md border p-2"
              value={agentID}
              onChange={(event) => setAgentID(event.target.value)}
            >
              <option value="">—</option>
              {agents
                .filter((agent) => agent.status === 'active')
                .map((agent) => (
                  <option key={agent.id} value={agent.id}>
                    {agent.name}
                  </option>
                ))}
            </select>
          </div>
          {agentID ? (
            <Button nativeButton={false} render={<a href={editURL(agentID)} />}>
              {t.assignmentEdit}
            </Button>
          ) : null}
        </div>
        {assigned.length === 0 ? (
          <p>{t.noAssignments}</p>
        ) : (
          <table className="w-full text-left text-sm">
            <thead>
              <tr>
                <th className="p-2">{t.assignments}</th>
                <th className="p-2">{t.operationsHeading}</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {assigned.map(({ agent, operations }) => (
                <tr key={agent.id} className="border-t">
                  <td className="p-2">{agent.name}</td>
                  <td className="p-2">
                    {operations.map((rule) => rule.tool!.display_name).join(', ')}
                  </td>
                  <td className="p-2">
                    <a className="underline" href={editURL(agent.id)}>
                      {t.assignmentEdit}
                    </a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </AdminShell>
  )
}

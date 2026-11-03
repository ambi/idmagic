import { type FormEvent, useRef, useState } from 'react'
import { tenantURL, writeRelationTuples } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { MCP_OPERATION_TYPE, operationResourceID, operationTuple } from '../../lib/mcpAccess'
import type { AdminAgent, McpResourceServer, RelationTuple } from '../../types'
import { agentResourcesDictionary } from './agentResources.i18n'
import { returnToAgent } from './agentNavigation'

export function AdminAgentResourcesPage({
  csrfToken,
  actorUsername,
  agent,
  resourceServers,
  tuples,
  edit = false,
  selectedServer = '',
  onSaved,
}: {
  csrfToken: string
  actorUsername?: string
  agent: AdminAgent
  resourceServers: McpResourceServer[]
  tuples: RelationTuple[]
  edit?: boolean
  selectedServer?: string
  onSaved?: () => void
}) {
  const t = useDictionary(agentResourcesDictionary)
  const base = tenantURL(`/admin/agents/${encodeURIComponent(agent.id)}/resources`)
  const [saved, setSaved] = useState(tuples)
  const assigned = new Set(
    saved
      .filter(
        (tuple) =>
          tuple.resource_type === MCP_OPERATION_TYPE &&
          tuple.relation === 'invoke' &&
          tuple.subject_type === 'agent' &&
          tuple.subject_id === agent.id &&
          !tuple.subject_relation,
      )
      .map((tuple) => tuple.resource_id),
  )
  const servers = resourceServers.filter((server) =>
    (server.access_rules ?? []).some((rule) => rule.tool),
  )
  const [serverID, setServerID] = useState(
    selectedServer ||
      servers.find((server) => server.state === 'Active')?.id ||
      servers[0]?.id ||
      '',
  )
  const [selected, setSelected] = useState(new Set(assigned))
  const [busy, setBusy] = useState(false)
  const running = useRef(false)
  const [error, setError] = useState('')
  const truncated = tuples.length >= 1000
  const server = servers.find((item) => item.id === serverID)
  const rules = server?.access_rules?.filter((rule) => rule.tool) ?? []
  const added = rules.filter(
    (rule) =>
      !assigned.has(operationResourceID(server!, rule.tool!.name)) &&
      selected.has(operationResourceID(server!, rule.tool!.name)),
  )
  const removed = rules.filter(
    (rule) =>
      assigned.has(operationResourceID(server!, rule.tool!.name)) &&
      !selected.has(operationResourceID(server!, rule.tool!.name)),
  )
  const granted = servers.flatMap((item) =>
    (item.access_rules ?? [])
      .filter((rule) => rule.tool && assigned.has(operationResourceID(item, rule.tool.name)))
      .map((rule) => ({ server: item, rule })),
  )
  async function save(event: FormEvent) {
    event.preventDefault()
    if (!server || truncated || running.current || (!added.length && !removed.length)) return
    if (server.state !== 'Active' && added.length) return
    running.current = true
    setBusy(true)
    setError('')
    try {
      const writes = added.map((rule) => operationTuple(server, rule.tool!.name, agent.id))
      const deletes = removed.map((rule) => operationTuple(server, rule.tool!.name, agent.id))
      await writeRelationTuples(csrfToken, { writes, deletes })
      const deleted = new Set(deletes.map((tuple) => tuple.resource_id))
      setSaved([...saved.filter((tuple) => !deleted.has(tuple.resource_id)), ...writes])
      if (onSaved) onSaved()
      else returnToAgent(agent.id, t.saved)
    } catch {
      setError(t.failed)
    } finally {
      running.current = false
      setBusy(false)
    }
  }
  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={edit ? t.edit : t.title}
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
      <Card className="space-y-5 p-6">
        <p className="text-sm text-slate-600">{t.help}</p>
        {truncated ? <Alert variant="destructive">{t.truncated}</Alert> : null}
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        {!edit ? (
          <>
            <Button nativeButton={false} render={<a href={`${base}/edit`} />}>
              {t.edit}
            </Button>
            {granted.length === 0 ? (
              <p>{t.noAccess}</p>
            ) : (
              <table className="w-full text-left text-sm">
                <thead>
                  <tr>
                    {[t.server, t.operation, t.target, t.approval].map((label) => (
                      <th key={label} className="p-2">
                        {label}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {granted.map(({ server: item, rule }) => (
                    <tr key={operationResourceID(item, rule.tool!.name)} className="border-t">
                      <td className="p-2">
                        {item.name}
                        {item.state !== 'Active' ? ` (${t.disabled})` : ''}
                      </td>
                      <td className="p-2">{rule.tool!.display_name}</td>
                      <td className="p-2">
                        {rule.resource_type === MCP_OPERATION_TYPE
                          ? t.targetService
                          : t.targetRequired}
                      </td>
                      <td className="p-2">
                        {rule.requires_approved_operation
                          ? t.approvalRequired
                          : rule.condition?.require_approval_when
                            ? t.approvalConditional
                            : t.approvalNone}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        ) : (
          <form onSubmit={(event) => void save(event)} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="resource-connection">{t.server}</Label>
              <select
                id="resource-connection"
                className="w-full rounded-md border p-2"
                value={serverID}
                disabled={busy}
                onChange={(event) => {
                  setServerID(event.target.value)
                  setSelected(new Set(assigned))
                  setError('')
                }}
              >
                {servers.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                    {item.state !== 'Active' ? ` (${t.disabled})` : ''}
                  </option>
                ))}
              </select>
            </div>
            {!rules.length ? (
              <p>{t.noTools}</p>
            ) : (
              <fieldset className="space-y-3">
                <legend className="mb-2 font-semibold">{t.operation}</legend>
                {rules.map((rule) => {
                  const id = operationResourceID(server!, rule.tool!.name)
                  return (
                    <label key={id} className="flex items-start gap-3 rounded-md border p-3">
                      <input
                        type="checkbox"
                        checked={selected.has(id)}
                        disabled={
                          busy ||
                          truncated ||
                          agent.status === 'killed' ||
                          (server?.state !== 'Active' && !assigned.has(id))
                        }
                        onChange={(event) =>
                          setSelected((current) => {
                            const next = new Set(current)
                            if (event.target.checked) next.add(id)
                            else next.delete(id)
                            return next
                          })
                        }
                      />
                      <span>
                        <span className="font-medium">{rule.tool!.display_name}</span>
                        <span className="block text-xs text-slate-600">
                          {rule.resource_type === MCP_OPERATION_TYPE
                            ? t.targetService
                            : t.targetRequired}{' '}
                          /{' '}
                          {rule.requires_approved_operation
                            ? t.approvalRequired
                            : rule.condition?.require_approval_when
                              ? t.approvalConditional
                              : t.approvalNone}
                        </span>
                      </span>
                    </label>
                  )
                })}
              </fieldset>
            )}
            <div className="space-y-1 text-sm" aria-live="polite">
              <p>
                {t.additions}: {added.map((rule) => rule.tool!.display_name).join(', ') || '—'}
              </p>
              <p>
                {t.removals}: {removed.map((rule) => rule.tool!.display_name).join(', ') || '—'}
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button
                type="button"
                variant="outline"
                nativeButton={false}
                render={<a href={base} />}
              >
                {t.cancel}
              </Button>
              <Button
                type="submit"
                disabled={
                  busy ||
                  truncated ||
                  (!added.length && !removed.length) ||
                  agent.status === 'killed'
                }
              >
                {t.save}
              </Button>
            </div>
          </form>
        )}
        <div className="flex gap-4 text-sm">
          <a
            className="underline"
            href={tenantURL(`/admin/authorization/access?agent=${encodeURIComponent(agent.id)}`)}
          >
            {t.advanced}
          </a>
          <a className="underline" href={tenantURL('/admin/mcp-resource-servers')}>
            {t.setup}
          </a>
          <a
            className="underline"
            href={tenantURL(`/admin/agents/${encodeURIComponent(agent.id)}/credentials`)}
          >
            {t.credentials}
          </a>
        </div>
      </Card>
    </AdminShell>
  )
}

import { type FormEvent, useRef, useState } from 'react'
import { tenantURL } from '../../api'
import { RELATION_TUPLE_LIST_LIMIT, writeRelationTuples } from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type {
  AdminAgent,
  McpResourceServer,
  RelationTuple,
  ResourceTypeDefinition,
} from '../../types'
import { authorizationErrorMessage, selectClassName } from './AdminAuthorizationShared'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'
import { agentAccessDictionary } from './agentAccess.i18n'

type Props = {
  csrfToken: string
  actorUsername?: string
  agents: AdminAgent[]
  resourceServers: McpResourceServer[]
  resourceTypes: ResourceTypeDefinition[]
  tuples: RelationTuple[]
  mode: 'list' | 'create'
  selectedServer?: string
  selectedAgent?: string
  justGranted?: boolean
  onGranted?: (selection: { server: string; agent: string }) => void
}

export function AdminAgentAccessPage({
  csrfToken,
  actorUsername,
  agents,
  resourceServers,
  resourceTypes,
  tuples: initialTuples,
  mode,
  selectedServer = '',
  selectedAgent = '',
  justGranted = false,
  onGranted,
}: Props) {
  const t = useDictionary(agentAccessDictionary)
  const authorization = useDictionary(adminAuthorizationDictionary)
  const [serverID, setServerID] = useState(selectedServer)
  const [permission, setPermission] = useState('')
  const [resourceID, setResourceID] = useState('')
  const [agentID, setAgentID] = useState(selectedAgent)
  const [tuples, setTuples] = useState(initialTuples)
  const [filter, setFilter] = useState('')
  const [busy, setBusy] = useState(false)
  const running = useRef(false)
  const [pending, setPending] = useState<RelationTuple | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState(justGranted ? t.granted : '')
  const server = resourceServers.find((item) => item.id === serverID && item.state === 'Active')
  const options = [
    ...new Map(
      (server?.access_rules ?? [])
        .filter((rule) =>
          resourceTypes
            .find((type) => type.name === rule.resource_type)
            ?.relations.find((relation) => relation.name === rule.relation)
            ?.rewrites.some(
              (rewrite) =>
                rewrite.kind === 'direct' && rewrite.direct_subject_types?.includes('agent'),
            ),
        )
        .map((rule) => [`${rule.resource_type}:${rule.relation}`, rule]),
    ).values(),
  ]
  const rule = options.find((item) => `${item.resource_type}:${item.relation}` === permission)
  const matchingRules = (server?.access_rules ?? []).filter(
    (item) => item.resource_type === rule?.resource_type && item.relation === rule?.relation,
  )
  const agent = agents.find((item) => item.id === agentID && item.status === 'active')
  const suggestions = [
    ...new Set(
      tuples
        .filter((tuple) => tuple.resource_type === rule?.resource_type)
        .map((tuple) => tuple.resource_id),
    ),
  ]
  const agentName = (id: string) => agents.find((item) => item.id === id)?.name ?? id
  const direct = tuples.filter(
    (tuple) =>
      tuple.subject_type === 'agent' && !tuple.subject_relation && tuple.subject_id !== '*',
  )
  const displayed = direct.filter(
    (tuple) =>
      (!selectedAgent || tuple.subject_id === selectedAgent) &&
      (!selectedServer ||
        resourceServers
          .find((item) => item.id === selectedServer)
          ?.access_rules?.some(
            (item) =>
              item.resource_type === tuple.resource_type && item.relation === tuple.relation,
          )) &&
      `${tuple.resource_type} ${tuple.resource_id} ${tuple.relation} ${agentName(tuple.subject_id)}`
        .toLocaleLowerCase()
        .includes(filter.toLocaleLowerCase()),
  )

  async function save(event: FormEvent) {
    event.preventDefault()
    if (running.current || !rule || !agent || !resourceID) return
    running.current = true
    setBusy(true)
    setError('')
    try {
      await writeRelationTuples(csrfToken, {
        writes: [
          {
            resource_type: rule.resource_type,
            resource_id: resourceID,
            relation: rule.relation,
            subject_type: 'agent',
            subject_id: agent.id,
          },
        ],
      })
      onGranted?.({ server: serverID, agent: agent.id })
    } catch (cause) {
      setError(authorizationErrorMessage(cause, authorization))
    } finally {
      running.current = false
      setBusy(false)
    }
  }

  async function revoke() {
    if (!pending || running.current) return
    running.current = true
    setBusy(true)
    setError('')
    try {
      await writeRelationTuples(csrfToken, { deletes: [pending] })
      setTuples((current) => current.filter((tuple) => tuple !== pending))
      setPending(null)
      setNotice(t.revoked)
    } catch (cause) {
      setError(authorizationErrorMessage(cause, authorization))
    } finally {
      running.current = false
      setBusy(false)
    }
  }

  const select = (
    id: string,
    label: string,
    value: string,
    items: { value: string; label: string }[],
    change: (value: string) => void,
  ) => (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id} required>
        {label}
      </Label>
      <select
        id={id}
        required
        disabled={busy}
        className={selectClassName}
        value={value}
        onChange={(event) => change(event.target.value)}
      >
        <option value="">{t.choose}</option>
        {items.map((item) => (
          <option key={item.value} value={item.value}>
            {item.label}
          </option>
        ))}
      </select>
    </div>
  )

  return (
    <AdminShell
      active="authorization"
      actorUsername={actorUsername}
      title={mode === 'create' ? t.add : t.title}
      description={t.description}
      actions={
        <Button
          nativeButton={false}
          variant={mode === 'create' ? 'outline' : 'default'}
          render={
            <a
              href={tenantURL(
                mode === 'create'
                  ? '/admin/authorization/access'
                  : `/admin/authorization/access/new?${new URLSearchParams({ server: selectedServer, agent: selectedAgent })}`,
              )}
            />
          }
        >
          {mode === 'create' ? t.cancel : t.add}
        </Button>
      }
    >
      <Toast message={notice} onDismiss={() => setNotice('')} />
      <p className="text-sm text-slate-600">{t.limits}</p>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {initialTuples.length >= RELATION_TUPLE_LIST_LIMIT ? (
        <Alert variant="destructive">{t.truncated}</Alert>
      ) : null}
      {mode === 'create' ? (
        <Card className="admin-form p-5">
          <form className="flex flex-col gap-4" onSubmit={save}>
            {select(
              'access-connection',
              t.connection,
              serverID,
              resourceServers
                .filter((item) => item.state === 'Active')
                .map((item) => ({ value: item.id, label: item.name })),
              (value) => {
                setServerID(value)
                setPermission('')
                setResourceID('')
              },
            )}
            {server ? <p className="break-all text-xs text-slate-500">{server.resource}</p> : null}
            {server && options.length === 0 ? (
              <p className="text-sm text-slate-600">{t.noPermission}</p>
            ) : null}
            {select(
              'access-permission',
              t.permission,
              permission,
              options.map((item) => ({
                value: `${item.resource_type}:${item.relation}`,
                label: `${item.relation} (${item.resource_type})`,
              })),
              (value) => {
                setPermission(value)
                setResourceID('')
              },
            )}
            {rule ? (
              <>
                <p className="text-sm text-slate-600">
                  {t.scopeHelp}:{' '}
                  {[...new Set(matchingRules.flatMap((item) => item.scopes))].join(', ')}
                </p>
                {(server?.access_rules ?? []).some(
                  (item) =>
                    item.resource_type === rule.resource_type &&
                    item.relation === rule.relation &&
                    item.requires_approved_operation,
                ) ? (
                  <p className="text-sm text-slate-600">{t.approval}</p>
                ) : null}
                {(server?.access_rules ?? []).some(
                  (item) =>
                    item.resource_type === rule.resource_type &&
                    item.relation === rule.relation &&
                    (item.condition || item.limits?.length || item.authorization_detail_type),
                ) ? (
                  <p className="text-sm text-slate-600">{t.conditions}</p>
                ) : null}
              </>
            ) : null}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="access-resource" required>
                {t.resource}
              </Label>
              <Input
                id="access-resource"
                required
                disabled={busy}
                list="access-resources"
                aria-describedby="access-resource-help"
                value={resourceID}
                onChange={(event) => setResourceID(event.target.value)}
              />
              <datalist id="access-resources">
                {suggestions.map((id) => (
                  <option key={id} value={id} />
                ))}
              </datalist>
              <p id="access-resource-help" className="text-xs text-slate-500">
                {t.resourceHelp}
              </p>
            </div>
            {select(
              'access-agent',
              t.agent,
              agentID,
              agents
                .filter((item) => item.status === 'active')
                .map((item) => ({ value: item.id, label: item.name })),
              setAgentID,
            )}
            {agents.length >= 1000 ? (
              <p className="text-xs text-slate-500">{t.agentLimit}</p>
            ) : null}
            {rule && resourceID && agent ? (
              <section aria-label={t.preview} className="rounded-md border p-3 text-sm">
                <h2 className="font-semibold">{t.preview}</h2>
                <p>
                  {agent.name} → {rule.resource_type}:{resourceID} → {rule.relation}
                </p>
              </section>
            ) : null}
            <p className="text-sm text-slate-600">{t.shared}</p>
            <Button type="submit" disabled={busy || !rule || !agent || !resourceID}>
              {busy ? t.granting : t.grant}
            </Button>
          </form>
        </Card>
      ) : (
        <>
          <Label htmlFor="access-search">{t.filter}</Label>
          <Input
            id="access-search"
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
          />
          <h2 className="text-base font-semibold">
            {t.results} ({displayed.length})
          </h2>
          {displayed.length === 0 ? (
            <Card className="p-5 text-sm text-slate-600">{t.empty}</Card>
          ) : (
            <Card className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr>
                    {[t.resource, t.permission, t.agent, t.revoke].map((label) => (
                      <th key={label} className="p-3">
                        {label}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {displayed.map((tuple) => (
                    <tr key={JSON.stringify(tuple)} className="border-t">
                      <td className="p-3">
                        <span>{tuple.resource_id}</span>
                        <p className="text-xs text-slate-500">{tuple.resource_type}</p>
                      </td>
                      <td className="p-3">{tuple.relation}</td>
                      <td className="p-3">{agentName(tuple.subject_id)}</td>
                      <td className="p-3">
                        <Button
                          variant="destructive"
                          disabled={busy}
                          onClick={() => {
                            setError('')
                            setPending(tuple)
                          }}
                        >
                          {t.revoke}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
          )}
        </>
      )}
      <details className="rounded-md border p-4 text-sm">
        <summary className="cursor-pointer font-medium">{t.developer}</summary>
        <div className="mt-3 flex flex-wrap gap-4">
          <a className="underline" href={tenantURL('/admin/authorization/model')}>
            {t.model}
          </a>
          <a className="underline" href={tenantURL('/admin/authorization/relation-tuples')}>
            {t.tuples}
          </a>
          <a className="underline" href={tenantURL('/admin/authorization/check')}>
            {t.check}
          </a>
        </div>
      </details>
      {pending ? (
        <ConfirmActionDialog
          title={t.revokeTitle}
          target={`${pending.resource_type}:${pending.resource_id} / ${pending.relation} / ${agentName(pending.subject_id)}`}
          description={t.revokeResult}
          confirmLabel={t.revoke}
          busy={busy}
          error={error}
          onClose={() => setPending(null)}
          onConfirm={() => void revoke()}
        />
      ) : null}
    </AdminShell>
  )
}

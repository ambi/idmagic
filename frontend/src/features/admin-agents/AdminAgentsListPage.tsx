import { IconPlus, IconRefresh, IconRobot, IconX } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, listAdminAgents, registerAdminAgent, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { AdminAgent } from '../../types'
import { AgentDetailCard } from './AdminAgentDetailCard'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { kindLabel, optionalValue, parseRoles, StatusBadge } from './AdminAgentsShared'

export function AdminAgentsPage({
  csrfToken,
  actorUsername,
  agents: initial,
}: {
  csrfToken: string
  actorUsername?: string
  agents: AdminAgent[]
}) {
  const [agents, setAgents] = useState(initial)
  const initialID = new URLSearchParams(window.location.search).get('agent')
  const [selectedID, setSelectedID] = useState<string>(
    () => initial.find((a) => a.id === initialID)?.id ?? initial[0]?.id ?? '',
  )
  const [showCreate, setShowCreate] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminAgentsDictionary)

  const selected = agents.find((a) => a.id === selectedID) ?? null

  async function refresh(preferredID = selectedID) {
    const next = await listAdminAgents()
    setAgents(next)
    setSelectedID(next.find((a) => a.id === preferredID)?.id ?? next[0]?.id ?? '')
  }

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(success)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setBusy(false)
    }
  }

  async function handleCreate(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = e.currentTarget
    const data = new FormData(form)
    await run(async () => {
      const created = await registerAdminAgent(csrfToken, {
        name: String(data.get('name') ?? ''),
        description: optionalValue(data.get('description')),
        kind: (String(data.get('kind') ?? 'autonomous') as AdminAgent['kind']) || undefined,
        owner_user_id: optionalValue(data.get('owner_user_id')),
        roles: parseRoles(String(data.get('roles') ?? '')),
      })
      form.reset()
      setShowCreate(false)
      await refresh(created.id)
    }, t.agentRegisteredNotice)
  }

  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <>
          <Button
            variant="outline"
            className="size-9 px-0"
            aria-label={t.reloadAriaLabel}
            onClick={() => run(() => refresh(), t.listRefreshedNotice)}
            disabled={busy}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
          <Button onClick={() => setShowCreate(true)} disabled={busy}>
            <IconPlus size={16} aria-hidden="true" />
            {t.addAgent}
          </Button>
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_440px]">
        <Card className="overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-3">{t.tableHeaderAgent}</th>
                <th className="px-4 py-3">{t.tableHeaderKind}</th>
                <th className="px-4 py-3">{t.tableHeaderOwner}</th>
                <th className="px-4 py-3">{t.tableHeaderStatus}</th>
                <th className="px-4 py-3 text-right">{t.tableHeaderRolesCredentials}</th>
              </tr>
            </thead>
            <tbody>
              {agents.map((agent) => (
                <tr
                  key={agent.id}
                  onClick={() => setSelectedID(agent.id)}
                  className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${
                    selectedID === agent.id ? 'bg-blue-50/60' : ''
                  }`}
                >
                  <td className="px-4 py-3">
                    <div className="font-semibold text-slate-900">{agent.name}</div>
                    {agent.description ? (
                      <div className="truncate text-xs text-slate-500">{agent.description}</div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-600">{kindLabel(agent.kind, t)}</td>
                  <td className="px-4 py-3 font-mono text-xs text-slate-600">
                    {agent.owner_user_id || '—'}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={agent.status} />
                  </td>
                  <td className="px-4 py-3 text-right text-xs text-slate-600">
                    {agent.roles.length} / {agent.client_ids.length}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {agents.length === 0 ? (
            <div className="flex min-h-40 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
              <IconRobot size={24} className="text-slate-400" aria-hidden="true" />
              <p className="mt-3">{t.emptyAgentsNotice}</p>
            </div>
          ) : null}
        </Card>

        <AgentDetailCard
          agent={selected}
          csrfToken={csrfToken}
          busy={busy}
          detailHref={
            selected ? tenantURL(`/admin/agents/${encodeURIComponent(selected.id)}`) : undefined
          }
          onChanged={(id) => run(() => refresh(id), t.agentUpdatedNotice)}
          onDeleted={() => run(() => refresh(), t.agentDeletedNotice)}
        />
      </div>

      {showCreate ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 px-4">
          <Card className="w-full max-w-md p-6">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold text-slate-900">{t.registerAgentHeading}</h2>
              <Button
                variant="ghost"
                className="px-2.5"
                onClick={() => setShowCreate(false)}
                aria-label={t.close}
              >
                <IconX size={18} aria-hidden="true" />
              </Button>
            </div>
            <form onSubmit={handleCreate} className="mt-4 grid gap-4">
              <div className="grid gap-1.5">
                <Label htmlFor="agent-name">{t.agentNameLabel}</Label>
                <Input id="agent-name" name="name" required placeholder="invoice-bot" />
                <p className="text-xs text-slate-500">{t.agentNameHelp}</p>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-description">{t.descriptionOptionalLabel}</Label>
                <Input
                  id="agent-description"
                  name="description"
                  placeholder={t.agentDescriptionPlaceholder}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-kind">{t.kindLabel}</Label>
                <select
                  id="agent-kind"
                  name="kind"
                  defaultValue="autonomous"
                  className="h-9 rounded-md border border-slate-300 bg-white px-2 text-sm"
                >
                  <option value="autonomous">{t.kindAutonomousOption}</option>
                  <option value="supervised">{t.kindSupervisedOption}</option>
                </select>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-owner">{t.ownerOptionalLabel}</Label>
                <Input id="agent-owner" name="owner_user_id" placeholder="user-1234" />
                <p className="text-xs text-slate-500">{t.ownerHelp}</p>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-roles">{t.rolesLabel}</Label>
                <Input id="agent-roles" name="roles" placeholder="invoice:read, invoice:write" />
                <p className="text-xs text-slate-500">{t.rolesHelp}</p>
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setShowCreate(false)}
                  disabled={busy}
                >
                  {t.cancel}
                </Button>
                <Button type="submit" disabled={busy}>
                  {t.register}
                </Button>
              </div>
            </form>
          </Card>
        </div>
      ) : null}
    </AdminShell>
  )
}

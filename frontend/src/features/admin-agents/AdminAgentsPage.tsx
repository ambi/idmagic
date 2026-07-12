import {
  IconArrowLeft,
  IconDotsVertical,
  IconKey,
  IconPencil,
  IconPlayerStop,
  IconPlus,
  IconPower,
  IconRefresh,
  IconRobot,
  IconTrash,
  IconX,
} from '@tabler/icons-react'
import { type FormEvent, useEffect, useState } from 'react'
import {
  AuthenticationAPIError,
  bindAdminAgentCredential,
  deleteAdminAgent,
  disableAdminAgent,
  enableAdminAgent,
  getAdminAgent,
  killAdminAgent,
  listAdminAgents,
  registerAdminAgent,
  tenantURL,
  unbindAdminAgentCredential,
  updateAdminAgent,
} from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../../components/ui/dropdown-menu'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AdminAgent } from '../../types'
import { adminAgentsDictionary, type AdminAgentsDictionary } from './AdminAgentsPage.i18n'

const STATUS_STYLES: Record<AdminAgent['status'], string> = {
  active: 'bg-emerald-100 text-emerald-700',
  disabled: 'bg-slate-200 text-slate-600',
  killed: 'bg-rose-100 text-rose-700',
}

function kindLabel(kind: AdminAgent['kind'], t: AdminAgentsDictionary) {
  return kind === 'autonomous' ? t.kindAutonomous : t.kindSupervised
}

function statusLabel(status: AdminAgent['status'], t: AdminAgentsDictionary) {
  return { active: t.statusActive, disabled: t.statusDisabled, killed: t.statusKilled }[status]
}

function StatusBadge({ status }: { status: AdminAgent['status'] }) {
  const t = useDictionary(adminAgentsDictionary)
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${STATUS_STYLES[status]}`}
    >
      {statusLabel(status, t)}
    </span>
  )
}

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
            {t.register}
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

// AdminAgentDetailPage はエージェントの編集・状態操作・資格情報管理を扱う詳細画面 (wi-49)。
export function AdminAgentDetailPage({
  csrfToken,
  actorUsername,
  agent: initialAgent,
}: {
  csrfToken: string
  actorUsername?: string
  agent: AdminAgent
}) {
  const [agent, setAgent] = useState(initialAgent)
  const [editing, setEditing] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [confirmKill, setConfirmKill] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminAgentsDictionary)

  async function reload(id: string) {
    try {
      const next = await getAdminAgent(id)
      setAgent(next)
      setError('')
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.agentReloadFailedError)
    }
  }

  async function run(action: () => Promise<void>) {
    setBusy(true)
    setError('')
    try {
      await action()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setBusy(false)
    }
  }

  async function handleDelete() {
    setBusy(true)
    setError('')
    try {
      await deleteAdminAgent(csrfToken, agent.id)
      window.location.assign(tenantURL('/admin/agents'))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.agentDeleteFailedError)
      setBusy(false)
    }
  }

  const killed = agent.status === 'killed'

  return (
    <>
      <AdminShell
        active="agents"
        actorUsername={actorUsername}
        title={agent.name}
        description={agent.description || agent.id}
        actions={
          <div className="flex items-center gap-2">
            <a
              href={tenantURL('/admin/agents')}
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
            >
              <IconArrowLeft size={16} aria-hidden="true" />
              {t.backToAgentList}
            </a>
            <Button type="button" disabled={busy || killed} onClick={() => setEditing(true)}>
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  className="size-9 px-0"
                  aria-label={t.agentActionsAriaLabel}
                  disabled={busy}
                >
                  <IconDotsVertical size={18} aria-hidden="true" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {!killed && agent.status === 'active' ? (
                  <DropdownMenuItem
                    onSelect={() =>
                      void run(async () => {
                        await disableAdminAgent(csrfToken, agent.id)
                        await reload(agent.id)
                      })
                    }
                  >
                    <IconPower size={17} aria-hidden="true" />
                    {t.disable}
                  </DropdownMenuItem>
                ) : null}
                {!killed && agent.status === 'disabled' ? (
                  <DropdownMenuItem
                    onSelect={() =>
                      void run(async () => {
                        await enableAdminAgent(csrfToken, agent.id)
                        await reload(agent.id)
                      })
                    }
                  >
                    <IconPower size={17} aria-hidden="true" />
                    {t.enable}
                  </DropdownMenuItem>
                ) : null}
                {!killed ? (
                  <DropdownMenuItem className="text-rose-700" onSelect={() => setConfirmKill(true)}>
                    <IconPlayerStop size={17} aria-hidden="true" />
                    {t.kill}
                  </DropdownMenuItem>
                ) : null}
                <DropdownMenuItem className="text-red-700" onSelect={() => setConfirmDelete(true)}>
                  <IconTrash size={17} aria-hidden="true" />
                  {t.deleteAgent}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        }
      >
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        {confirmKill ? (
          <Alert
            variant="destructive"
            className="flex flex-wrap items-center justify-between gap-2"
          >
            <span>{t.confirmKillPrompt}</span>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setConfirmKill(false)} disabled={busy}>
                {t.dismissConfirm}
              </Button>
              <Button
                variant="destructive"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    await killAdminAgent(csrfToken, agent.id)
                    setConfirmKill(false)
                    await reload(agent.id)
                  })
                }
              >
                <IconPlayerStop size={14} aria-hidden="true" />
                {t.confirmKill}
              </Button>
            </div>
          </Alert>
        ) : null}
        {confirmDelete ? (
          <Alert
            variant="destructive"
            className="flex flex-wrap items-center justify-between gap-2"
          >
            <span>{t.confirmDeleteAgentPrompt}</span>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setConfirmDelete(false)} disabled={busy}>
                {t.dismissConfirm}
              </Button>
              <Button variant="destructive" disabled={busy} onClick={() => void handleDelete()}>
                <IconTrash size={14} aria-hidden="true" />
                {t.confirmDelete}
              </Button>
            </div>
          </Alert>
        ) : null}
        <div className="max-w-3xl">
          <AgentDetailCard
            agent={agent}
            csrfToken={csrfToken}
            busy={busy}
            showActions={false}
            onChanged={(id) => void reload(id)}
            onDeleted={() => window.location.assign(tenantURL('/admin/agents'))}
          />
        </div>
      </AdminShell>
      {editing ? (
        <AgentEditorDialog
          agent={agent}
          csrfToken={csrfToken}
          onClose={() => setEditing(false)}
          onSaved={(id) => {
            setEditing(false)
            void reload(id)
          }}
        />
      ) : null}
    </>
  )
}

function AgentDetailCard({
  agent,
  csrfToken,
  busy,
  detailHref,
  showActions = true,
  onChanged,
  onDeleted,
}: {
  agent: AdminAgent | null
  csrfToken: string
  busy: boolean
  detailHref?: string
  showActions?: boolean
  onChanged: (id: string) => void
  onDeleted: () => void
}) {
  const [clientIDs, setClientIDs] = useState<string[]>(agent?.client_ids ?? [])
  const [status, setStatus] = useState<AdminAgent['status']>(agent?.status ?? 'active')
  const [addClientID, setAddClientID] = useState('')
  const [localBusy, setLocalBusy] = useState(false)
  const [localError, setLocalError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [editing, setEditing] = useState(false)
  const t = useDictionary(adminAgentsDictionary)

  useEffect(() => {
    setConfirmDelete(false)
    setEditing(false)
    setLocalError('')
    setAddClientID('')
    setClientIDs(agent?.client_ids ?? [])
    setStatus(agent?.status ?? 'active')
  }, [agent])

  if (!agent) {
    return (
      <Card className="p-5">
        <p className="text-sm text-slate-500">{t.selectAgentPrompt}</p>
      </Card>
    )
  }
  const activeAgent = agent
  const killed = status === 'killed'

  async function withLocal(action: () => Promise<void>) {
    setLocalBusy(true)
    setLocalError('')
    try {
      await action()
    } catch (cause) {
      setLocalError(cause instanceof AuthenticationAPIError ? cause.message : t.genericOpError)
    } finally {
      setLocalBusy(false)
    }
  }

  async function reloadCredentials() {
    const next = await getAdminAgent(activeAgent.id)
    setClientIDs(next.client_ids)
    setStatus(next.status)
  }

  return (
    <>
      <Card className="overflow-hidden">
        <div className="border-b border-slate-200 bg-white p-5">
          <div className="flex items-start gap-3">
            <span className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-700">
              <IconRobot size={22} aria-hidden="true" />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h2 className="truncate text-lg font-semibold text-slate-950">{agent.name}</h2>
                <StatusBadge status={status} />
              </div>
              <p className="mt-0.5 truncate font-mono text-sm text-slate-500">{agent.id}</p>
            </div>
          </div>

          {showActions ? (
            <div className="mt-4">
              <AdminPaneActions
                detailHref={detailHref}
                busy={busy || localBusy}
                onEdit={killed ? undefined : () => setEditing(true)}
                actions={[
                  {
                    label: t.deleteAgent,
                    icon: IconTrash,
                    onClick: () => setConfirmDelete(true),
                    tone: 'danger',
                  },
                ]}
              />
            </div>
          ) : null}
        </div>

        {confirmDelete ? (
          <Alert
            variant="destructive"
            className="m-5 flex flex-wrap items-center justify-between gap-2"
          >
            <span>{t.confirmDeleteAgentPrompt}</span>
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => setConfirmDelete(false)}
                disabled={localBusy}
              >
                {t.dismissConfirm}
              </Button>
              <Button
                variant="destructive"
                disabled={busy || localBusy}
                onClick={() =>
                  void withLocal(async () => {
                    await deleteAdminAgent(csrfToken, activeAgent.id)
                    onDeleted()
                  })
                }
              >
                <IconTrash size={14} aria-hidden="true" />
                {t.confirmDelete}
              </Button>
            </div>
          </Alert>
        ) : null}

        {localError ? (
          <Alert variant="destructive" className="m-5">
            {localError}
          </Alert>
        ) : null}

        <dl className="grid gap-4 p-5">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <dt className="text-xs font-bold uppercase tracking-normal text-slate-400">
                {t.kindLabel}
              </dt>
              <dd className="mt-1 text-sm text-slate-700">{kindLabel(agent.kind, t)}</dd>
            </div>
            <div>
              <dt className="text-xs font-bold uppercase tracking-normal text-slate-400">
                {t.tableHeaderOwner}
              </dt>
              <dd className="mt-1 truncate font-mono text-sm text-slate-700">
                {agent.owner_user_id || '—'}
              </dd>
            </div>
          </div>
          <div>
            <dt className="text-xs font-bold uppercase tracking-normal text-slate-400">
              {t.descriptionLabel}
            </dt>
            <dd className="mt-1 text-sm text-slate-700">{agent.description || '—'}</dd>
          </div>
          <div>
            <dt className="text-xs font-bold uppercase tracking-normal text-slate-400">
              {t.rolesLabel}
            </dt>
            <dd className="mt-1 flex flex-wrap gap-1.5">
              {agent.roles.length > 0 ? (
                agent.roles.map((role) => (
                  <span
                    key={role}
                    className="rounded-md bg-slate-100 px-2 py-1 font-mono text-xs text-slate-700"
                  >
                    {role}
                  </span>
                ))
              ) : (
                <span className="text-sm text-slate-400">{t.noneLabel}</span>
              )}
            </dd>
          </div>
        </dl>

        <section className="border-t border-slate-100 p-5">
          <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
            {t.credentialsHeading.replace('{count}', String(clientIDs.length))}
          </h3>
          <p className="mt-1 text-xs text-slate-500">{t.credentialsDescription}</p>
          <ul className="mt-3 grid gap-2">
            {clientIDs.map((clientID) => (
              <li
                key={clientID}
                className="flex items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
              >
                <span className="truncate font-mono text-slate-700">{clientID}</span>
                <Button
                  variant="ghost"
                  className="text-rose-700 hover:bg-rose-50"
                  disabled={localBusy || killed}
                  onClick={() =>
                    withLocal(async () => {
                      await unbindAdminAgentCredential(csrfToken, activeAgent.id, clientID)
                      await reloadCredentials()
                    })
                  }
                >
                  <IconX size={14} aria-hidden="true" />
                  {t.unbind}
                </Button>
              </li>
            ))}
            {clientIDs.length === 0 ? (
              <li className="text-xs text-slate-400">{t.noCredentialsNotice}</li>
            ) : null}
          </ul>

          <div className="mt-3 flex items-center gap-2">
            <Input
              value={addClientID}
              onChange={(e) => setAddClientID(e.target.value)}
              placeholder="client_id"
              aria-label={t.bindClientIdAria}
              disabled={killed}
            />
            <Button
              disabled={localBusy || killed || !addClientID.trim()}
              onClick={() =>
                withLocal(async () => {
                  await bindAdminAgentCredential(csrfToken, activeAgent.id, addClientID.trim())
                  setAddClientID('')
                  await reloadCredentials()
                })
              }
            >
              <IconKey size={14} aria-hidden="true" />
              {t.bind}
            </Button>
          </div>
        </section>
      </Card>
      {editing ? (
        <AgentEditorDialog
          agent={activeAgent}
          csrfToken={csrfToken}
          onClose={() => setEditing(false)}
          onSaved={(id) => {
            setEditing(false)
            onChanged(id)
          }}
        />
      ) : null}
    </>
  )
}

function AgentEditorDialog({
  agent,
  csrfToken,
  onClose,
  onSaved,
}: {
  agent: AdminAgent
  csrfToken: string
  onClose: () => void
  onSaved: (id: string) => void
}) {
  const [name, setName] = useState(agent.name)
  const [description, setDescription] = useState(agent.description ?? '')
  const [kind, setKind] = useState<AdminAgent['kind']>(agent.kind)
  const [ownerSub, setOwnerSub] = useState(agent.owner_user_id)
  const [roles, setRoles] = useState(agent.roles.join(', '))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminAgentsDictionary)

  const trimmedName = name.trim()
  const nextRoles = parseRoles(roles)
  const nameInvalid = trimmedName === ''
  const changed =
    trimmedName !== agent.name ||
    description.trim() !== (agent.description ?? '') ||
    kind !== agent.kind ||
    ownerSub.trim() !== agent.owner_user_id ||
    nextRoles.join(',') !== agent.roles.join(',')

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (nameInvalid || !changed) return
    setSaving(true)
    setError('')
    try {
      await updateAdminAgent(csrfToken, agent.id, {
        name: trimmedName !== agent.name ? trimmedName : undefined,
        description:
          description.trim() !== (agent.description ?? '') ? description.trim() : undefined,
        kind: kind !== agent.kind ? kind : undefined,
        owner_user_id: ownerSub.trim() !== agent.owner_user_id ? ownerSub.trim() : undefined,
        roles: nextRoles.join(',') !== agent.roles.join(',') ? nextRoles : undefined,
      })
      onSaved(agent.id)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.agentUpdateFailedError)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/30 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="agent-editor-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative flex max-h-[88vh] w-full max-w-lg flex-col overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <div>
            <p className="text-xs font-bold uppercase tracking-normal text-blue-700">
              {t.agentEyebrow}
            </p>
            <h2 id="agent-editor-title" className="mt-1 text-xl font-semibold">
              {t.editAgentHeading}
            </h2>
            <p className="mt-1 text-sm text-slate-500">{agent.name}</p>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>
        <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 overflow-y-auto">
            {error ? (
              <Alert variant="destructive" className="mb-4">
                {error}
              </Alert>
            ) : null}
            <div className="grid gap-6 p-6">
              <section className="grid gap-4">
                <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
                  {t.basicInfoHeading}
                </h3>
                <div className="grid gap-1.5">
                  <Label htmlFor="agent-editor-name">{t.agentNameLabel}</Label>
                  <Input
                    id="agent-editor-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                    aria-invalid={nameInvalid}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="agent-editor-description">{t.descriptionLabel}</Label>
                  <Input
                    id="agent-editor-description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="agent-editor-kind">{t.kindLabel}</Label>
                  <select
                    id="agent-editor-kind"
                    value={kind}
                    onChange={(e) => setKind(e.target.value as AdminAgent['kind'])}
                    className="h-9 rounded-md border border-slate-300 bg-white px-2 text-sm"
                  >
                    <option value="autonomous">{t.kindAutonomousOption}</option>
                    <option value="supervised">{t.kindSupervisedOption}</option>
                  </select>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="agent-editor-owner">{t.ownerSubLabel}</Label>
                  <Input
                    id="agent-editor-owner"
                    value={ownerSub}
                    onChange={(e) => setOwnerSub(e.target.value)}
                  />
                </div>
              </section>
              <section className="grid gap-3 border-t border-slate-200 pt-5">
                <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
                  {t.rolesLabel}
                </h3>
                <div className="grid gap-1.5">
                  <Label htmlFor="agent-editor-roles">{t.rolesLabel}</Label>
                  <Input
                    id="agent-editor-roles"
                    value={roles}
                    onChange={(e) => setRoles(e.target.value)}
                    placeholder="invoice:read, invoice:write"
                  />
                  <p className="text-xs text-slate-500">{t.rolesHelp}</p>
                </div>
              </section>
            </div>
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button type="button" variant="outline" onClick={onClose}>
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || nameInvalid || !changed}>
              {saving ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}

function parseRoles(value: string) {
  return [
    ...new Set(
      value
        .split(',')
        .map((role) => role.trim())
        .filter(Boolean),
    ),
  ]
}

function optionalValue(value: FormDataEntryValue | null) {
  const normalized = String(value ?? '').trim()
  return normalized || undefined
}

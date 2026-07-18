import {
  IconArrowLeft,
  IconDotsVertical,
  IconPencil,
  IconPlayerStop,
  IconPower,
  IconTrash,
} from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  deleteAdminAgent,
  disableAdminAgent,
  enableAdminAgent,
  getAdminAgent,
  killAdminAgent,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../../components/ui/dropdown-menu'
import { useDictionary } from '../../lib/i18n'
import type { AdminAgent } from '../../types'
import { AgentDetailCard } from './AdminAgentDetailCard'
import { AgentEditorDialog } from './AgentEditorDialog'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'

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

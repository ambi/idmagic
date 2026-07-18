import { IconKey, IconRobot, IconTrash, IconX } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import {
  AuthenticationAPIError,
  bindAdminAgentCredential,
  deleteAdminAgent,
  getAdminAgent,
  unbindAdminAgentCredential,
} from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { useDictionary } from '../../lib/i18n'
import type { AdminAgent } from '../../types'
import { AgentEditorDialog } from './AgentEditorDialog'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { kindLabel, StatusBadge } from './AdminAgentsShared'

export function AgentDetailCard({
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

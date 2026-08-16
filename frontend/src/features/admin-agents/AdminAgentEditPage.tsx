import { IconArrowLeft, IconKey, IconX } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  bindAdminAgentCredential,
  getAdminAgent,
  tenantURL,
  unbindAdminAgentCredential,
  updateAdminAgent,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import type { AdminAgent } from '../../types'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { parseRoles } from './AdminAgentsShared'

// AdminAgentEditPage はエージェントの編集専用画面 (wi-314 T012)。従来
// AgentEditorDialog がモーダルとして担っていたプロフィール編集に加え、資格情報の
// バインド/解除もここに一本化する (一覧/詳細は参照専用)。
export function AdminAgentEditPage({
  csrfToken,
  actorUsername,
  agent,
}: {
  csrfToken: string
  actorUsername?: string
  agent: AdminAgent
}) {
  const detailPath = tenantURL(`/admin/agents/${encodeURIComponent(agent.id)}`)
  const [name, setName] = useState(agent.name)
  const [description, setDescription] = useState(agent.description ?? '')
  const [kind, setKind] = useState<AdminAgent['kind']>(agent.kind)
  const [ownerSub, setOwnerSub] = useState(agent.owner_user_id)
  const [roles, setRoles] = useState(agent.roles.join(', '))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [clientIDs, setClientIDs] = useState(agent.client_ids)
  const [addClientID, setAddClientID] = useState('')
  const [credentialBusy, setCredentialBusy] = useState(false)
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
  const killed = agent.status === 'killed'

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
      window.location.assign(detailPath)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.agentUpdateFailedError)
      setSaving(false)
    }
  }

  async function withCredential(action: () => Promise<void>) {
    setCredentialBusy(true)
    setError('')
    try {
      await action()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericOpError)
    } finally {
      setCredentialBusy(false)
    }
  }

  async function reloadCredentials() {
    const next = await getAdminAgent(agent.id)
    setClientIDs(next.client_ids)
  }

  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.editAgentHeading}
      description={agent.name}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={detailPath} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.agentDetail}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full max-w-2xl overflow-hidden">
        <div className="border-b border-slate-200 px-6 py-5">
          <p className="text-xs font-bold uppercase tracking-[0.12em] text-blue-700">
            {t.agentEyebrow}
          </p>
          <h2 className="mt-1 text-xl font-semibold">{t.editAgentHeading}</h2>
        </div>
        <form onSubmit={handleSubmit} className="flex flex-col">
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
                  maxLength={LENGTH.name}
                  aria-invalid={nameInvalid}
                  disabled={killed}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-editor-description">{t.descriptionLabel}</Label>
                <Input
                  id="agent-editor-description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  maxLength={LENGTH.description}
                  disabled={killed}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-editor-kind">{t.kindLabel}</Label>
                <select
                  id="agent-editor-kind"
                  value={kind}
                  onChange={(e) => setKind(e.target.value as AdminAgent['kind'])}
                  className="h-9 rounded-md border border-slate-300 bg-white px-2 text-sm"
                  disabled={killed}
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
                  disabled={killed}
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
                  disabled={killed}
                />
                <p className="text-xs text-slate-500">{t.rolesHelp}</p>
              </div>
            </section>

            <section className="border-t border-slate-200 pt-5">
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
                      type="button"
                      variant="ghost"
                      className="text-rose-700 hover:bg-rose-50"
                      disabled={credentialBusy || killed}
                      onClick={() =>
                        void withCredential(async () => {
                          await unbindAdminAgentCredential(csrfToken, agent.id, clientID)
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
                  type="button"
                  disabled={credentialBusy || killed || !addClientID.trim()}
                  onClick={() =>
                    void withCredential(async () => {
                      await bindAdminAgentCredential(csrfToken, agent.id, addClientID.trim())
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
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button variant="outline" nativeButton={false} render={<a href={detailPath} />}>
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || nameInvalid || !changed || killed}>
              {saving ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

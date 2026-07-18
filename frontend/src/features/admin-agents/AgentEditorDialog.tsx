import { IconX } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateAdminAgent } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AdminAgent } from '../../types'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { parseRoles } from './AdminAgentsShared'

export function AgentEditorDialog({
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

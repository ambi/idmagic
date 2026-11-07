import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, tenantURL, updateAdminAgent } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { CharacterCount } from '../../components/CharacterCount'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { formSchema, requiredText, text, useFormValidation } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'

const editAgentSchema = formSchema({
  name: requiredText(LENGTH.name),
  description: text(LENGTH.description),
})
import type { AdminAgent, AdminUser } from '../../types'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { OwnerSelect } from './AdminAgentsShared'
import { returnToAgent } from './agentNavigation'

// 基本情報を一度の保存で確定する。クレデンシャルとガードレールは別の作業へ分ける。
export function AdminAgentEditPage({
  csrfToken,
  actorUsername,
  agent,
  users = [],
}: {
  csrfToken: string
  actorUsername?: string
  agent: AdminAgent
  users?: AdminUser[]
}) {
  const detailPath = tenantURL(`/admin/agents/${encodeURIComponent(agent.id)}`)
  const [name, setName] = useState(agent.name)
  const [description, setDescription] = useState(agent.description ?? '')
  const [kind, setKind] = useState<AdminAgent['kind']>(agent.kind)
  const [ownerSub, setOwnerSub] = useState(agent.owner_user_id)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminAgentsDictionary)

  const form = useFormValidation(editAgentSchema, { name, description })
  const data = form.data
  const nameChanged = data !== undefined && data.name !== agent.name
  const descriptionChanged = data !== undefined && data.description !== (agent.description ?? '')
  const changed =
    nameChanged ||
    descriptionChanged ||
    kind !== agent.kind ||
    ownerSub.trim() !== agent.owner_user_id
  const killed = agent.status === 'killed'

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!data) {
      form.revealAll()
      return
    }
    if (saving || !changed) return
    setSaving(true)
    setError('')
    try {
      await updateAdminAgent(csrfToken, agent.id, {
        name: nameChanged ? data.name : undefined,
        description: descriptionChanged ? data.description : undefined,
        kind: kind !== agent.kind ? kind : undefined,
        owner_user_id: ownerSub.trim() !== agent.owner_user_id ? ownerSub.trim() : undefined,
      })
      returnToAgent(agent.id, t.agentUpdatedNotice)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.agentUpdateFailedError)
      setSaving(false)
    }
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
      <Card className="w-full admin-form overflow-hidden">
        <form onSubmit={handleSubmit} className="flex flex-col">
          <div className="grid gap-6 p-6">
            <section className="grid gap-4">
              <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
                {t.basicInfoHeading}
              </h3>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-editor-name" required>
                  {t.agentNameLabel}
                </Label>
                <Input
                  id="agent-editor-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  disabled={killed}
                  {...form.fieldProps('name')}
                />
                <FieldError form={form} name="name" />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-editor-description">{t.descriptionLabel}</Label>
                <Input
                  id="agent-editor-description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  disabled={killed}
                  {...form.fieldProps('description')}
                />
                <CharacterCount value={description} max={LENGTH.description} />
                <FieldError form={form} name="description" />
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
                <OwnerSelect
                  id="agent-editor-owner"
                  value={ownerSub}
                  onChange={setOwnerSub}
                  users={users}
                  disabled={killed}
                />
                <p className="text-xs text-slate-500">{t.ownerEditHelp}</p>
              </div>
            </section>
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button variant="outline" nativeButton={false} render={<a href={detailPath} />}>
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || !data || !changed || killed}>
              {saving ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, registerAdminAgent, tenantURL } from '../../api'
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
import { optionalValue, parseRoles } from './AdminAgentsShared'

// AdminAgentCreatePage はエージェント登録の専用画面 (wi-314 T012)。従来
// AdminAgentsListPage 内のインラインモーダルだった登録フォームを、専用ルートへ移す。
export function AdminAgentCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = tenantURL('/admin/agents')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [kind, setKind] = useState<AdminAgent['kind']>('autonomous')
  const t = useDictionary(adminAgentsDictionary)

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = e.currentTarget
    const data = new FormData(form)
    const name = String(data.get('name') ?? '').trim()
    if (!name) return

    setBusy(true)
    setError('')
    try {
      const created = await registerAdminAgent(csrfToken, {
        name,
        description: optionalValue(data.get('description')),
        kind,
        owner_user_id: optionalValue(data.get('owner_user_id')),
        roles: parseRoles(String(data.get('roles') ?? '')),
      })
      window.location.assign(tenantURL(`/admin/agents/${encodeURIComponent(created.id)}`))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.registerAgentHeading}
      description={t.registerAgentDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToAgentListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">
          {t.registerAgentHeading}
        </h1>
      </div>

      <div className="mt-6 max-w-2xl">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6">
              {error && <Alert variant="destructive">{error}</Alert>}

              <div className="grid gap-1.5">
                <Label htmlFor="agent-name">{t.agentNameLabel}</Label>
                <Input
                  id="agent-name"
                  name="name"
                  required
                  maxLength={LENGTH.name}
                  placeholder="invoice-bot"
                />
                <p className="text-xs text-slate-500">{t.agentNameHelp}</p>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-description">{t.descriptionOptionalLabel}</Label>
                <Input
                  id="agent-description"
                  name="description"
                  maxLength={LENGTH.description}
                  placeholder={t.agentDescriptionPlaceholder}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="agent-kind">{t.kindLabel}</Label>
                <select
                  id="agent-kind"
                  value={kind}
                  onChange={(e) => setKind(e.target.value as AdminAgent['kind'])}
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
            </div>

            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <a
                href={listPath}
                className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              >
                {t.cancel}
              </a>
              <Button type="submit" disabled={busy}>
                {t.register}
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </AdminShell>
  )
}

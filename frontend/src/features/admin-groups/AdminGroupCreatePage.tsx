import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, createAdminGroup, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import { optionalValue, parseRoles } from './AdminGroupsShared'

export function AdminGroupCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = tenantURL('/admin/groups')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [membershipType, setMembershipType] = useState<'manual' | 'dynamic'>('manual')
  const t = useDictionary(adminGroupsDictionary)

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = e.currentTarget
    const data = new FormData(form)
    const name = String(data.get('name') ?? '').trim()
    if (!name) return

    setBusy(true)
    setError('')
    try {
      const created = await createAdminGroup(csrfToken, {
        name,
        description: optionalValue(data.get('description')),
        roles: parseRoles(String(data.get('roles') ?? '')),
        membership_type: membershipType,
        dynamic_rule:
          membershipType === 'dynamic'
            ? { expression: String(data.get('dynamic-rule') ?? '').trim() }
            : undefined,
      })
      window.location.assign(tenantURL(`/admin/groups/${encodeURIComponent(created.id)}`))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.groupCreateFailedError)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="groups"
      actorUsername={actorUsername}
      title={t.addGroup}
      description={t.addGroupDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToGroupListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.addGroup}</h1>
      </div>

      <div className="mt-6 max-w-2xl">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6">
              {error && <Alert variant="destructive">{error}</Alert>}

              <div className="grid gap-1.5">
                <Label htmlFor="group-name">
                  {t.groupNameLabel} <span className="text-red-500">*</span>
                </Label>
                <Input id="group-name" name="name" required placeholder="engineering" />
                <p className="text-xs text-slate-500">{t.groupNameHelp}</p>
              </div>

              <div className="grid gap-1.5">
                <Label htmlFor="group-membership-type">{t.membershipType}</Label>
                <select
                  id="group-membership-type"
                  value={membershipType}
                  onChange={(event) =>
                    setMembershipType(event.target.value as 'manual' | 'dynamic')
                  }
                  className="h-9 rounded-md border border-slate-300 bg-white px-2 text-sm"
                >
                  <option value="manual">{t.manualMembership}</option>
                  <option value="dynamic">{t.dynamicMembership}</option>
                </select>
              </div>
              {membershipType === 'dynamic' ? (
                <div className="grid gap-1.5">
                  <Label htmlFor="group-dynamic-rule">{t.dynamicRuleExpression}</Label>
                  <textarea
                    id="group-dynamic-rule"
                    name="dynamic-rule"
                    required
                    className="min-h-28 rounded-md border border-slate-300 bg-white p-3 font-mono text-sm"
                    placeholder={'user.department == "Engineering"'}
                  />
                  <p className="text-xs text-slate-500">{t.dynamicRuleHelp}</p>
                </div>
              ) : null}

              <div className="grid gap-1.5">
                <Label htmlFor="group-description">{t.descriptionOptionalLabel}</Label>
                <Input
                  id="group-description"
                  name="description"
                  placeholder={t.groupDescriptionPlaceholder}
                />
              </div>

              <div className="grid gap-1.5">
                <Label htmlFor="group-roles">{t.rolesLabel}</Label>
                <Input id="group-roles" name="roles" placeholder="catalog:read, invoice:read" />
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
                {t.create}
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </AdminShell>
  )
}

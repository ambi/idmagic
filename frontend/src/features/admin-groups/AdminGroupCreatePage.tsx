import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useMemo, useState } from 'react'
import { AuthenticationAPIError, createAdminGroup, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { CharacterCount } from '../../components/CharacterCount'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useForm } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import type { TenantGroupAttributeSchema } from '../../types'
import { AdminGroupAttributeEditor, groupAttributeMapFromDraft } from './AdminGroupAttributeEditor'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import { createGroupSchema } from './AdminGroupsShared'

export function AdminGroupCreatePage({
  csrfToken,
  actorUsername,
  groupAttributeSchema,
}: {
  csrfToken: string
  actorUsername?: string
  groupAttributeSchema: TenantGroupAttributeSchema
}) {
  const listPath = tenantURL('/admin/groups')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [membershipType, setMembershipType] = useState<'manual' | 'dynamic'>('manual')
  const [attributeDraft, setAttributeDraft] = useState<Record<string, string>>({})
  const t = useDictionary(adminGroupsDictionary)
  const schema = useMemo(() => createGroupSchema(membershipType === 'dynamic'), [membershipType])
  const form = useForm(schema, { name: '', description: '', email: '', expression: '', roles: '' })

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const data = form.data
    if (!data) {
      form.revealAll()
      return
    }

    setBusy(true)
    setError('')
    try {
      const created = await createAdminGroup(csrfToken, {
        name: data.name,
        description: data.description,
        email: data.email,
        attributes: groupAttributeMapFromDraft(attributeDraft, groupAttributeSchema.attributes),
        roles: data.roles,
        membership_type: membershipType,
        dynamic_rule: data.expression === undefined ? undefined : { expression: data.expression },
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
      </div>

      <div className="mt-6 admin-form">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6 md:grid-cols-2">
              {error && <Alert variant="destructive">{error}</Alert>}

              <div className="grid gap-1.5">
                <Label htmlFor="group-name" required>
                  {t.groupNameLabel}
                </Label>
                <Input id="group-name" required placeholder="engineering" {...form.input('name')} />
                <FieldError form={form} name="name" />
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
                  <Label htmlFor="group-dynamic-rule" required>
                    {t.dynamicRuleExpression}
                  </Label>
                  <textarea
                    id="group-dynamic-rule"
                    required
                    className="min-h-28 rounded-md border border-slate-300 bg-white p-3 font-mono text-sm"
                    placeholder={'user.department == "Engineering"'}
                    {...form.input('expression')}
                  />
                  <FieldError form={form} name="expression" />
                  <p className="text-xs text-slate-500">{t.dynamicRuleHelp}</p>
                </div>
              ) : null}

              <div className="grid gap-1.5">
                <Label htmlFor="group-description" optional>
                  {t.descriptionOptionalLabel}
                </Label>
                <Input
                  id="group-description"
                  placeholder={t.groupDescriptionPlaceholder}
                  {...form.input('description')}
                />
                <CharacterCount value={form.values.description} max={LENGTH.description} />
                <FieldError form={form} name="description" />
              </div>

              <div className="grid gap-1.5">
                <Label htmlFor="group-email" optional>
                  {t.emailOptionalLabel}
                </Label>
                <Input id="group-email" type="email" {...form.input('email')} />
                <FieldError form={form} name="email" />
                <p className="text-xs text-slate-500">{t.emailHelp}</p>
              </div>

              <AdminGroupAttributeEditor
                defs={groupAttributeSchema.attributes}
                values={attributeDraft}
                onChange={(key, next) =>
                  setAttributeDraft((current) => ({ ...current, [key]: next }))
                }
              />

              <div className="grid gap-1.5">
                <Label htmlFor="group-roles">{t.rolesLabel}</Label>
                <Input
                  id="group-roles"
                  placeholder="catalog:read, invoice:read"
                  {...form.input('roles')}
                />
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
              <Button type="submit" disabled={busy || !form.data}>
                {t.create}
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </AdminShell>
  )
}

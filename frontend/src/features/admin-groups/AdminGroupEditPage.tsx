import { IconArrowLeft, IconAlertTriangle } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, tenantURL, updateAdminGroup } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import type { AdminGroup, TenantGroupAttributeSchema, TenantUserAttributeSchema } from '../../types'
import {
  AdminGroupAttributeEditor,
  groupAttributeDraft,
  groupAttributeMapFromDraft,
} from './AdminGroupAttributeEditor'
import { GroupMembersSection } from './AdminGroupDetailCard'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import { parseRoles } from './AdminGroupsShared'
import { DynamicRuleEditor } from './DynamicRuleEditor'

export function AdminGroupEditPage({
  csrfToken,
  actorUsername,
  group,
  schema,
  groupAttributeSchema,
}: {
  csrfToken: string
  actorUsername?: string
  group: AdminGroup
  schema: TenantUserAttributeSchema
  groupAttributeSchema: TenantGroupAttributeSchema
}) {
  const detailPath = tenantURL(`/admin/groups/${encodeURIComponent(group.id)}`)
  const [name, setName] = useState(group.name)
  const [description, setDescription] = useState(group.description ?? '')
  const [email, setEmail] = useState(group.email ?? '')
  const [attributeDraft, setAttributeDraft] = useState(() =>
    groupAttributeDraft(group, groupAttributeSchema.attributes),
  )
  const [roles, setRoles] = useState(group.roles.join(', '))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminGroupsDictionary)

  const trimmedName = name.trim()
  const nextRoles = parseRoles(roles)
  const nameInvalid = trimmedName === ''
  const nextAttributes = groupAttributeMapFromDraft(attributeDraft, groupAttributeSchema.attributes)
  const attributesChanged =
    JSON.stringify(nextAttributes) !== JSON.stringify(group.attributes ?? {})
  const changed =
    trimmedName !== group.name ||
    description.trim() !== (group.description ?? '') ||
    email.trim() !== (group.email ?? '') ||
    nextRoles.join(',') !== group.roles.join(',') ||
    attributesChanged

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (nameInvalid || !changed) return
    setSaving(true)
    setError('')
    try {
      await updateAdminGroup(csrfToken, group.id, {
        name: trimmedName !== group.name ? trimmedName : undefined,
        description:
          description.trim() !== (group.description ?? '') ? description.trim() : undefined,
        email: email.trim() !== (group.email ?? '') ? email.trim() : undefined,
        attributes: attributesChanged ? nextAttributes : undefined,
        roles: nextRoles.join(',') !== group.roles.join(',') ? nextRoles : undefined,
      })
      window.location.assign(detailPath)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.groupUpdateFailedError)
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="groups"
      actorUsername={actorUsername}
      title={t.editGroup}
      description={t.editGroupDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={detailPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToDetailAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.editGroup}</h1>
      </div>

      <div className="mt-6 max-w-2xl">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6">
              {error && <Alert variant="destructive">{error}</Alert>}

              {group.scim_source && (
                <Alert className="mb-2">
                  <div className="flex gap-3">
                    <IconAlertTriangle className="mt-0.5 shrink-0 text-blue-700" size={19} />
                    <div>
                      <p className="text-sm font-semibold text-blue-950">{t.scimSyncGroupTitle}</p>
                      <p className="mt-1 text-xs leading-5 text-blue-800">
                        {t.scimSyncGroupDescription.replace('{source}', group.scim_source)}
                      </p>
                    </div>
                  </div>
                </Alert>
              )}

              <section className="grid gap-4">
                <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
                  {t.basicInfoHeading}
                </h3>
                <div className="grid gap-1.5">
                  <Label htmlFor="group-editor-name">{t.groupNameLabel}</Label>
                  <Input
                    id="group-editor-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                    maxLength={LENGTH.name}
                    aria-invalid={nameInvalid}
                    readOnly={!!group.scim_source}
                    className={group.scim_source ? 'bg-slate-50' : undefined}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="group-editor-description">{t.descriptionLabel}</Label>
                  <Input
                    id="group-editor-description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    maxLength={LENGTH.description}
                    readOnly={!!group.scim_source}
                    className={group.scim_source ? 'bg-slate-50' : undefined}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="group-editor-email">{t.emailLabel}</Label>
                  <Input
                    id="group-editor-email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    maxLength={LENGTH.email}
                  />
                  <p className="text-xs text-slate-500">{t.emailHelp}</p>
                </div>
              </section>
              <AdminGroupAttributeEditor
                defs={groupAttributeSchema.attributes}
                values={attributeDraft}
                onChange={(key, next) =>
                  setAttributeDraft((current) => ({ ...current, [key]: next }))
                }
              />
              <section className="grid gap-3 border-t border-slate-200 pt-5">
                <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
                  {t.rolesLabel}
                </h3>
                <div className="grid gap-1.5">
                  <Label htmlFor="group-editor-roles">{t.rolesLabel}</Label>
                  <Input
                    id="group-editor-roles"
                    value={roles}
                    onChange={(e) => setRoles(e.target.value)}
                    placeholder="catalog:read, invoice:read"
                  />
                  <p className="text-xs text-slate-500">{t.rolesHelp}</p>
                </div>
              </section>
            </div>

            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <a
                href={detailPath}
                className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              >
                {t.cancel}
              </a>
              <Button type="submit" disabled={saving || nameInvalid || !changed}>
                {saving ? t.saving : t.save}
              </Button>
            </div>
          </form>
        </Card>
      </div>

      <div className="mt-6 max-w-2xl">
        <Card className="overflow-hidden">
          <GroupMembersSection group={group} csrfToken={csrfToken} />
        </Card>
      </div>

      {group.membership_type === 'dynamic' ? (
        <div className="mt-6 max-w-2xl">
          <DynamicRuleEditor
            csrfToken={csrfToken}
            groupId={group.id}
            initialRule={group.dynamic_rule}
            customAttributes={schema.attributes}
          />
        </div>
      ) : null}
    </AdminShell>
  )
}

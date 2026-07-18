import { IconAlertTriangle, IconArrowLeft, IconShield } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  tenantURL,
  type UpdateAdminUserInput,
  updateAdminUser,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { domainLabelsDictionary } from '../../lib/i18n/domainLabels.i18n'
import { attributeLabel, cn } from '../../lib/utils'
import type {
  AdminUser,
  AttributeValue,
  TenantUserAttributeSchema,
  UserAttributeDef,
} from '../../types'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import { parseRoles } from './AdminUsersPrimitives'
import { attributeValueToText, groupedAttributeDefs } from './AdminUsersShared'

function textToAttributeValue(def: UserAttributeDef, text: string): AttributeValue | undefined {
  const trimmed = text.trim()
  switch (def.type) {
    case 'boolean':
      return { type: 'boolean', boolean: text === 'true' }
    case 'number':
      return trimmed ? { type: 'number', number: Number(trimmed) } : undefined
    case 'date':
      return trimmed ? { type: 'date', date: trimmed } : undefined
    case 'string_array': {
      const items = trimmed
        .split(',')
        .map((item) => item.trim())
        .filter((item) => item.length > 0)
      return items.length ? { type: 'string_array', string_array: items } : undefined
    }
    default:
      return trimmed ? { type: 'string', string: trimmed } : undefined
  }
}

function attributeDraftFromUser(user: AdminUser, defs: UserAttributeDef[]): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const def of defs) {
    const value = user.attributes?.[def.key]
    draft[def.key] = value ? attributeValueToText(value) : ''
  }
  return draft
}

function attributeMapFromDraft(
  draft: Record<string, string>,
  defs: UserAttributeDef[],
): Record<string, AttributeValue> {
  const map: Record<string, AttributeValue> = {}
  for (const def of defs) {
    const value = textToAttributeValue(def, draft[def.key] ?? '')
    if (value) {
      map[def.key] = value
    }
  }
  return map
}

function AdminAttributeField({
  def,
  value,
  onChange,
  readOnly,
}: {
  def: UserAttributeDef
  value: string
  onChange: (next: string) => void
  readOnly?: boolean
}) {
  const id = `user-editor-attr-${def.key}`
  const label = attributeLabel(def)
  const t = useDictionary(adminUsersDictionary)
  if (def.type === 'boolean') {
    return (
      <label htmlFor={id} className="inline-flex items-center gap-2 text-sm text-slate-700">
        <input
          id={id}
          type="checkbox"
          checked={value === 'true'}
          onChange={(event) => !readOnly && onChange(event.target.checked ? 'true' : 'false')}
          disabled={readOnly}
          className="size-4 rounded border-slate-300 disabled:opacity-50"
        />
        <span className="font-mono">{label}</span>
      </label>
    )
  }
  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id} className="font-mono text-xs">
        {label}
      </Label>
      <Input
        id={id}
        type={def.type === 'number' ? 'number' : def.type === 'date' ? 'date' : 'text'}
        value={value}
        placeholder={def.type === 'string_array' ? t.commaSeparated : undefined}
        onChange={(event) => onChange(event.target.value)}
        readOnly={readOnly}
        className={readOnly ? 'bg-slate-50' : undefined}
      />
    </div>
  )
}

function AdminAttributeEditorGroups({
  defs,
  values,
  onChange,
  readOnly,
}: {
  defs: UserAttributeDef[]
  values: Record<string, string>
  onChange: (key: string, next: string) => void
  readOnly?: boolean
}) {
  const tLabels = useDictionary(domainLabelsDictionary)
  const t = useDictionary(adminUsersDictionary)
  const groups = groupedAttributeDefs(defs, tLabels)
  if (groups.length === 0) return null
  return (
    <section className="grid gap-4 border-t border-slate-200 pt-5">
      <div>
        <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
          {t.accountInfoHeading}
        </h3>
        <p className="mt-1 text-xs leading-5 text-slate-500">{t.accountInfoDescription}</p>
      </div>
      {groups.map((group) => (
        <fieldset key={group.key} className="grid gap-3 rounded-lg border border-slate-200 p-4">
          <legend className="px-1 text-xs font-bold uppercase tracking-normal text-slate-500">
            {group.title}
          </legend>
          {group.defs.map((def) => (
            <AdminAttributeField
              key={def.key}
              def={def}
              value={values[def.key] ?? ''}
              onChange={(next) => onChange(def.key, next)}
              readOnly={readOnly}
            />
          ))}
        </fieldset>
      ))}
    </section>
  )
}

function RoleDiff({
  title,
  roles,
  tone,
}: {
  title: string
  roles: string[]
  tone: 'add' | 'remove'
}) {
  const t = useDictionary(adminUsersDictionary)
  return (
    <div>
      <p className="text-xs font-semibold text-slate-500">{title}</p>
      <div className="mt-2 flex min-h-16 flex-wrap content-start gap-1.5 rounded-xl border border-slate-200 bg-white p-3">
        {roles.length === 0 ? (
          <span className="text-xs text-slate-400">{t.roleNone}</span>
        ) : (
          roles.map((role) => (
            <span
              key={role}
              className={cn(
                'rounded-md px-2 py-1 text-xs font-semibold',
                tone === 'add' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700',
              )}
            >
              {tone === 'add' ? '+' : '-'} {role}
            </span>
          ))
        )}
      </div>
    </div>
  )
}

// AdminUserEditPage はユーザー編集の専用画面 (wi-126 §6)。従来モーダルだった編集
// フォームを詳細→編集ポリシーに沿って独立画面へ移し、保存後は詳細画面へ戻す。
// ロール変更を含む場合は保存前に確認ステップ (confirming) を同一画面で挟む。
export function AdminUserEditPage({
  csrfToken,
  actorUsername,
  user,
  schema,
}: {
  csrfToken: string
  actorUsername?: string
  user: AdminUser
  schema: TenantUserAttributeSchema
}) {
  const attributeDefs = [...schema.builtin, ...schema.attributes]
  const detailPath = tenantURL(`/admin/users/${encodeURIComponent(user.id)}`)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminUsersDictionary)

  async function persist(input: UpdateAdminUserInput) {
    setBusy(true)
    setError('')
    try {
      await updateAdminUser(csrfToken, user.id, input)
      window.location.assign(detailPath)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.updateFailed)
      setBusy(false)
    }
  }

  const initialUsername = user.preferred_username
  const initialName = user.name ?? ''
  const initialGivenName = user.given_name ?? ''
  const initialFamilyName = user.family_name ?? ''
  const initialEmail = user.email ?? ''
  const initialEmailVerified = user.email_verified
  const initialAttrDraft = attributeDraftFromUser(user, attributeDefs)

  const [username, setUsername] = useState(initialUsername)
  const [name, setName] = useState(initialName)
  const [givenName, setGivenName] = useState(initialGivenName)
  const [familyName, setFamilyName] = useState(initialFamilyName)
  const [email, setEmail] = useState(initialEmail)
  const [emailVerified, setEmailVerified] = useState(initialEmailVerified)
  const [emailVerifiedTouched, setEmailVerifiedTouched] = useState(false)
  const [roles, setRoles] = useState(user.roles.join(', '))
  const [attrDraft, setAttrDraft] = useState<Record<string, string>>(initialAttrDraft)
  const [confirming, setConfirming] = useState(false)

  const emailChanged = email !== initialEmail
  const effectiveEmailVerified = emailChanged && !emailVerifiedTouched ? false : emailVerified
  const trimmedUsername = username.trim()
  const usernameInvalid = trimmedUsername === ''
  const attributesChanged = attributeDefs.some(
    (def) => (attrDraft[def.key] ?? '') !== (initialAttrDraft[def.key] ?? ''),
  )
  const profileChanged =
    trimmedUsername !== initialUsername ||
    name !== initialName ||
    givenName !== initialGivenName ||
    familyName !== initialFamilyName ||
    email !== initialEmail ||
    effectiveEmailVerified !== initialEmailVerified ||
    attributesChanged
  const nextRoles = parseRoles(roles)
  const addedRoles = nextRoles.filter((role) => !user.roles.includes(role))
  const removedRoles = user.roles.filter((role) => !nextRoles.includes(role))
  const rolesChanged = addedRoles.length > 0 || removedRoles.length > 0
  const changed = profileChanged || rolesChanged

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (usernameInvalid || !changed) return
    if (rolesChanged && !confirming) {
      setConfirming(true)
      return
    }
    const input: UpdateAdminUserInput = {}
    if (trimmedUsername !== initialUsername) input.preferred_username = trimmedUsername
    if (name !== initialName) input.name = name
    if (givenName !== initialGivenName) input.given_name = givenName
    if (familyName !== initialFamilyName) input.family_name = familyName
    if (email !== initialEmail) input.email = email
    if (effectiveEmailVerified !== initialEmailVerified) {
      input.email_verified = effectiveEmailVerified
    }
    if (rolesChanged) input.roles = nextRoles
    // admin は属性バッグ全体を置換するため、ドラフトから完全な map を再構成する。
    if (attributesChanged) input.attributes = attributeMapFromDraft(attrDraft, attributeDefs)
    void persist(input)
  }

  return (
    <AdminShell
      active="users"
      actorUsername={actorUsername}
      title={t.editUserTitle}
      description={`${user.name || user.preferred_username} (@${user.preferred_username})`}
      actions={
        <Button asChild variant="outline">
          <a href={detailPath}>
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.userDetail}
          </a>
        </Button>
      }
    >
      {error && <Alert>{error}</Alert>}
      <Card className="mx-auto w-full max-w-3xl overflow-hidden">
        <div className="border-b border-slate-200 px-6 py-5">
          <p className="text-xs font-bold uppercase tracking-[0.12em] text-blue-700">
            {t.profileAndAccessLabel}
          </p>
          <h2 className="mt-1 text-xl font-semibold">
            {confirming ? t.confirmChangesHeading : t.editUserTitle}
          </h2>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col">
          <div>
            {confirming ? (
              <div className="p-6">
                <div className="rounded-xl border border-amber-200 bg-amber-50 p-4">
                  <div className="flex gap-3">
                    <IconShield
                      size={19}
                      className="mt-0.5 shrink-0 text-amber-700"
                      aria-hidden="true"
                    />
                    <div>
                      <p className="text-sm font-semibold text-amber-950">
                        {t.roleChangeWarningTitle}
                      </p>
                      <p className="mt-1 text-xs leading-5 text-amber-800">
                        {t.roleChangeWarningDescription}
                      </p>
                    </div>
                  </div>
                </div>
                <div className="mt-5 grid gap-4 sm:grid-cols-2">
                  <RoleDiff title={t.rolesAdded} roles={addedRoles} tone="add" />
                  <RoleDiff title={t.rolesRemoved} roles={removedRoles} tone="remove" />
                </div>
                {profileChanged && (
                  <p className="mt-4 text-xs leading-5 text-slate-500">{t.profileChangeNotice}</p>
                )}
              </div>
            ) : (
              <div className="grid gap-6 p-6">
                {user.scim_source && (
                  <Alert>
                    <div className="flex gap-3">
                      <IconAlertTriangle className="mt-0.5 shrink-0 text-blue-700" size={19} />
                      <div>
                        <p className="text-sm font-semibold text-blue-950">{t.scimSyncUserTitle}</p>
                        <p className="mt-1 text-xs leading-5 text-blue-800">
                          {t.scimSyncUserDescription.replace('{source}', user.scim_source)}
                        </p>
                      </div>
                    </div>
                  </Alert>
                )}

                <section className="grid gap-4">
                  <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
                    {t.profileHeading}
                  </h3>
                  <div className="grid gap-2">
                    <Label htmlFor="user-editor-username">{t.username}</Label>
                    <Input
                      id="user-editor-username"
                      value={username}
                      onChange={(event) => setUsername(event.target.value)}
                      autoFocus={!user.scim_source}
                      required
                      aria-invalid={usernameInvalid}
                      readOnly={!!user.scim_source}
                      className={user.scim_source ? 'bg-slate-50' : undefined}
                    />
                    <p className="text-xs leading-5 text-slate-500">{t.usernameHelp}</p>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="user-editor-name">{t.displayName}</Label>
                    <Input
                      id="user-editor-name"
                      value={name}
                      onChange={(event) => setName(event.target.value)}
                      readOnly={!!user.scim_source}
                      className={user.scim_source ? 'bg-slate-50' : undefined}
                    />
                  </div>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="grid gap-2">
                      <Label htmlFor="user-editor-given-name">{t.givenName} (given_name)</Label>
                      <Input
                        id="user-editor-given-name"
                        value={givenName}
                        onChange={(event) => setGivenName(event.target.value)}
                        readOnly={!!user.scim_source}
                        className={user.scim_source ? 'bg-slate-50' : undefined}
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label htmlFor="user-editor-family-name">{t.familyName} (family_name)</Label>
                      <Input
                        id="user-editor-family-name"
                        value={familyName}
                        onChange={(event) => setFamilyName(event.target.value)}
                        readOnly={!!user.scim_source}
                        className={user.scim_source ? 'bg-slate-50' : undefined}
                      />
                    </div>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="user-editor-email">{t.emailFieldLabel}</Label>
                    <Input
                      id="user-editor-email"
                      type="email"
                      value={email}
                      onChange={(event) => {
                        setEmail(event.target.value)
                        setEmailVerifiedTouched(false)
                      }}
                      readOnly={!!user.scim_source}
                      className={user.scim_source ? 'bg-slate-50' : undefined}
                    />
                    {emailChanged && (
                      <p className="text-xs leading-5 text-amber-700">
                        {t.emailChangedVerificationNotice}
                      </p>
                    )}
                  </div>
                  <label className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700">
                    <input
                      type="checkbox"
                      className="mt-0.5 size-4 rounded border-slate-300 disabled:opacity-50"
                      checked={effectiveEmailVerified}
                      onChange={(event) => {
                        setEmailVerified(event.target.checked)
                        setEmailVerifiedTouched(true)
                      }}
                      disabled={!!user.scim_source}
                    />
                    <span>
                      <span className="block font-semibold text-slate-900">{t.saveAsVerified}</span>
                      <span className="mt-0.5 block text-xs leading-5 text-slate-500">
                        {t.verifiedOwnershipNotice}
                      </span>
                    </span>
                  </label>
                </section>
                <section className="grid gap-2 border-t border-slate-200 pt-5">
                  <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
                    {t.rolesHeading}
                  </h3>
                  <Label htmlFor="user-editor-roles">{t.rolesHeading}</Label>
                  <Input
                    id="user-editor-roles"
                    value={roles}
                    onChange={(event) => setRoles(event.target.value)}
                    placeholder="admin, support"
                  />
                  <p className="text-xs leading-5 text-slate-500">{t.rolesHelp}</p>
                </section>
                <AdminAttributeEditorGroups
                  defs={attributeDefs}
                  values={attrDraft}
                  onChange={(key, next) => setAttrDraft((current) => ({ ...current, [key]: next }))}
                  readOnly={!!user.scim_source}
                />
              </div>
            )}
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            {confirming ? (
              <Button type="button" variant="outline" onClick={() => setConfirming(false)}>
                {t.back}
              </Button>
            ) : (
              <Button asChild variant="outline">
                <a href={detailPath}>{t.cancel}</a>
              </Button>
            )}
            <Button type="submit" disabled={busy || usernameInvalid || !changed}>
              {confirming ? t.confirmChanges : rolesChanged ? t.confirmChangesHeading : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

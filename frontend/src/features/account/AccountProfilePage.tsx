import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateAccountProfile } from '../../api'
import { attributeGroupKey, attributeGroupTitle, attributeLabel } from '../../lib/utils'
import { useDictionary } from '../../lib/i18n'
import {
  domainLabelsDictionary,
  type DomainLabelsDictionary,
} from '../../lib/i18n/domainLabels.i18n'
import { AccountShell } from '../../components/AccountShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import type { AccountProfile, AttributeValue, UserAttributeDef } from '../../types'
import { accountProfileDictionary } from './AccountProfilePage.i18n'

// 編集フォーム上の属性値は文字列で保持し、保存時に AttributeValue へ整形する。
export type AttributeDraft = Record<string, string>

export function draftFromProfile(profile: AccountProfile): AttributeDraft {
  const draft: AttributeDraft = {}
  for (const def of profile.editable_attributes) {
    const value = profile.attributes[def.key]
    draft[def.key] = value ? valueToText(value) : ''
  }
  return draft
}

export function valueToText(value: AttributeValue): string {
  switch (value.type) {
    case 'string':
      return value.string ?? ''
    case 'date':
      return value.date ?? ''
    case 'number':
      return value.number?.toString() ?? ''
    case 'boolean':
      return value.boolean ? 'true' : 'false'
    case 'string_array':
      return (value.string_array ?? []).join(', ')
    default:
      return ''
  }
}

// textToValue は空入力なら undefined を返し、その key を送らない (self-delete はしない)。
export function textToValue(def: UserAttributeDef, text: string): AttributeValue | undefined {
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

export function AccountProfilePage({
  profile,
  isAdmin,
}: {
  csrfToken: string
  profile: AccountProfile
  isAdmin: boolean
}) {
  const t = useDictionary(accountProfileDictionary)
  const [notice, setNotice] = useState(() => {
    return new URLSearchParams(window.location.search).get('notice') === 'success' ? t.updated : ''
  })

  return (
    <AccountProfilePresentation
      profile={profile}
      isAdmin={isAdmin}
      notice={notice}
      onDismissNotice={() => setNotice('')}
    />
  )
}

export function AccountProfilePresentation({
  profile,
  isAdmin,
  notice,
  onDismissNotice,
}: {
  profile: AccountProfile
  isAdmin: boolean
  notice: string
  onDismissNotice: () => void
}) {
  const t = useDictionary(accountProfileDictionary)
  return (
    <AccountShell
      active="profile"
      username={profile.preferred_username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      <div className="grid gap-6">
        <Toast message={notice} onDismiss={onDismissNotice} />

        <Card className="p-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h2 className="text-base font-semibold text-slate-900">{t.profile}</h2>
              <p className="mt-1 text-sm text-slate-600">{t.profileDescription}</p>
            </div>
            <Button asChild variant="outline">
              <a href="/account/profile/edit">{t.edit}</a>
            </Button>
          </div>

          <dl className="mt-5 grid gap-3 sm:grid-cols-2">
            <ReadField label={t.displayName} value={profile.name ?? t.notSet} />
            <ReadField label={t.givenName} value={profile.given_name ?? t.notSet} />
            <ReadField label={t.familyName} value={profile.family_name ?? t.notSet} />
            <ReadField
              label={t.email}
              value={profile.email ?? t.notSet}
              action={
                <a
                  href="/account/emails"
                  className="text-xs font-semibold text-blue-600 hover:text-blue-700 hover:underline"
                >
                  {t.change}
                </a>
              }
            />
            <ReadField
              label={t.emailVerification}
              value={profile.email_verified ? t.verified : t.unverified}
            />
            <ReadField label={t.mfa} value={profile.mfa_enrolled ? t.enrolled : t.notEnrolled} />
            <ReadField label={t.status} value={profile.status} />
          </dl>
          <div className="mt-5 grid gap-4">
            <ProfileAttributeGroups
              defs={profile.readable_attributes}
              values={profile.attributes}
            />
          </div>
        </Card>
      </div>
    </AccountShell>
  )
}

export function AccountProfileEditPage({
  csrfToken,
  profile,
  isAdmin,
}: {
  csrfToken: string
  profile: AccountProfile
  isAdmin: boolean
}) {
  const t = useDictionary(accountProfileDictionary)
  const [name, setName] = useState(profile.name ?? '')
  const [givenName, setGivenName] = useState(profile.given_name ?? '')
  const [familyName, setFamilyName] = useState(profile.family_name ?? '')
  const [attributes, setAttributes] = useState<AttributeDraft>(draftFromProfile(profile))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const nextAttributes: AccountProfile['attributes'] = {}
      for (const def of profile.editable_attributes) {
        const value = textToValue(def, attributes[def.key] ?? '')
        if (value) {
          nextAttributes[def.key] = value
        }
      }
      await updateAccountProfile(csrfToken, {
        name: name.trim() || undefined,
        given_name: givenName.trim() || undefined,
        family_name: familyName.trim() || undefined,
        attributes: nextAttributes,
      })
      window.location.assign('/account/profile?notice=success')
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.updateFailed)
      setSaving(false)
    }
  }

  return (
    <AccountProfileEditPresentation
      profile={profile}
      isAdmin={isAdmin}
      name={name}
      givenName={givenName}
      familyName={familyName}
      attributes={attributes}
      saving={saving}
      error={error}
      onNameChange={setName}
      onGivenNameChange={setGivenName}
      onFamilyNameChange={setFamilyName}
      onAttributeChange={(key, next) => setAttributes((current) => ({ ...current, [key]: next }))}
      onSubmit={handleSave}
    />
  )
}

export function AccountProfileEditPresentation({
  profile,
  isAdmin,
  name,
  givenName,
  familyName,
  attributes,
  saving,
  error,
  onNameChange,
  onGivenNameChange,
  onFamilyNameChange,
  onAttributeChange,
  onSubmit,
}: {
  profile: AccountProfile
  isAdmin: boolean
  name: string
  givenName: string
  familyName: string
  attributes: AttributeDraft
  saving: boolean
  error: string
  onNameChange: (value: string) => void
  onGivenNameChange: (value: string) => void
  onFamilyNameChange: (value: string) => void
  onAttributeChange: (key: string, value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  const t = useDictionary(accountProfileDictionary)
  return (
    <AccountShell
      active="profile"
      username={profile.preferred_username}
      isAdmin={isAdmin}
      title={t.editTitle}
      description={t.editDescription}
    >
      <div className="grid gap-6">
        {error ? <Alert variant="destructive">{error}</Alert> : null}

        <Card className="p-5">
          <div className="flex items-center gap-3">
            <a
              href="/account/profile"
              className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
              aria-label={t.back}
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <title>{t.backIcon}</title>
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10 19l-7-7m0 0l7-7m-7 7h18"
                />
              </svg>
            </a>
            <h2 className="text-base font-semibold text-slate-900">{t.editHeading}</h2>
          </div>

          <form onSubmit={onSubmit} className="mt-5 grid gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="name">{t.displayName}</Label>
              <Input
                id="name"
                value={name}
                onChange={(event) => onNameChange(event.target.value)}
              />
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="given-name">{t.givenName} (given_name)</Label>
                <Input
                  id="given-name"
                  value={givenName}
                  onChange={(event) => onGivenNameChange(event.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="family-name">{t.familyName} (family_name)</Label>
                <Input
                  id="family-name"
                  value={familyName}
                  onChange={(event) => onFamilyNameChange(event.target.value)}
                />
              </div>
            </div>

            <EditableAttributeGroups
              defs={profile.editable_attributes}
              values={attributes}
              onChange={onAttributeChange}
            />

            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving}>
                {saving ? t.saving : t.save}
              </Button>
              <Button type="button" variant="ghost" disabled={saving} asChild>
                <a href="/account/profile">{t.cancel}</a>
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </AccountShell>
  )
}

function ReadField({
  label,
  value,
  action,
}: {
  label: string
  value: string
  action?: React.ReactNode
}) {
  return (
    <div className="relative rounded-lg border border-slate-200/80 bg-white/70 px-3 py-2.5">
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className="mt-0.5 flex items-center justify-between gap-2">
        <span className="text-sm font-medium text-slate-900">{value}</span>
        {action}
      </dd>
    </div>
  )
}

function groupedAttributes(defs: UserAttributeDef[], t: DomainLabelsDictionary) {
  const groups = new Map<ReturnType<typeof attributeGroupKey>, UserAttributeDef[]>()
  for (const def of defs) {
    const key = attributeGroupKey(def)
    groups.set(key, [...(groups.get(key) ?? []), def])
  }
  return (['profile', 'organization', 'custom'] as const)
    .map((key) => ({ key, title: attributeGroupTitle(key, t), defs: groups.get(key) ?? [] }))
    .filter((group) => group.defs.length > 0)
}

function ProfileAttributeGroups({
  defs,
  values,
}: {
  defs: UserAttributeDef[]
  values: AccountProfile['attributes']
}) {
  const accountT = useDictionary(accountProfileDictionary)
  const knownKeys = new Set(defs.map((def) => def.key))
  const readOnlyDefs: UserAttributeDef[] = Object.entries(values)
    .filter(([key]) => !knownKeys.has(key))
    .map(([key, value]) => ({
      key,
      type: value.type,
      multi_valued: value.type === 'string_array',
      required: false,
      editable_by_user: false,
      visibility: 'self_readable',
      pii: false,
    }))
  const t = useDictionary(domainLabelsDictionary)
  const groups = groupedAttributes([...defs, ...readOnlyDefs], t)
  if (groups.length === 0) return null
  return (
    <>
      {groups.map((group) => (
        <section key={group.key} className="grid gap-2">
          <h3 className="text-xs font-bold uppercase tracking-normal text-slate-500">
            {group.title}
          </h3>
          <dl className="grid gap-3 sm:grid-cols-2">
            {group.defs.map((def) => (
              <ReadField
                key={def.key}
                label={attributeLabel(def)}
                value={
                  values[def.key] ? valueToDisplayText(values[def.key], accountT) : accountT.notSet
                }
              />
            ))}
          </dl>
        </section>
      ))}
    </>
  )
}

function EditableAttributeGroups({
  defs,
  values,
  onChange,
}: {
  defs: UserAttributeDef[]
  values: AttributeDraft
  onChange: (key: string, next: string) => void
}) {
  const accountT = useDictionary(accountProfileDictionary)
  const t = useDictionary(domainLabelsDictionary)
  const groups = groupedAttributes(defs, t)
  if (groups.length === 0) return null
  return (
    <div className="grid gap-4 rounded-lg border border-slate-200 p-4">
      <p className="text-sm font-medium text-slate-700">{accountT.additional}</p>
      {groups.map((group) => (
        <fieldset
          key={group.key}
          className="grid gap-3 border-t border-slate-100 pt-4 first:border-t-0 first:pt-0"
        >
          <legend className="text-xs font-bold uppercase tracking-normal text-slate-500">
            {group.title}
          </legend>
          {group.defs.map((def) => (
            <AttributeField
              key={def.key}
              def={def}
              value={values[def.key] ?? ''}
              onChange={(next) => onChange(def.key, next)}
            />
          ))}
        </fieldset>
      ))}
    </div>
  )
}

function valueToDisplayText(value: AttributeValue, t: typeof accountProfileDictionary.ja): string {
  const text = valueToText(value)
  if (value.type === 'boolean') return text === 'true' ? t.yes : t.no
  return text || t.notSet
}

function AttributeField({
  def,
  value,
  onChange,
}: {
  def: UserAttributeDef
  value: string
  onChange: (next: string) => void
}) {
  const t = useDictionary(accountProfileDictionary)
  const id = `attr-${def.key}`
  if (def.type === 'boolean') {
    return (
      <label htmlFor={id} className="inline-flex items-center gap-2 text-sm text-slate-700">
        <input
          id={id}
          type="checkbox"
          checked={value === 'true'}
          onChange={(event) => onChange(event.target.checked ? 'true' : 'false')}
          className="h-4 w-4 rounded border-slate-300"
        />
        {attributeLabel(def)}
      </label>
    )
  }
  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id}>{attributeLabel(def)}</Label>
      <Input
        id={id}
        type={def.type === 'number' ? 'number' : def.type === 'date' ? 'date' : 'text'}
        value={value}
        placeholder={def.type === 'string_array' ? t.commaSeparated : undefined}
        onChange={(event) => onChange(event.target.value)}
      />
      {def.type === 'string_array' ? <p className="text-xs text-slate-500">{t.commaHelp}</p> : null}
    </div>
  )
}

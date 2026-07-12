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
  const [notice, setNotice] = useState(() => {
    return new URLSearchParams(window.location.search).get('notice') === 'success'
      ? 'プロフィールを更新しました。'
      : ''
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
  return (
    <AccountShell
      active="profile"
      username={profile.preferred_username}
      isAdmin={isAdmin}
      title="アカウント情報"
      description="登録されているプロフィール情報を確認できます。"
    >
      <div className="grid gap-6">
        <Toast message={notice} onDismiss={onDismissNotice} />

        <Card className="p-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h2 className="text-base font-semibold text-slate-900">プロフィール</h2>
              <p className="mt-1 text-sm text-slate-600">登録されているアカウント基本情報です。</p>
            </div>
            <Button asChild variant="outline">
              <a href="/account/profile/edit">編集</a>
            </Button>
          </div>

          <dl className="mt-5 grid gap-3 sm:grid-cols-2">
            <ReadField label="表示名" value={profile.name ?? '未設定'} />
            <ReadField label="名" value={profile.given_name ?? '未設定'} />
            <ReadField label="姓" value={profile.family_name ?? '未設定'} />
            <ReadField
              label="メール"
              value={profile.email ?? '未設定'}
              action={
                <a
                  href="/account/emails"
                  className="text-xs font-semibold text-blue-600 hover:text-blue-700 hover:underline"
                >
                  変更する
                </a>
              }
            />
            <ReadField label="メール確認" value={profile.email_verified ? '確認済み' : '未確認'} />
            <ReadField label="MFA" value={profile.mfa_enrolled ? '登録済み' : '未登録'} />
            <ReadField label="状態" value={profile.status} />
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
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : 'プロフィールを更新できませんでした。',
      )
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
  return (
    <AccountShell
      active="profile"
      username={profile.preferred_username}
      isAdmin={isAdmin}
      title="プロフィールを編集"
      description="プロフィール情報を更新します。"
    >
      <div className="grid gap-6">
        {error ? <Alert variant="destructive">{error}</Alert> : null}

        <Card className="p-5">
          <div className="flex items-center gap-3">
            <a
              href="/account/profile"
              className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
              aria-label="アカウント情報に戻る"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <title>戻る</title>
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10 19l-7-7m0 0l7-7m-7 7h18"
                />
              </svg>
            </a>
            <h2 className="text-base font-semibold text-slate-900">プロフィールの編集</h2>
          </div>

          <form onSubmit={onSubmit} className="mt-5 grid gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="name">表示名</Label>
              <Input
                id="name"
                value={name}
                onChange={(event) => onNameChange(event.target.value)}
              />
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="given-name">名 (given_name)</Label>
                <Input
                  id="given-name"
                  value={givenName}
                  onChange={(event) => onGivenNameChange(event.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="family-name">姓 (family_name)</Label>
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
                {saving ? '保存中…' : '保存'}
              </Button>
              <Button type="button" variant="ghost" disabled={saving} asChild>
                <a href="/account/profile">キャンセル</a>
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
                value={values[def.key] ? valueToDisplayText(values[def.key]) : '未設定'}
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
  const t = useDictionary(domainLabelsDictionary)
  const groups = groupedAttributes(defs, t)
  if (groups.length === 0) return null
  return (
    <div className="grid gap-4 rounded-lg border border-slate-200 p-4">
      <p className="text-sm font-medium text-slate-700">追加項目</p>
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

function valueToDisplayText(value: AttributeValue): string {
  const text = valueToText(value)
  if (value.type === 'boolean') return text === 'true' ? 'はい' : 'いいえ'
  return text || '未設定'
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
        placeholder={def.type === 'string_array' ? 'カンマ区切り' : undefined}
        onChange={(event) => onChange(event.target.value)}
      />
      {def.type === 'string_array' ? (
        <p className="text-xs text-slate-500">複数値はカンマ区切りで入力します。</p>
      ) : null}
    </div>
  )
}

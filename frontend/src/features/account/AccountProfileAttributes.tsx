import type { ReactNode } from 'react'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import {
  domainLabelsDictionary,
  type DomainLabelsDictionary,
} from '../../lib/i18n/domainLabels.i18n'
import { attributeGroupKey, attributeGroupTitle, attributeLabel } from '../../lib/utils'
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

export function ProfileReadField({
  label,
  value,
  action,
}: {
  label: string
  value: string
  action?: ReactNode
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

export function ProfileAttributeGroups({
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
              <ProfileReadField
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

export function EditableAttributeGroups({
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

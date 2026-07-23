import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { domainLabelsDictionary } from '../../lib/i18n/domainLabels.i18n'
import { attributeLabel } from '../../lib/utils'
import type { AdminUser, AttributeValue, UserAttributeDef } from '../../types'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
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

export function attributeDraftFromUser(
  user: AdminUser,
  defs: UserAttributeDef[],
): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const def of defs) {
    const value = user.attributes?.[def.key]
    draft[def.key] = value ? attributeValueToText(value) : ''
  }
  return draft
}

export function attributeMapFromDraft(
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

export function AdminAttributeField({
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

export function AdminAttributeEditorGroups({
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

import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AdminGroup, AttributeValue, GroupAttributeDef } from '../../types'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

// AttributeValue <-> text draft の変換は admin-users の AdminUserAttributeEditor と
// 同じ形だが、GroupAttributeDef は editable_by_user / visibility を持たないため
// 別実装として duplicate する (frontend の feature 間 import は他箇所でも行っていない)。
function attributeValueToText(value: AttributeValue): string {
  switch (value.type) {
    case 'boolean':
      return value.boolean ? 'true' : 'false'
    case 'number':
      return value.number !== undefined ? String(value.number) : ''
    case 'date':
      return value.date ?? ''
    case 'string_array':
      return (value.string_array ?? []).join(', ')
    default:
      return value.string ?? ''
  }
}

function textToAttributeValue(def: GroupAttributeDef, text: string): AttributeValue | undefined {
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

export function groupAttributeDraft(
  group: Pick<AdminGroup, 'attributes'>,
  defs: GroupAttributeDef[],
): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const def of defs) {
    const value = group.attributes?.[def.key]
    draft[def.key] = value ? attributeValueToText(value) : ''
  }
  return draft
}

export function groupAttributeMapFromDraft(
  draft: Record<string, string>,
  defs: GroupAttributeDef[],
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

function AdminGroupAttributeField({
  def,
  value,
  onChange,
}: {
  def: GroupAttributeDef
  value: string
  onChange: (next: string) => void
}) {
  const id = `group-editor-attr-${def.key}`
  const label = def.label || def.key
  const t = useDictionary(adminGroupsDictionary)
  if (def.type === 'boolean') {
    return (
      <label htmlFor={id} className="inline-flex items-center gap-2 text-sm text-slate-700">
        <input
          id={id}
          type="checkbox"
          checked={value === 'true'}
          onChange={(event) => onChange(event.target.checked ? 'true' : 'false')}
          className="size-4 rounded border-slate-300"
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
      />
    </div>
  )
}

export function AdminGroupAttributeEditor({
  defs,
  values,
  onChange,
}: {
  defs: GroupAttributeDef[]
  values: Record<string, string>
  onChange: (key: string, next: string) => void
}) {
  const t = useDictionary(adminGroupsDictionary)
  if (defs.length === 0) return null
  return (
    <section className="grid gap-3 border-t border-slate-200 pt-5">
      <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
        {t.customAttributesHeading}
      </h3>
      <div className="grid gap-3 sm:grid-cols-2">
        {defs.map((def) => (
          <AdminGroupAttributeField
            key={def.key}
            def={def}
            value={values[def.key] ?? ''}
            onChange={(next) => onChange(def.key, next)}
          />
        ))}
      </div>
    </section>
  )
}

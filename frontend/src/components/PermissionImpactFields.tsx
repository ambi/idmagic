import { useDictionary } from '../lib/i18n'
import type { PermissionImpact } from '../types'
import { Label } from './ui/label'
import { permissionImpactDictionary } from './PermissionImpact.i18n'

export function PermissionImpactFields({
  id,
  label,
  value,
  onChange,
}: {
  id: string
  label?: string
  value?: PermissionImpact | null
  onChange: (value: PermissionImpact | null) => void
}) {
  const t = useDictionary(permissionImpactDictionary)
  return (
    <fieldset className="grid gap-2 rounded-md border p-3">
      <legend className="px-1 text-sm font-medium">{label ?? t.impact}</legend>
      <Label htmlFor={`${id}-changes`}>{t.changes}</Label>
      <select
        id={`${id}-changes`}
        className="rounded-md border p-2 text-sm"
        value={value?.changes ?? ''}
        onChange={(event) =>
          onChange(
            event.target.value
              ? {
                  changes: event.target.value as PermissionImpact['changes'],
                  personal_data: value?.personal_data ?? true,
                }
              : null,
          )
        }
      >
        <option value="">{t.unknown}</option>
        <option value="none">{t.none}</option>
        <option value="reversible">{t.reversible}</option>
        <option value="irreversible">{t.irreversible}</option>
      </select>
      {value ? (
        <>
          <Label htmlFor={`${id}-personal-data`}>{t.personalLabel}</Label>
          <select
            id={`${id}-personal-data`}
            className="rounded-md border p-2 text-sm"
            value={String(value.personal_data)}
            onChange={(event) =>
              onChange({ ...value, personal_data: event.target.value === 'true' })
            }
          >
            <option value="true">{t.yes}</option>
            <option value="false">{t.no}</option>
          </select>
        </>
      ) : null}
    </fieldset>
  )
}

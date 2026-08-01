import { tenantURL } from '../../api'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import type { AttributeType, AttrVisibility, UserAttributeDef } from '../../types'
import type { AdminTenantAttributesDictionary } from './AdminTenantAttributesPage.i18n'

export const ATTRIBUTE_TYPES: AttributeType[] = [
  'string',
  'number',
  'boolean',
  'date',
  'string_array',
]
export const VISIBILITIES: AttrVisibility[] = [
  'private',
  'self_readable',
  'admin_readable',
  'claim_exposed',
]

export function visibilityLabel(
  visibility: AttrVisibility,
  t: AdminTenantAttributesDictionary,
): string {
  return {
    private: t.visibilityPrivate,
    self_readable: t.visibilitySelfReadable,
    admin_readable: t.visibilityAdminReadable,
    claim_exposed: t.visibilityClaimExposed,
  }[visibility]
}

export function newAttribute(): UserAttributeDef {
  return {
    key: '',
    label: '',
    type: 'string',
    multi_valued: false,
    required: false,
    editable_by_user: false,
    visibility: 'admin_readable',
    pii: true,
  }
}

export function normalizeAttribute(draft: UserAttributeDef): UserAttributeDef {
  return {
    ...draft,
    key: draft.key.trim(),
    label: draft.label?.trim() || undefined,
    multi_valued: draft.type === 'string_array',
    claim_name: draft.claim_name?.trim() || undefined,
    oidc_scope: draft.oidc_scope?.trim() || undefined,
  }
}

export function listURL(): string {
  return tenantURL('/admin/tenant/attributes')
}

export function newURL(): string {
  return tenantURL('/admin/tenant/attributes/new')
}

export function Toggle({
  id,
  label,
  checked,
  onChange,
}: {
  id: string
  label: string
  checked: boolean
  onChange: (next: boolean) => void
}) {
  return (
    <label htmlFor={id} className="inline-flex items-center gap-2 text-sm text-slate-700">
      <input
        id={id}
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="h-4 w-4 rounded border-slate-300"
      />
      {label}
    </label>
  )
}

// AttributeFormFields は作成/編集で共有するフォーム本体。
export function AttributeFormFields({
  draft,
  onChange,
  t,
}: {
  draft: UserAttributeDef
  onChange: (change: Partial<UserAttributeDef>) => void
  t: AdminTenantAttributesDictionary
}) {
  const keyInvalid = draft.key.trim() === ''
  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid gap-1.5 sm:col-span-2">
          <Label htmlFor="attr-label">{t.displayNameFieldLabel}</Label>
          <Input
            id="attr-label"
            value={draft.label ?? ''}
            placeholder={t.displayNamePlaceholder}
            onChange={(event) => onChange({ label: event.target.value })}
          />
          <p className="text-xs text-slate-500">{t.displayNameHelp}</p>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="attr-key">{t.keyFieldLabel}</Label>
          <Input
            id="attr-key"
            value={draft.key}
            placeholder="region"
            className="font-mono"
            aria-invalid={keyInvalid}
            onChange={(event) => onChange({ key: event.target.value })}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="attr-type">{t.tableHeaderType}</Label>
          <select
            id="attr-type"
            value={draft.type}
            onChange={(event) => onChange({ type: event.target.value as AttributeType })}
            className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
          >
            {ATTRIBUTE_TYPES.map((type) => (
              <option key={type} value={type}>
                {type}
              </option>
            ))}
          </select>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="attr-visibility">{t.tableHeaderVisibility}</Label>
          <select
            id="attr-visibility"
            value={draft.visibility}
            onChange={(event) => onChange({ visibility: event.target.value as AttrVisibility })}
            className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
          >
            {VISIBILITIES.map((v) => (
              <option key={v} value={v}>
                {visibilityLabel(v, t)}
              </option>
            ))}
          </select>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="attr-claim">{t.claimNameFieldLabel}</Label>
          <Input
            id="attr-claim"
            value={draft.claim_name ?? ''}
            placeholder={t.claimNamePlaceholder}
            className="font-mono"
            onChange={(event) => onChange({ claim_name: event.target.value })}
          />
        </div>
        <div className="grid gap-1.5 sm:col-span-2">
          <Label htmlFor="attr-scope">{t.oidcScopeFieldLabel}</Label>
          <Input
            id="attr-scope"
            value={draft.oidc_scope ?? ''}
            placeholder={t.oidcScopePlaceholder}
            className="font-mono"
            onChange={(event) => onChange({ oidc_scope: event.target.value })}
          />
        </div>
      </div>
      <div className="mt-5 flex flex-wrap items-center gap-x-5 gap-y-2 border-t border-slate-100 pt-5">
        <Toggle
          id="attr-required"
          label={t.requiredToggle}
          checked={draft.required}
          onChange={(checked) => onChange({ required: checked })}
        />
        <Toggle
          id="attr-editable"
          label={t.editableByUserToggle}
          checked={draft.editable_by_user}
          onChange={(checked) => onChange({ editable_by_user: checked })}
        />
        <Toggle
          id="attr-pii"
          label={t.piiToggle}
          checked={draft.pii}
          onChange={(checked) => onChange({ pii: checked })}
        />
      </div>
    </>
  )
}

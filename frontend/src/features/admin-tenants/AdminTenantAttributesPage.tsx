import { IconPencil, IconPlus, IconTrash, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, updateTenantUserAttributeSchema } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type {
  AttributeType,
  AttrVisibility,
  UserAttributeDef,
  TenantUserAttributeSchema,
} from '../../types'
import {
  adminTenantAttributesDictionary,
  type AdminTenantAttributesDictionary,
} from './AdminTenantAttributesPage.i18n'

const ATTRIBUTE_TYPES: AttributeType[] = ['string', 'number', 'boolean', 'date', 'string_array']
const VISIBILITIES: AttrVisibility[] = [
  'private',
  'self_readable',
  'admin_readable',
  'claim_exposed',
]

function visibilityLabel(visibility: AttrVisibility, t: AdminTenantAttributesDictionary): string {
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

// editing は追加 (index === null) か既存行の編集 (index >= 0)。
type EditingState = { index: number | null; draft: UserAttributeDef } | null

export function AdminTenantAttributesPage({
  csrfToken,
  actorUsername,
  schema,
}: {
  csrfToken: string
  actorUsername?: string
  schema: TenantUserAttributeSchema
}) {
  const [attributes, setAttributes] = useState<UserAttributeDef[]>(schema.attributes)
  const [editing, setEditing] = useState<EditingState>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminTenantAttributesDictionary)

  // persist は custom 定義一覧を全置換で保存し、成功したらサーバ正規化後の値で更新する。
  async function persist(next: UserAttributeDef[], success: string) {
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const result = await updateTenantUserAttributeSchema(csrfToken, next)
      setAttributes(result.attributes)
      setNotice(success)
      return true
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
      return false
    } finally {
      setSaving(false)
    }
  }

  async function handleSubmit(draft: UserAttributeDef, index: number | null) {
    const cleaned = normalizeAttribute(draft)
    const next =
      index === null
        ? [...attributes, cleaned]
        : attributes.map((def, i) => (i === index ? cleaned : def))
    const ok = await persist(next, index === null ? t.addedNotice : t.updatedNotice)
    if (ok) setEditing(null)
  }

  async function handleDelete(index: number) {
    await persist(
      attributes.filter((_, i) => i !== index),
      t.deletedNotice,
    )
  }

  return (
    <AdminShell
      active="tenant-attributes"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button type="button" onClick={() => setEditing({ index: null, draft: newAttribute() })}>
          <IconPlus size={16} stroke={1.8} aria-hidden="true" />
          <span className="ml-1">{t.addAttribute}</span>
        </Button>
      }
    >
      <div className="grid gap-6">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />

        <Card className="overflow-hidden">
          <div className="border-b border-slate-200 p-5">
            <h2 className="text-base font-semibold text-slate-900">{t.customAttributesHeading}</h2>
            <p className="mt-1 text-sm text-slate-600">{t.customAttributesDescription}</p>
          </div>
          {attributes.length === 0 ? (
            <p className="px-5 py-10 text-center text-sm text-slate-500">
              {t.noCustomAttributesNotice}
            </p>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="px-5 py-3">{t.tableHeaderAttribute}</th>
                  <th className="px-5 py-3">{t.tableHeaderType}</th>
                  <th className="px-5 py-3">{t.tableHeaderVisibility}</th>
                  <th className="px-5 py-3">{t.tableHeaderSelfEditable}</th>
                  <th className="px-5 py-3" />
                </tr>
              </thead>
              <tbody>
                {attributes.map((def, index) => (
                  <tr key={def.key} className="border-t border-slate-100">
                    <td className="px-5 py-3">
                      <div className="text-slate-800">{def.label || def.key}</div>
                      {def.label ? (
                        <div className="font-mono text-xs text-slate-500">{def.key}</div>
                      ) : null}
                    </td>
                    <td className="px-5 py-3 text-slate-600">{def.type}</td>
                    <td className="px-5 py-3 text-slate-600">
                      {visibilityLabel(def.visibility, t)}
                    </td>
                    <td className="px-5 py-3 text-slate-600">
                      {def.editable_by_user ? t.yes : t.no}
                    </td>
                    <td className="px-5 py-3">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          className="px-2.5"
                          aria-label={t.editAttributeAria.replace('{key}', def.key)}
                          disabled={saving}
                          onClick={() => setEditing({ index, draft: def })}
                        >
                          <IconPencil size={15} aria-hidden="true" />
                        </Button>
                        <Button
                          variant="ghost"
                          className="px-2.5 text-rose-700 hover:bg-rose-50"
                          aria-label={t.deleteAttributeAria.replace('{key}', def.key)}
                          disabled={saving}
                          onClick={() => void handleDelete(index)}
                        >
                          <IconTrash size={15} aria-hidden="true" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>

        <BuiltinReference builtin={schema.builtin} />
      </div>

      {editing ? (
        <AttributeEditorDialog
          initial={editing.draft}
          isNew={editing.index === null}
          saving={saving}
          onClose={() => setEditing(null)}
          onSubmit={(draft) => void handleSubmit(draft, editing.index)}
        />
      ) : null}
    </AdminShell>
  )
}

function AttributeEditorDialog({
  initial,
  isNew,
  saving,
  onClose,
  onSubmit,
}: {
  initial: UserAttributeDef
  isNew: boolean
  saving: boolean
  onClose: () => void
  onSubmit: (draft: UserAttributeDef) => void
}) {
  const [draft, setDraft] = useState<UserAttributeDef>(initial)
  const keyInvalid = draft.key.trim() === ''
  const t = useDictionary(adminTenantAttributesDictionary)

  function patch(change: Partial<UserAttributeDef>) {
    setDraft((current) => ({ ...current, ...change }))
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/30 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="attribute-editor-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative flex max-h-[88vh] w-full max-w-lg flex-col overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <h2 id="attribute-editor-title" className="text-xl font-semibold">
            {isNew ? t.addAttribute : t.editAttributeTitle}
          </h2>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>
        <form
          onSubmit={(event) => {
            event.preventDefault()
            if (!keyInvalid) onSubmit(draft)
          }}
          className="flex min-h-0 flex-1 flex-col"
        >
          <div className="min-h-0 flex-1 overflow-y-auto p-6">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5 sm:col-span-2">
                <Label htmlFor="attr-label">{t.displayNameFieldLabel}</Label>
                <Input
                  id="attr-label"
                  value={draft.label ?? ''}
                  placeholder={t.displayNamePlaceholder}
                  onChange={(event) => patch({ label: event.target.value })}
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
                  onChange={(event) => patch({ key: event.target.value })}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="attr-type">{t.tableHeaderType}</Label>
                <select
                  id="attr-type"
                  value={draft.type}
                  onChange={(event) => patch({ type: event.target.value as AttributeType })}
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
                  onChange={(event) => patch({ visibility: event.target.value as AttrVisibility })}
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
                  onChange={(event) => patch({ claim_name: event.target.value })}
                />
              </div>
              <div className="grid gap-1.5 sm:col-span-2">
                <Label htmlFor="attr-scope">{t.oidcScopeFieldLabel}</Label>
                <Input
                  id="attr-scope"
                  value={draft.oidc_scope ?? ''}
                  placeholder={t.oidcScopePlaceholder}
                  className="font-mono"
                  onChange={(event) => patch({ oidc_scope: event.target.value })}
                />
              </div>
            </div>
            <div className="mt-5 flex flex-wrap items-center gap-x-5 gap-y-2 border-t border-slate-100 pt-5">
              <Toggle
                id="attr-required"
                label={t.requiredToggle}
                checked={draft.required}
                onChange={(checked) => patch({ required: checked })}
              />
              <Toggle
                id="attr-editable"
                label={t.editableByUserToggle}
                checked={draft.editable_by_user}
                onChange={(checked) => patch({ editable_by_user: checked })}
              />
              <Toggle
                id="attr-pii"
                label={t.piiToggle}
                checked={draft.pii}
                onChange={(checked) => patch({ pii: checked })}
              />
            </div>
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button type="button" variant="outline" onClick={onClose}>
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || keyInvalid}>
              {saving ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}

function Toggle({
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

function BuiltinReference({ builtin }: { builtin: UserAttributeDef[] }) {
  const t = useDictionary(adminTenantAttributesDictionary)
  return (
    <Card className="p-6">
      <h2 className="text-base font-semibold text-slate-900">{t.builtinAttributesHeading}</h2>
      <p className="mt-1 text-sm text-slate-600">{t.builtinAttributesDescription}</p>
      <div className="mt-4 overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-500">
              <th className="py-2 pr-4 font-medium">{t.displayNameFieldLabel}</th>
              <th className="py-2 pr-4 font-medium">{t.keyFieldLabel}</th>
              <th className="py-2 pr-4 font-medium">{t.tableHeaderType}</th>
              <th className="py-2 pr-4 font-medium">{t.tableHeaderVisibility}</th>
              <th className="py-2 pr-4 font-medium">{t.tableHeaderScope}</th>
            </tr>
          </thead>
          <tbody>
            {builtin.map((def) => (
              <tr key={def.key} className="border-b border-slate-100">
                <td className="py-2 pr-4 text-slate-800">{def.label || '—'}</td>
                <td className="py-2 pr-4 font-mono text-slate-600">{def.key}</td>
                <td className="py-2 pr-4 text-slate-600">{def.type}</td>
                <td className="py-2 pr-4 text-slate-600">{visibilityLabel(def.visibility, t)}</td>
                <td className="py-2 pr-4 font-mono text-slate-500">{def.oidc_scope ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}

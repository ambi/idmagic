import { IconPencil, IconPlus, IconTrash, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, updateTenantGroupAttributeSchema } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AttributeType, GroupAttributeDef, TenantGroupAttributeSchema } from '../../types'
import { adminTenantGroupAttributesDictionary } from './AdminTenantGroupAttributesPage.i18n'

const ATTRIBUTE_TYPES: AttributeType[] = ['string', 'number', 'boolean', 'date', 'string_array']

function newAttribute(): GroupAttributeDef {
  return { key: '', label: '', type: 'string', multi_valued: false, required: false }
}

function normalizeAttribute(draft: GroupAttributeDef): GroupAttributeDef {
  return {
    ...draft,
    key: draft.key.trim(),
    label: draft.label?.trim() || undefined,
    multi_valued: draft.type === 'string_array',
  }
}

export function AdminTenantGroupAttributesPage({
  csrfToken,
  actorUsername,
  schema,
}: {
  csrfToken: string
  actorUsername?: string
  schema: TenantGroupAttributeSchema
}) {
  const [attributes, setAttributes] = useState<GroupAttributeDef[]>(schema.attributes)
  // editingIndex は既存属性の編集対象 index、'new' は追加ダイアログを開いている状態。
  const [editingIndex, setEditingIndex] = useState<number | 'new' | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminTenantGroupAttributesDictionary)

  async function persist(next: GroupAttributeDef[], success: string) {
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const result = await updateTenantGroupAttributeSchema(csrfToken, next)
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

  async function handleSubmit(draft: GroupAttributeDef, index: number | 'new') {
    const cleaned = normalizeAttribute(draft)
    const next =
      index === 'new'
        ? [...attributes, cleaned]
        : attributes.map((d, i) => (i === index ? cleaned : d))
    const ok = await persist(next, index === 'new' ? t.addedNotice : t.updatedNotice)
    if (ok) setEditingIndex(null)
  }

  async function handleDelete(index: number) {
    await persist(
      attributes.filter((_, i) => i !== index),
      t.deletedNotice,
    )
  }

  return (
    <AdminShell
      active="tenant-group-attributes"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button onClick={() => setEditingIndex('new')}>
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
                  <th className="px-5 py-3">{t.tableHeaderRequired}</th>
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
                    <td className="px-5 py-3 text-slate-600">{def.required ? t.yes : t.no}</td>
                    <td className="px-5 py-3">
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="outline"
                          aria-label={t.editAttributeAria.replace('{key}', def.key)}
                          disabled={saving}
                          onClick={() => setEditingIndex(index)}
                        >
                          <IconPencil size={15} aria-hidden="true" />
                          {t.edit}
                        </Button>
                        <Button
                          variant="destructive"
                          aria-label={t.deleteAttributeAria.replace('{key}', def.key)}
                          disabled={saving}
                          onClick={() => void handleDelete(index)}
                        >
                          <IconTrash size={15} aria-hidden="true" />
                          {t.delete}
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
      </div>

      {editingIndex !== null ? (
        <AttributeEditorDialog
          initial={editingIndex === 'new' ? newAttribute() : attributes[editingIndex]}
          isNew={editingIndex === 'new'}
          saving={saving}
          onClose={() => setEditingIndex(null)}
          onSubmit={(draft) => void handleSubmit(draft, editingIndex)}
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
  initial: GroupAttributeDef
  isNew: boolean
  saving: boolean
  onClose: () => void
  onSubmit: (draft: GroupAttributeDef) => void
}) {
  const [draft, setDraft] = useState<GroupAttributeDef>(initial)
  const keyInvalid = draft.key.trim() === ''
  const t = useDictionary(adminTenantGroupAttributesDictionary)

  function patch(change: Partial<GroupAttributeDef>) {
    setDraft((current) => ({ ...current, ...change }))
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/30 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="group-attribute-editor-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative flex max-h-[88vh] w-full max-w-lg flex-col overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <h2 id="group-attribute-editor-title" className="text-xl font-semibold">
            {isNew ? t.addAttributeTitle : t.editAttributeTitle}
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
                <Label htmlFor="group-attr-label">{t.displayNameFieldLabel}</Label>
                <Input
                  id="group-attr-label"
                  value={draft.label ?? ''}
                  placeholder={t.displayNamePlaceholder}
                  onChange={(event) => patch({ label: event.target.value })}
                />
                <p className="text-xs text-slate-500">{t.displayNameHelp}</p>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="group-attr-key">{t.keyFieldLabel}</Label>
                <Input
                  id="group-attr-key"
                  value={draft.key}
                  placeholder="cost_center"
                  className="font-mono"
                  aria-invalid={keyInvalid}
                  onChange={(event) => patch({ key: event.target.value })}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="group-attr-type">{t.tableHeaderType}</Label>
                <select
                  id="group-attr-type"
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
            </div>
            <div className="mt-5 flex flex-wrap items-center gap-x-5 gap-y-2 border-t border-slate-100 pt-5">
              <label
                htmlFor="group-attr-required"
                className="inline-flex items-center gap-2 text-sm text-slate-700"
              >
                <input
                  id="group-attr-required"
                  type="checkbox"
                  checked={draft.required}
                  onChange={(event) => patch({ required: event.target.checked })}
                  className="h-4 w-4 rounded border-slate-300"
                />
                {t.requiredToggle}
              </label>
            </div>
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button type="button" variant="outline" onClick={onClose}>
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || keyInvalid}>
              {saving ? t.saving : isNew ? t.add : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}

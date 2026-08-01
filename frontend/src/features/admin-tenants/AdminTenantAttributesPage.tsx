import { IconPencil, IconPlus, IconTrash, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, updateTenantUserAttributeSchema } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { UserAttributeDef, TenantUserAttributeSchema } from '../../types'
import { adminTenantAttributesDictionary } from './AdminTenantAttributesPage.i18n'
import {
  AttributeFormFields,
  newURL,
  normalizeAttribute,
  visibilityLabel,
} from './AdminTenantAttributesShared'

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
  const [editingIndex, setEditingIndex] = useState<number | null>(null)
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

  async function handleSubmit(draft: UserAttributeDef, index: number) {
    const cleaned = normalizeAttribute(draft)
    const next = attributes.map((def, i) => (i === index ? cleaned : def))
    const ok = await persist(next, t.updatedNotice)
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
      active="tenant-attributes"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button nativeButton={false} render={<a href={newURL()} />}>
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

        <BuiltinReference builtin={schema.builtin} />
      </div>

      {editingIndex !== null ? (
        <AttributeEditorDialog
          initial={attributes[editingIndex]}
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
  saving,
  onClose,
  onSubmit,
}: {
  initial: UserAttributeDef
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
            {t.editAttributeTitle}
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
            <AttributeFormFields draft={draft} onChange={patch} t={t} />
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

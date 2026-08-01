import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateTenantUserAttributeSchema } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { UserAttributeDef } from '../../types'
import { adminTenantAttributesDictionary } from './AdminTenantAttributesPage.i18n'
import {
  AttributeFormFields,
  listURL,
  newAttribute,
  normalizeAttribute,
} from './AdminTenantAttributesShared'

// AdminTenantAttributeCreatePage はユーザー属性追加の専用画面 (wi-314 T015)。
// カスタム属性は単一のテナントスキーマ配列として保存されるため、既存の
// attributes 配列に新規属性を追加した全体を PATCH する。
export function AdminTenantAttributeCreatePage({
  csrfToken,
  actorUsername,
  existingAttributes,
}: {
  csrfToken: string
  actorUsername?: string
  existingAttributes: UserAttributeDef[]
}) {
  const [draft, setDraft] = useState<UserAttributeDef>(newAttribute())
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminTenantAttributesDictionary)
  const keyInvalid = draft.key.trim() === ''

  function patch(change: Partial<UserAttributeDef>) {
    setDraft((current) => ({ ...current, ...change }))
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (keyInvalid) return
    setSaving(true)
    setError('')
    try {
      await updateTenantUserAttributeSchema(csrfToken, [
        ...existingAttributes,
        normalizeAttribute(draft),
      ])
      window.location.assign(listURL())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="tenant-attributes"
      actorUsername={actorUsername}
      title={t.addAttribute}
      description={t.pageDescription}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full max-w-2xl p-6">
        <form onSubmit={handleSubmit}>
          <AttributeFormFields draft={draft} onChange={patch} t={t} />
          <div className="mt-5 flex justify-end gap-2">
            <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || keyInvalid}>
              {saving ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

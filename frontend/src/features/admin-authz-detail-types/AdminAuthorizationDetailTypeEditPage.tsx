import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateAuthorizationDetailType } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { AuthorizationDetailType } from '../../types'
import { adminAuthorizationDetailTypesDictionary } from './AdminAuthorizationDetailTypesPage.i18n'
import {
  AuthorizationDetailTypeFormFields,
  listURL,
  parseSchemaObject,
  toForm,
  useDetailTypeValidation,
} from './AdminAuthorizationDetailTypesShared'

// AdminAuthorizationDetailTypeEditPage は認可詳細タイプ編集の専用画面 (wi-314 T014)。
export function AdminAuthorizationDetailTypeEditPage({
  csrfToken,
  actorUsername,
  detailType,
}: {
  csrfToken: string
  actorUsername?: string
  detailType: AuthorizationDetailType
}) {
  const [form, setForm] = useState(() => toForm(detailType))
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const t = useDictionary(adminAuthorizationDetailTypesDictionary)
  const validation = useDetailTypeValidation(form, t)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    setSaving(true)
    try {
      await updateAuthorizationDetailType(csrfToken, detailType.type, {
        type: data.type,
        description: form.description,
        display_template: data.displayTemplate,
        state: form.state,
        schema: parseSchemaObject(data.schemaJSON)!,
      })
      window.location.assign(listURL())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="authz-detail-types"
      actorUsername={actorUsername}
      title={t.editType.replace('{type}', detailType.type)}
      description={t.pageDescription}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full admin-form p-4">
        <form className="grid gap-4 md:grid-cols-2" onSubmit={handleSubmit}>
          <AuthorizationDetailTypeFormFields
            form={form}
            validation={validation}
            onChange={setForm}
            locked={true}
            t={t}
          />
          <div className="flex gap-2.5 md:col-span-2">
            <Button type="submit" disabled={saving || !validation.data}>
              {t.update}
            </Button>
            <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
              {t.cancel}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

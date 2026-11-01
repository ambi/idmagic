import { useState } from 'react'
import { getTenantGroupAttributeSchema, updateTenantGroupAttributeSchema } from '../../api/admin'
import { AuthenticationAPIError, tenantURL } from '../../api/core'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { useDictionary } from '../../lib/i18n'
import type { GroupAttributeDef } from '../../types'
import { adminTenantGroupAttributesDictionary } from './AdminTenantGroupAttributesPage.i18n'
import { AttributeEditorDialog } from './AdminTenantGroupAttributesPage'

export function AdminTenantGroupAttributeCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const t = useDictionary(adminTenantGroupAttributesDictionary)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const back = () => window.location.assign(tenantURL('/admin/tenant/group_attributes'))
  async function create(draft: GroupAttributeDef) {
    setSaving(true)
    setError('')
    try {
      const schema = await getTenantGroupAttributeSchema()
      await updateTenantGroupAttributeSchema(csrfToken, [
        ...schema.attributes,
        {
          ...draft,
          key: draft.key.trim(),
          label: draft.label?.trim() || undefined,
          multi_valued: draft.type === 'string_array',
        },
      ])
      back()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
    } finally {
      setSaving(false)
    }
  }
  return (
    <AdminShell
      active="tenant-group-attributes"
      actorUsername={actorUsername}
      title={t.addAttributeTitle}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <AttributeEditorDialog
        initial={{ key: '', type: 'string', required: false, multi_valued: false }}
        isNew
        saving={saving}
        onClose={back}
        onSubmit={(draft) => void create(draft)}
      />
    </AdminShell>
  )
}

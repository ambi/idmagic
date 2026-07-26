import { useState } from 'react'
import { AuthenticationAPIError, setAdminTenantEndpointStyle } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'

export function TenantEndpointStyleEditor({
  tenant,
  csrfToken,
  busy,
  onSaved,
}: {
  tenant: AdminTenant
  csrfToken: string
  busy: boolean
  onSaved: (id: string) => void
}) {
  const [style, setStyle] = useState<NonNullable<AdminTenant['endpoint_style']>>(
    tenant.endpoint_style ?? 'path',
  )
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(systemTenantsDictionary)
  const changed = style !== (tenant.endpoint_style ?? 'path')

  async function save() {
    if (!changed || !window.confirm(t.endpointStyleWarning)) return
    setSaving(true)
    setError('')
    try {
      await setAdminTenantEndpointStyle(csrfToken, tenant.realm, style)
      onSaved(tenant.id)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.tenantUpdateFailedError)
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="mt-5 grid gap-2 border-t border-slate-100 pt-5">
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
        {t.endpointStyleLabel}
      </p>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Label htmlFor={`endpoint-style-${tenant.id}`}>{t.endpointStyleLabel}</Label>
      <select
        id={`endpoint-style-${tenant.id}`}
        value={style}
        onChange={(event) =>
          setStyle(event.target.value as NonNullable<AdminTenant['endpoint_style']>)
        }
        className="h-9 rounded-md border border-slate-200 bg-white px-3 text-sm"
      >
        <option value="path">path</option>
        <option value="subdomain">subdomain</option>
      </select>
      <p className="text-xs text-amber-700">{t.endpointStyleHelp}</p>
      <Button
        type="button"
        onClick={save}
        disabled={busy || saving || !changed}
        className="justify-self-start"
      >
        {saving ? t.saving : t.save}
      </Button>
    </section>
  )
}

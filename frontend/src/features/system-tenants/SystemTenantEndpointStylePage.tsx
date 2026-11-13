import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, setAdminTenantEndpointStyle } from '../../api'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'
import { tenantDetailURL, tenantPath } from './SystemTenantsShared'

type EndpointStyle = NonNullable<AdminTenant['endpoint_style']>

// SystemTenantEndpointStylePage は正規ロケーションを切り替える。
// 切替は発行者、RP ID、Cookie の範囲を変えるため、保存の前に確認を経る（REQ-FRONTEND-004）。
export function SystemTenantEndpointStylePage({
  csrfToken,
  actorUsername,
  tenant,
}: {
  csrfToken: string
  actorUsername?: string
  tenant: AdminTenant
}) {
  const t = useDictionary(systemTenantsDictionary)
  const current = tenant.endpoint_style ?? 'path'
  const [style, setStyle] = useState<EndpointStyle>(current)
  const [confirming, setConfirming] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const changed = style !== current

  function requestConfirmation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (changed) setConfirming(true)
  }

  async function save() {
    setSaving(true)
    setError('')
    try {
      await setAdminTenantEndpointStyle(csrfToken, tenant.realm, style)
      window.location.assign(tenantDetailURL(tenant.realm, 'updated'))
    } catch (cause) {
      setConfirming(false)
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.tenantUpdateFailedError)
      setSaving(false)
    }
  }

  return (
    <SystemShell
      active="tenants"
      actorUsername={actorUsername}
      title={t.switchEndpointStyle}
      description={tenant.realm}
    >
      <Card className="max-w-2xl p-6">
        <form onSubmit={requestConfirmation} className="grid gap-4">
          {error ? <Alert variant="destructive">{error}</Alert> : null}
          <div className="grid gap-1.5">
            <Label htmlFor="tenant-endpoint-style">{t.endpointStyleLabel}</Label>
            <select
              id="tenant-endpoint-style"
              value={style}
              onChange={(event) => setStyle(event.target.value as EndpointStyle)}
              className="h-9 rounded-md border border-slate-200 bg-white px-3 text-sm"
            >
              <option value="path">path</option>
              <option value="subdomain">subdomain</option>
            </select>
            <p className="text-xs text-amber-700">{t.endpointStyleHelp}</p>
          </div>
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              nativeButton={false}
              render={<a href={tenantPath(tenant.realm)} />}
            >
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || !changed}>
              {saving ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>

      {confirming ? (
        <ConfirmActionDialog
          title={t.confirmEndpointStyleTitle}
          target={`${tenant.realm}: ${current} → ${style}`}
          description={t.endpointStyleWarning}
          confirmLabel={t.confirmEndpointStyle}
          busy={saving}
          onClose={() => setConfirming(false)}
          onConfirm={() => void save()}
        />
      ) : null}
    </SystemShell>
  )
}

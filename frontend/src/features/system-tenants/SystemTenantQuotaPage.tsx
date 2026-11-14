import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateAdminTenantQuota } from '../../api'
import { FieldError } from '../../components/FieldError'
import { SystemShell } from '../../components/SystemShell'
import { StepUpCancelledError, useStepUpGuard } from '../../components/StepUpDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { formSchema, optionalNonNegativeInteger, useFormValidation } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'
import {
  QUOTA_KEYS,
  QUOTA_LABELS,
  type QuotaKey,
  tenantDetailURL,
  tenantPath,
} from './SystemTenantsShared'

// 上限は TenantQuotaUpdateRequest の @minValue(0) に従う。
const quotaSchema = formSchema(
  Object.fromEntries(QUOTA_KEYS.map((key) => [key, optionalNonNegativeInteger()])) as Record<
    QuotaKey,
    ReturnType<typeof optionalNonNegativeInteger>
  >,
)

function quotaForm(tenant: AdminTenant): Record<QuotaKey, string> {
  return Object.fromEntries(
    QUOTA_KEYS.map((key) => [key, tenant.quota?.[key]?.toString() ?? '']),
  ) as Record<QuotaKey, string>
}

// SystemTenantQuotaPage は上限の上書きの全体を UpdateTenantQuota の一回の要求で置き換える。
export function SystemTenantQuotaPage({
  csrfToken,
  actorUsername,
  tenant,
}: {
  csrfToken: string
  actorUsername?: string
  tenant: AdminTenant
}) {
  const t = useDictionary(systemTenantsDictionary)
  const [quota, setQuota] = useState(() => quotaForm(tenant))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  // 制御面の変更は直近の MFA の認証を求める (REQ-SYSTEM-033)。
  const { guard, dialog } = useStepUpGuard()
  const validation = useFormValidation(quotaSchema, quota)

  async function handleSave(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    setSaving(true)
    setError('')
    try {
      await guard(() =>
        updateAdminTenantQuota(
          csrfToken,
          tenant.realm,
          Object.fromEntries(QUOTA_KEYS.map((key) => [key, data[key]])),
        ),
      )
      window.location.assign(tenantDetailURL(tenant.realm, 'updated'))
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) {
        setSaving(false)
        return
      }
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.tenantUpdateFailedError)
      setSaving(false)
    }
  }

  return (
    <SystemShell
      active="tenants"
      actorUsername={actorUsername}
      title={t.editQuota}
      description={tenant.realm}
    >
      <Card className="max-w-3xl p-6">
        <form onSubmit={(event) => void handleSave(event)} className="grid gap-4">
          {error ? <Alert variant="destructive">{error}</Alert> : null}
          <p className="text-sm text-slate-500">{t.quotaHelp}</p>
          <div className="grid gap-3 sm:grid-cols-2">
            {QUOTA_KEYS.map((key) => (
              <div key={key} className="grid gap-1.5">
                <Label htmlFor={`tenant-quota-${key}`}>{t[QUOTA_LABELS[key]]}</Label>
                <Input
                  id={`tenant-quota-${key}`}
                  type="number"
                  min={0}
                  value={quota[key]}
                  onChange={(e) => setQuota((current) => ({ ...current, [key]: e.target.value }))}
                  {...validation.fieldProps(key)}
                />
                <FieldError form={validation} name={key} />
              </div>
            ))}
          </div>
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              nativeButton={false}
              render={<a href={tenantPath(tenant.realm)} />}
            >
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || !validation.data}>
              {saving ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>
      {dialog}
    </SystemShell>
  )
}

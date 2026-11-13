import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateAdminTenant } from '../../api'
import { FieldError } from '../../components/FieldError'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import {
  formSchema,
  optionalNonNegativeInteger,
  requiredText,
  useFormValidation,
} from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'
import { tenantDetailURL, tenantPath } from './SystemTenantsShared'

// 表示名は Tenant の上限に従う。
// パスワードポリシーの下限と上限は全体の既定値で決まり、この画面は値を持たないのでサーバーが判定する。
const tenantSettingsSchema = formSchema({
  displayName: requiredText(LENGTH.displayName),
  minLength: optionalNonNegativeInteger(),
  maxLength: optionalNonNegativeInteger(),
  historyDepth: optionalNonNegativeInteger(),
})

// SystemTenantEditPage は表示名とパスワードポリシーの上書きだけを、UpdateTenant の一回の要求で保存する。
export function SystemTenantEditPage({
  csrfToken,
  actorUsername,
  tenant,
}: {
  csrfToken: string
  actorUsername?: string
  tenant: AdminTenant
}) {
  const t = useDictionary(systemTenantsDictionary)
  const [displayName, setDisplayName] = useState(tenant.display_name)
  const [minLength, setMinLength] = useState(
    tenant.password_policy_override?.min_length?.toString() ?? '',
  )
  const [maxLength, setMaxLength] = useState(
    tenant.password_policy_override?.max_length?.toString() ?? '',
  )
  const [historyDepth, setHistoryDepth] = useState(
    tenant.password_policy_override?.history_depth?.toString() ?? '',
  )
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const validation = useFormValidation(tenantSettingsSchema, {
    displayName,
    minLength,
    maxLength,
    historyDepth,
  })

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
      const policy: AdminTenant['password_policy_override'] = {}
      if (data.minLength !== undefined) policy.min_length = data.minLength
      if (data.maxLength !== undefined) policy.max_length = data.maxLength
      if (data.historyDepth !== undefined) policy.history_depth = data.historyDepth
      await updateAdminTenant(csrfToken, tenant.realm, {
        display_name: data.displayName !== tenant.display_name ? data.displayName : undefined,
        password_policy_override: Object.keys(policy).length > 0 ? policy : undefined,
      })
      window.location.assign(tenantDetailURL(tenant.realm, 'updated'))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.tenantUpdateFailedError)
      setSaving(false)
    }
  }

  return (
    <SystemShell
      active="tenants"
      actorUsername={actorUsername}
      title={t.editSettings}
      description={tenant.realm}
    >
      <Card className="max-w-2xl p-6">
        <form onSubmit={(event) => void handleSave(event)} className="grid gap-4">
          {error ? <Alert variant="destructive">{error}</Alert> : null}
          <div className="grid gap-1.5">
            <Label htmlFor="tenant-display-name" required>
              {t.displayNameLabel}
            </Label>
            <Input
              id="tenant-display-name"
              required
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              {...validation.fieldProps('displayName')}
            />
            <FieldError form={validation} name="displayName" />
          </div>
          <fieldset className="grid gap-2">
            <legend className="text-sm font-semibold text-slate-700">
              {t.passwordPolicyOverrideHeading}
            </legend>
            <p className="text-xs text-slate-500">{t.passwordPolicyOverrideHelp}</p>
            <div className="grid gap-3 sm:grid-cols-3">
              <NumberField
                id="tenant-policy-min"
                label={t.minLabel}
                min={1}
                value={minLength}
                onChange={setMinLength}
                validation={validation}
                name="minLength"
              />
              <NumberField
                id="tenant-policy-max"
                label={t.maxLabel}
                min={1}
                value={maxLength}
                onChange={setMaxLength}
                validation={validation}
                name="maxLength"
              />
              <NumberField
                id="tenant-policy-history"
                label={t.historyLabel}
                min={0}
                value={historyDepth}
                onChange={setHistoryDepth}
                validation={validation}
                name="historyDepth"
              />
            </div>
          </fieldset>
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
    </SystemShell>
  )
}

type SettingsValidation = ReturnType<typeof useFormValidation<typeof tenantSettingsSchema>>

function NumberField({
  id,
  label,
  min,
  value,
  onChange,
  validation,
  name,
}: {
  id: string
  label: string
  min: number
  value: string
  onChange: (value: string) => void
  validation: SettingsValidation
  name: 'minLength' | 'maxLength' | 'historyDepth'
}) {
  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="number"
        min={min}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        {...validation.fieldProps(name)}
      />
      <FieldError form={validation} name={name} />
    </div>
  )
}

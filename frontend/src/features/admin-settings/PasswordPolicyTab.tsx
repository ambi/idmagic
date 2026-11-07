import { IconPencil } from '@tabler/icons-react'
import { type FormEvent, type ReactNode, useMemo, useState } from 'react'
import { AuthenticationAPIError, updateAdminSettings } from '../../api'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { formSchema, integerInRange, rule, useFormValidation } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import type { AdminSettings } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { ReadSetting } from './AdminSettingsShared'

// The server rejects anything outside this range (REQ-TENANCY-019); the input
// bounds only spare the administrator a round trip.
const MAX_AGE_DAYS_MIN = 30
const MAX_AGE_DAYS_MAX = 3650

// An absent or 0 max_age_days both mean "no expiry"; an older server omits the
// field entirely.
function formatMaxAge(days: number | undefined, daysSuffix: string, noExpiry: string) {
  return days ? `${days}${daysSuffix}` : noExpiry
}

export function PasswordPolicyTab({
  csrfToken,
  settings,
  onSaved,
}: {
  csrfToken: string
  settings: AdminSettings
  onSaved: (next: AdminSettings) => void
}) {
  const override = settings.password_policy_override
  const defaults = settings.password_policy_defaults
  const [minLength, setMinLength] = useState(override?.min_length?.toString() ?? '')
  const [maxLength, setMaxLength] = useState(override?.max_length?.toString() ?? '')
  const [historyDepth, setHistoryDepth] = useState(override?.history_depth?.toString() ?? '')
  const [maxAgeDays, setMaxAgeDays] = useState(override?.max_age_days?.toString() ?? '')
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminSettingsDictionary)

  // 上書きは全体の値より厳しい方向にだけ動かせる（manage_tenants.go の validatePasswordPolicyOverride）。
  // 有効期限は 0（無効）か、システムの範囲の日数である。
  const schema = useMemo(
    () =>
      formSchema({
        minLength: integerInRange(defaults.min_length, defaults.max_length),
        maxLength: integerInRange(defaults.min_length, defaults.max_length),
        historyDepth: integerInRange(defaults.history_depth, 50),
        maxAgeDays: rule(integerInRange(0, MAX_AGE_DAYS_MAX), (value) =>
          value === undefined || value === 0 || value >= MAX_AGE_DAYS_MIN
            ? undefined
            : { kind: 'integerRange', min: MAX_AGE_DAYS_MIN, max: MAX_AGE_DAYS_MAX },
        ),
      }),
    [defaults.min_length, defaults.max_length, defaults.history_depth],
  )
  const validation = useFormValidation(schema, { minLength, maxLength, historyDepth, maxAgeDays })

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const policy: NonNullable<AdminSettings['password_policy_override']> = {}
      if (data.minLength !== undefined) policy.min_length = data.minLength
      if (data.maxLength !== undefined) policy.max_length = data.maxLength
      if (data.historyDepth !== undefined) policy.history_depth = data.historyDepth
      if (data.maxAgeDays !== undefined) policy.max_age_days = data.maxAgeDays
      const next = await updateAdminSettings(csrfToken, {
        password_policy_override: policy,
      })
      onSaved(next)
      setMinLength(next.password_policy_override?.min_length?.toString() ?? '')
      setMaxLength(next.password_policy_override?.max_length?.toString() ?? '')
      setHistoryDepth(next.password_policy_override?.history_depth?.toString() ?? '')
      setMaxAgeDays(next.password_policy_override?.max_age_days?.toString() ?? '')
      setEditing(false)
      setNotice(t.passwordPolicyUpdatedNotice)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : t.passwordPolicyUpdateFailedError,
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="p-6">
      <header>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900">{t.passwordPolicyHeading}</h2>
            <p className="mt-1 text-sm text-slate-600">{t.passwordPolicySubheading}</p>
          </div>
          {!editing ? (
            <Button type="button" variant="outline" onClick={() => setEditing(true)}>
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </Button>
          ) : null}
        </div>
        <dl className="mt-3 grid grid-cols-2 gap-3 rounded-md border border-slate-200 bg-slate-50 px-4 py-3 text-xs sm:grid-cols-4">
          <div>
            <dt className="text-slate-500">{t.standardMinLengthLabel}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {`${defaults.min_length}${t.charsSuffix}`}
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">{t.standardMaxLengthLabel}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {`${defaults.max_length}${t.charsSuffix}`}
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">{t.standardHistoryDepthLabel}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {`${defaults.history_depth}${t.countSuffix}`}
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">{t.standardMaxAgeLabel}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {formatMaxAge(defaults.max_age_days, t.daysSuffix, t.noExpiryValue)}
            </dd>
          </div>
        </dl>
        <p className="mt-2 text-xs text-slate-500">{t.weakerPolicyWarning}</p>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-4">
            <ReadSetting
              label={t.minLengthLabel}
              value={`${override?.min_length ?? defaults.min_length}${t.charsSuffix}`}
            />
            <ReadSetting
              label={t.maxLengthLabel}
              value={`${override?.max_length ?? defaults.max_length}${t.charsSuffix}`}
            />
            <ReadSetting
              label={t.historyDepthLabel}
              value={`${override?.history_depth ?? defaults.history_depth}${t.countSuffix}`}
            />
            <ReadSetting
              label={t.maxAgeLabel}
              value={formatMaxAge(
                override?.max_age_days ?? defaults.max_age_days,
                t.daysSuffix,
                t.noExpiryValue,
              )}
            />
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4">
            <div className="grid gap-4 sm:grid-cols-4">
              <PolicyField
                id="min-length"
                label={t.minLengthFieldLabel}
                value={minLength}
                onChange={setMinLength}
                min={defaults.min_length}
                max={defaults.max_length}
                placeholder={defaults.min_length.toString()}
                hint={t.atLeastHint.replace('{n}', defaults.min_length.toString())}
                fieldProps={validation.fieldProps('minLength')}
                error={<FieldError form={validation} name="minLength" />}
              />
              <PolicyField
                id="max-length"
                label={t.maxLengthFieldLabel}
                value={maxLength}
                onChange={setMaxLength}
                min={defaults.min_length}
                max={defaults.max_length}
                placeholder={defaults.max_length.toString()}
                hint={t.atMostHint.replace('{n}', defaults.max_length.toString())}
                fieldProps={validation.fieldProps('maxLength')}
                error={<FieldError form={validation} name="maxLength" />}
              />
              <PolicyField
                id="history-depth"
                label={t.historyDepthFieldLabel}
                value={historyDepth}
                onChange={setHistoryDepth}
                min={defaults.history_depth}
                max={50}
                placeholder={defaults.history_depth.toString()}
                hint={t.atLeastHint.replace('{n}', defaults.history_depth.toString())}
                fieldProps={validation.fieldProps('historyDepth')}
                error={<FieldError form={validation} name="historyDepth" />}
              />
              <PolicyField
                id="max-age-days"
                label={t.maxAgeFieldLabel}
                value={maxAgeDays}
                onChange={setMaxAgeDays}
                min={MAX_AGE_DAYS_MIN}
                max={MAX_AGE_DAYS_MAX}
                placeholder=""
                hint={t.maxAgeHint
                  .replace('{min}', MAX_AGE_DAYS_MIN.toString())
                  .replace('{max}', MAX_AGE_DAYS_MAX.toString())}
                fieldProps={validation.fieldProps('maxAgeDays')}
                error={<FieldError form={validation} name="maxAgeDays" />}
              />
            </div>
            <p className="text-xs text-slate-500">{t.maxAgeDescription}</p>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving || !validation.data}>
                {saving ? t.saving : t.save}
              </Button>
              <Button
                type="button"
                variant="ghost"
                disabled={saving}
                onClick={() => {
                  setMinLength(settings.password_policy_override?.min_length?.toString() ?? '')
                  setMaxLength(settings.password_policy_override?.max_length?.toString() ?? '')
                  setHistoryDepth(
                    settings.password_policy_override?.history_depth?.toString() ?? '',
                  )
                  setMaxAgeDays(settings.password_policy_override?.max_age_days?.toString() ?? '')
                  setEditing(false)
                }}
              >
                {t.cancel}
              </Button>
            </div>
          </form>
        )}
      </div>
    </Card>
  )
}

function PolicyField({
  id,
  label,
  value,
  onChange,
  min,
  max,
  placeholder,
  hint,
  fieldProps,
  error,
}: {
  id: string
  label: string
  value: string
  onChange: (next: string) => void
  min: number
  max: number
  placeholder: string
  hint: string
  fieldProps: { 'aria-invalid'?: true; 'aria-describedby'?: string; onBlur: () => void }
  error: ReactNode
}) {
  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="number"
        min={min}
        max={max}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
        {...fieldProps}
      />
      {error}
      <p className="text-xs text-slate-500">{hint}</p>
    </div>
  )
}

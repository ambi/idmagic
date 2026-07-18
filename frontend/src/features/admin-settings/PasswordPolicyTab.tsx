import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateAdminSettings } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { AdminSettings } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { passwordPolicyOverride, ReadSetting } from './AdminSettingsShared'

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
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminSettingsDictionary)

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const policy = passwordPolicyOverride(minLength, maxLength, historyDepth)
      const next = await updateAdminSettings(csrfToken, {
        password_policy_override: policy,
      })
      onSaved(next)
      setMinLength(next.password_policy_override?.min_length?.toString() ?? '')
      setMaxLength(next.password_policy_override?.max_length?.toString() ?? '')
      setHistoryDepth(next.password_policy_override?.history_depth?.toString() ?? '')
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
              {t.edit}
            </Button>
          ) : null}
        </div>
        <dl className="mt-3 grid grid-cols-3 gap-3 rounded-md border border-slate-200 bg-slate-50 px-4 py-3 text-xs">
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
        </dl>
        <p className="mt-2 text-xs text-slate-500">{t.weakerPolicyWarning}</p>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-3">
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
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <PolicyField
                id="min-length"
                label={t.minLengthFieldLabel}
                value={minLength}
                onChange={setMinLength}
                min={defaults.min_length}
                max={defaults.max_length}
                placeholder={defaults.min_length.toString()}
                hint={t.atLeastHint.replace('{n}', defaults.min_length.toString())}
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
              />
            </div>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving}>
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
}: {
  id: string
  label: string
  value: string
  onChange: (next: string) => void
  min: number
  max: number
  placeholder: string
  hint: string
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
      />
      <p className="text-xs text-slate-500">{hint}</p>
    </div>
  )
}

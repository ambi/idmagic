import { IconPencil } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateAdminSettings } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import type { AdminSettings } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { displayNameError, ReadSetting } from './AdminSettingsShared'

export function GeneralTab({
  csrfToken,
  settings,
  onSaved,
}: {
  csrfToken: string
  settings: AdminSettings
  onSaved: (next: AdminSettings) => void
}) {
  const [displayName, setDisplayName] = useState(settings.display_name)
  // 空文字列は「システムの既定に従う」。API 側も空文字列で未設定へ戻す。
  const [defaultLocale, setDefaultLocale] = useState(settings.default_locale ?? '')
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminSettingsDictionary)

  const localeOptions = [
    { value: '', label: t.defaultLocaleSystemOption },
    ...(settings.supported_locales ?? []).map((locale) => ({
      value: locale,
      label:
        locale === 'ja'
          ? t.defaultLocaleJaOption
          : locale === 'en'
            ? t.defaultLocaleEnOption
            : locale,
    })),
  ]

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const trimmed = displayName.trim()
      const validationError = displayNameError(displayName, t)
      if (validationError) {
        setError(validationError)
        return
      }
      const displayNameChanged = trimmed !== settings.display_name
      const localeChanged = defaultLocale !== (settings.default_locale ?? '')
      if (!displayNameChanged && !localeChanged) {
        setNotice(t.noChangesNotice)
        return
      }
      const next = await updateAdminSettings(csrfToken, {
        ...(displayNameChanged ? { display_name: trimmed } : {}),
        ...(localeChanged ? { default_locale: defaultLocale } : {}),
      })
      onSaved(next)
      setDisplayName(next.display_name)
      setDefaultLocale(next.default_locale ?? '')
      setEditing(false)
      setNotice(displayNameChanged ? t.displayNameUpdatedNotice : t.settingsUpdatedNotice)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : t.settingsUpdateFailedError,
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
            <h2 className="text-base font-semibold text-slate-900">{t.generalHeading}</h2>
            <p className="mt-1 text-sm text-slate-600">{t.generalSubheading}</p>
          </div>
          {!editing ? (
            <Button type="button" variant="outline" onClick={() => setEditing(true)}>
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </Button>
          ) : null}
        </div>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-2">
            <ReadSetting label={t.tenantIdLabel} value={settings.tenant_id} mono />
            <ReadSetting label={t.displayNameLabel} value={settings.display_name} />
            <ReadSetting
              label={t.defaultLocaleLabel}
              value={
                localeOptions.find((option) => option.value === (settings.default_locale ?? ''))
                  ?.label ?? t.defaultLocaleSystemOption
              }
            />
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="tenant-id">{t.tenantIdLabel}</Label>
              <Input
                id="tenant-id"
                value={settings.tenant_id}
                readOnly
                aria-readonly="true"
                className="bg-slate-50 font-mono"
                tabIndex={-1}
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="display-name">{t.displayNameLabel}</Label>
              <Input
                id="display-name"
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                maxLength={LENGTH.displayName}
              />
              <p className="text-xs text-slate-500">{t.displayNameHelp}</p>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="default-locale">{t.defaultLocaleLabel}</Label>
              <select
                id="default-locale"
                value={defaultLocale}
                onChange={(event) => setDefaultLocale(event.target.value)}
                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
              >
                {localeOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
              <p className="text-xs text-slate-500">{t.defaultLocaleHelp}</p>
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
                  setDisplayName(settings.display_name)
                  setDefaultLocale(settings.default_locale ?? '')
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

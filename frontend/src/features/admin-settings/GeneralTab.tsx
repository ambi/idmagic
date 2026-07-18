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
      const trimmed = displayName.trim()
      const validationError = displayNameError(displayName, t)
      if (validationError) {
        setError(validationError)
        return
      }
      if (trimmed === settings.display_name) {
        setNotice(t.noChangesNotice)
        return
      }
      const next = await updateAdminSettings(csrfToken, { display_name: trimmed })
      onSaved(next)
      setDisplayName(next.display_name)
      setEditing(false)
      setNotice(t.displayNameUpdatedNotice)
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
                maxLength={200}
              />
              <p className="text-xs text-slate-500">{t.displayNameHelp}</p>
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

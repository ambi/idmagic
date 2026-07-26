import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, tenantURL, updateSamlIDPProfile } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { useDictionary } from '../../lib/i18n'
import type { AdminSamlIDPProfile } from '../../types'
import type { SamlIDPProfileMode } from '../../types'
import { adminSamlIDPProfilesDictionary } from './AdminSamlIDPProfilesPage.i18n'

export function AdminSamlIDPProfileEditPage({
  csrfToken,
  actorUsername,
  entry,
}: {
  csrfToken: string
  actorUsername?: string
  entry: AdminSamlIDPProfile
}) {
  const [name, setName] = useState(entry.profile.name)
  const [mode, setMode] = useState<SamlIDPProfileMode>(entry.profile.mode)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminSamlIDPProfilesDictionary)
  const detailPath = tenantURL(
    `/admin/settings/saml-idp-profiles/${encodeURIComponent(entry.profile.profile_id)}`,
  )
  const changed = name.trim() !== entry.profile.name || mode !== entry.profile.mode

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!name.trim() || !changed) return
    setBusy(true)
    setError('')
    try {
      await updateSamlIDPProfile(csrfToken, entry.profile.profile_id, {
        name: name.trim(),
        mode,
      })
      window.location.assign(detailPath)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.updateFailed)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="settings"
      actorUsername={actorUsername}
      title={t.editTitle}
      description={t.editDescription}
    >
      <Card className="max-w-2xl">
        <form onSubmit={submit}>
          <div className="grid gap-5 p-6">
            {error ? <Alert variant="destructive">{error}</Alert> : null}
            <div className="grid gap-1.5">
              <Label htmlFor="saml-profile-name">{t.profileName}</Label>
              <Input
                id="saml-profile-name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t.sharingMode}</Label>
              <Select
                value={mode}
                onValueChange={(value) => setMode(value as SamlIDPProfileMode)}
                options={[
                  { value: 'shared', label: t.shared },
                  { value: 'dedicated', label: t.dedicated },
                ]}
              />
            </div>
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button asChild variant="outline">
              <a href={detailPath}>{t.cancel}</a>
            </Button>
            <Button type="submit" disabled={busy || !name.trim() || !changed}>
              {busy ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

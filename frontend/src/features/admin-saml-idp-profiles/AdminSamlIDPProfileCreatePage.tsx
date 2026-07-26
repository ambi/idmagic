import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, createSamlIDPProfile, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { useDictionary } from '../../lib/i18n'
import type { SamlIDPProfileMode } from '../../types'
import { adminSamlIDPProfilesDictionary } from './AdminSamlIDPProfilesPage.i18n'

export function AdminSamlIDPProfileCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const [name, setName] = useState('')
  const [mode, setMode] = useState<SamlIDPProfileMode>('shared')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminSamlIDPProfilesDictionary)
  const listPath = tenantURL('/admin/settings/saml-idp-profiles')

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    setBusy(true)
    setError('')
    try {
      const created = await createSamlIDPProfile(csrfToken, { name: trimmed, mode })
      window.location.assign(`${listPath}/${encodeURIComponent(created.profile.profile_id)}`)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.createFailed)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="settings"
      actorUsername={actorUsername}
      title={t.newTitle}
      description={t.newDescription}
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
              <a href={listPath}>{t.cancel}</a>
            </Button>
            <Button type="submit" disabled={busy || !name.trim()}>
              {busy ? t.saving : t.create}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

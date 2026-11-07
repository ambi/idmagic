import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, tenantURL, updateSamlIDPProfile } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { useFormValidation } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import type { AdminSamlIDPProfile } from '../../types'
import type { SamlIDPProfileMode } from '../../types'
import { adminSamlIDPProfilesDictionary } from './AdminSamlIDPProfilesPage.i18n'
import { profileFormSchema } from './SamlIDPProfileFields'

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
  const validation = useFormValidation(profileFormSchema, { name })
  const changed = name.trim() !== entry.profile.name || mode !== entry.profile.mode

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    if (!changed) return
    setBusy(true)
    setError('')
    try {
      await updateSamlIDPProfile(csrfToken, entry.profile.profile_id, {
        name: data.name,
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
      <Card className="admin-form">
        <form onSubmit={submit}>
          <div className="grid gap-5 p-6">
            {error ? <Alert variant="destructive">{error}</Alert> : null}
            <div className="grid gap-1.5">
              <Label htmlFor="saml-profile-name" required>
                {t.profileName}
              </Label>
              <Input
                id="saml-profile-name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
                {...validation.fieldProps('name')}
              />
              <FieldError form={validation} name="name" />
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
            <Button variant="outline" nativeButton={false} render={<a href={detailPath} />}>
              {t.cancel}
            </Button>
            <Button type="submit" disabled={busy || !validation.data || !changed}>
              {busy ? t.saving : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

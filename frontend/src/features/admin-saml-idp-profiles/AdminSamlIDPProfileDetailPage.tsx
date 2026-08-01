import { IconArrowLeft, IconPencil, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, deleteSamlIDPProfile, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { AdminSamlIDPProfile } from '../../types'
import { adminSamlIDPProfilesDictionary } from './AdminSamlIDPProfilesPage.i18n'
import { SamlIDPProfileFields } from './SamlIDPProfileFields'

export function AdminSamlIDPProfileDetailPage({
  csrfToken,
  actorUsername,
  entry,
}: {
  csrfToken: string
  actorUsername?: string
  entry: AdminSamlIDPProfile
}) {
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminSamlIDPProfilesDictionary)
  const profile = entry.profile
  const listPath = tenantURL('/admin/settings/saml-idp-profiles')
  const detailPath = `${listPath}/${encodeURIComponent(profile.profile_id)}`
  const deleteDisabled = entry.service_provider_count > 0

  async function remove() {
    setBusy(true)
    setError('')
    try {
      await deleteSamlIDPProfile(csrfToken, profile.profile_id)
      window.location.assign(listPath)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.deleteFailed)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="settings"
      actorUsername={actorUsername}
      title={profile.name}
      description={t.detailDescription}
      actions={
        <>
          <Button variant="outline" nativeButton={false} render={<a href={listPath} />}>
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.listTitle}
          </Button>
          {!profile.is_default ? (
            <>
              <Button nativeButton={false} render={<a href={`${detailPath}/edit`} />}>
                <IconPencil size={16} aria-hidden="true" />
                {t.editProfile}
              </Button>
              <Button
                type="button"
                variant="destructive"
                disabled={busy || deleteDisabled}
                onClick={() => setConfirmDelete(true)}
              >
                <IconTrash size={16} aria-hidden="true" />
                {t.deleteProfile}
              </Button>
            </>
          ) : null}
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {profile.is_default ? <Alert>{t.defaultImmutable}</Alert> : null}
      {deleteDisabled && !profile.is_default ? <Alert>{t.profileInUse}</Alert> : null}
      {confirmDelete ? (
        <Alert variant="destructive" className="flex items-center justify-between gap-3">
          <span>{t.confirmDelete}</span>
          <div className="flex gap-2">
            <Button variant="outline" disabled={busy} onClick={() => setConfirmDelete(false)}>
              {t.dismiss}
            </Button>
            <Button variant="destructive" disabled={busy} onClick={() => void remove()}>
              {t.delete}
            </Button>
          </div>
        </Alert>
      ) : null}
      <Card className="max-w-3xl p-6">
        <div className="mb-5 grid gap-1">
          <p className="text-sm font-medium text-slate-900">
            {profile.mode === 'shared' ? t.shared : t.dedicated}
          </p>
          <p className="text-xs text-slate-500">
            {profile.is_default ? `${t.defaultProfile}. ` : ''}
            {t.assignedServiceProviders.replace('{count}', String(entry.service_provider_count))}
          </p>
        </div>
        <SamlIDPProfileFields entry={entry} t={t} />
      </Card>
    </AdminShell>
  )
}

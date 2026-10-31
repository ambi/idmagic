import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, createDirectoryConnector } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { DirectoryConnectorEnrollmentGrant } from '../../types'
import { adminDirectoryConnectorsDictionary } from './AdminDirectoryConnectorsPage.i18n'
import { EnrollmentTokenNotice } from './EnrollmentTokenNotice'
import { detailURL, listURL } from './presentation'

export function AdminDirectoryConnectorCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  // 応答を受けても遷移しない。遷移で渡すとトークンが URL か遷移の状態に残る。
  const [grant, setGrant] = useState<DirectoryConnectorEnrollmentGrant>()
  const t = useDictionary(adminDirectoryConnectorsDictionary)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      setGrant(await createDirectoryConnector(csrfToken, displayName))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
    } finally {
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="directory-connectors"
      actorUsername={actorUsername}
      title={t.createConnector}
      description={t.pageDescription}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full max-w-2xl p-4">
        {grant ? (
          <div className="flex flex-col gap-4">
            <EnrollmentTokenNotice token={grant.enrollment_token} t={t} />
            <div>
              <Button nativeButton={false} render={<a href={detailURL(grant.connector.id)} />}>
                {t.toDetail}
              </Button>
            </div>
          </div>
        ) : (
          <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="display_name">{t.displayNameLabel}</Label>
              <Input
                id="display_name"
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                required
              />
              <p className="text-xs leading-5 text-slate-500">{t.displayNameHelp}</p>
            </div>
            <div className="flex gap-2.5">
              <Button type="submit" disabled={saving}>
                {t.create}
              </Button>
              <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
                {t.cancel}
              </Button>
            </div>
          </form>
        )}
      </Card>
    </AdminShell>
  )
}

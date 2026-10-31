import { IconArrowLeft } from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  disableDirectoryConnector,
  reissueDirectoryConnectorEnrollment,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useFormatters } from '../../lib/i18n'
import type { DirectoryConnector, IngestionRunSummary } from '../../types'
import { adminDirectoryConnectorsDictionary } from './AdminDirectoryConnectorsPage.i18n'
import { EnrollmentTokenNotice } from './EnrollmentTokenNotice'
import { IngestionRunsCard } from './IngestionRunsCard'
import { canManage, connectorStatusLabel, enrollmentExpired, listURL } from './presentation'

export function AdminDirectoryConnectorDetailPage({
  csrfToken,
  actorUsername,
  connector: initialConnector,
  runs,
}: {
  csrfToken: string
  actorUsername?: string
  connector: DirectoryConnector
  runs: IngestionRunSummary[]
}) {
  const [connector, setConnector] = useState(initialConnector)
  const [token, setToken] = useState('')
  const [confirming, setConfirming] = useState<'reissue' | 'disable'>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminDirectoryConnectorsDictionary)
  const { formatDateTime } = useFormatters()

  async function reissue() {
    setBusy(true)
    setError('')
    setToken('')
    try {
      const grant = await reissueDirectoryConnectorEnrollment(csrfToken, connector.id)
      setConnector(grant.connector)
      setToken(grant.enrollment_token)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.reissueFailedError)
    } finally {
      setBusy(false)
      setConfirming(undefined)
    }
  }

  async function disable() {
    setBusy(true)
    setError('')
    setToken('')
    try {
      setConnector(await disableDirectoryConnector(csrfToken, connector.id))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.disableFailedError)
    } finally {
      setBusy(false)
      setConfirming(undefined)
    }
  }

  const rows: [string, string | undefined][] = [
    [t.connectorIdLabel, connector.id],
    [t.createdAtLabel, formatDateTime(connector.created_at)],
    [t.lastSeenLabel, connector.last_seen_at ? formatDateTime(connector.last_seen_at) : '—'],
    [t.enrolledAtLabel, connector.enrolled_at && formatDateTime(connector.enrolled_at)],
    [t.keyIdLabel, connector.key_id],
    [
      t.enrollmentExpiresAtLabel,
      connector.enrollment_expires_at && formatDateTime(connector.enrollment_expires_at),
    ],
    [t.disabledAtLabel, connector.disabled_at && formatDateTime(connector.disabled_at)],
  ]

  return (
    <AdminShell
      active="directory-connectors"
      actorUsername={actorUsername}
      title={connector.display_name}
      description={connectorStatusLabel(connector.status, t)}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {token ? <EnrollmentTokenNotice token={token} t={t} /> : null}

      <Card className="p-4">
        <h2 className="text-base font-semibold text-slate-950">{t.summaryHeading}</h2>
        <dl className="mt-3 grid gap-x-6 gap-y-2 text-sm sm:grid-cols-[max-content_1fr]">
          {rows.map(([label, value]) =>
            value ? (
              <div key={label} className="contents">
                <dt className="text-slate-500">{label}</dt>
                <dd className="break-all text-slate-900">{value}</dd>
              </div>
            ) : null,
          )}
        </dl>
        {enrollmentExpired(connector, new Date()) ? (
          <p className="mt-3 text-sm text-amber-700">{t.enrollmentExpiredNotice}</p>
        ) : null}
        {canManage(connector) ? (
          <div className="mt-4 flex flex-wrap gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={() =>
                // 登録待ちには破棄する鍵がなく、同期も止まらないので確認しない。
                connector.status === 'active' ? setConfirming('reissue') : void reissue()
              }
            >
              {t.reissue}
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={() => setConfirming('disable')}
            >
              {t.disable}
            </Button>
          </div>
        ) : null}
        {confirming ? (
          <Alert variant="destructive" className="mt-4 grid gap-3">
            <p>{confirming === 'reissue' ? t.reissueConfirmMessage : t.disableConfirmMessage}</p>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                disabled={busy}
                onClick={() => setConfirming(undefined)}
              >
                {t.cancel}
              </Button>
              <Button
                type="button"
                variant="destructive"
                disabled={busy}
                onClick={() => void (confirming === 'reissue' ? reissue() : disable())}
              >
                {confirming === 'reissue' ? t.reissueConfirm : t.disableConfirm}
              </Button>
            </div>
          </Alert>
        ) : null}
      </Card>

      <IngestionRunsCard connectorID={connector.id} runs={runs} t={t} />
    </AdminShell>
  )
}

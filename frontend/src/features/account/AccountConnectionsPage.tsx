import { useState } from 'react'
import {
  authorizeMyTokenVaultConnection,
  listMyTokenVaultConnections,
  unlinkMyTokenVaultConnection,
} from '../../api/account'
import { AccountShell } from '../../components/AccountShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useFormatters } from '../../lib/i18n'
import type { TokenVaultMyConnection } from '../../types'
import { accountConnectionsDictionary } from './AccountConnectionsPage.i18n'

type Dictionary = (typeof accountConnectionsDictionary)['en']

function resultMessage(
  result: string | undefined,
  t: Dictionary,
): { notice?: string; error?: string } {
  switch (result) {
    case 'linked':
      return { notice: t.resultLinked }
    case 'invalid_state':
      return { error: t.resultInvalidState }
    case 'denied':
      return { error: t.resultDenied }
    case 'exchange_failed':
      return { error: t.resultExchangeFailed }
    default:
      return {}
  }
}

function statusLabel(connection: TokenVaultMyConnection, t: Dictionary): string {
  if (connection.status === 'active') return t.statusActive
  if (connection.status === 'reauthorization_required') return t.statusReauthorization
  return t.statusNotLinked
}

export function AccountConnectionsPage({
  csrfToken,
  username,
  isAdmin,
  connections: initial,
  result,
  navigate = (url: string) => window.location.assign(url),
}: {
  csrfToken: string
  username: string
  isAdmin: boolean
  connections: TokenVaultMyConnection[]
  result?: string
  navigate?: (url: string) => void
}) {
  const t = useDictionary(accountConnectionsDictionary)
  const { formatDateTime } = useFormatters()
  const message = resultMessage(result, t)
  const [connections, setConnections] = useState(initial)
  const [notice, setNotice] = useState(message.notice ?? '')
  const [error, setError] = useState(message.error ?? '')
  const [busy, setBusy] = useState(false)

  async function run(action: () => Promise<void>) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
    } catch {
      setError(t.genericError)
    } finally {
      setBusy(false)
    }
  }

  const link = (connectorId: string) =>
    run(async () => navigate(await authorizeMyTokenVaultConnection(csrfToken, connectorId)))

  const unlink = (connectorId: string) =>
    run(async () => {
      await unlinkMyTokenVaultConnection(csrfToken, connectorId)
      setConnections(await listMyTokenVaultConnections())
      setNotice(t.unlinkedNotice)
    })

  return (
    <AccountShell
      active="connections"
      username={username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      <Toast message={notice} onDismiss={() => setNotice('')} />
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {connections.length === 0 ? (
        <Card className="p-6 text-sm text-slate-500">{t.empty}</Card>
      ) : null}
      <div className="flex flex-col gap-3">
        {connections.map((connection) => (
          <Card
            key={connection.connector_id}
            className="flex flex-col gap-3 p-4 md:flex-row md:items-center md:justify-between"
          >
            <div className="min-w-0">
              <h2 className="font-semibold">{connection.connector_name}</h2>
              <p className="text-sm text-slate-600">
                {connection.connector_status === 'disabled'
                  ? t.connectorDisabled
                  : statusLabel(connection, t)}
              </p>
              {connection.granted_scopes.length > 0 ? (
                <p className="break-words text-xs text-slate-500">
                  {t.scopes}: {connection.granted_scopes.join(' ')}
                </p>
              ) : null}
              {connection.status ? (
                <p className="text-xs text-slate-500">
                  {t.lastUsed}:{' '}
                  {connection.last_used_at ? formatDateTime(connection.last_used_at) : t.neverUsed}
                </p>
              ) : null}
            </div>
            <div className="flex shrink-0 gap-2">
              {connection.connector_status === 'active' && connection.status !== 'active' ? (
                <Button disabled={busy} onClick={() => link(connection.connector_id)}>
                  {connection.status ? t.relink : t.link}
                </Button>
              ) : null}
              {connection.status ? (
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => unlink(connection.connector_id)}
                >
                  {t.unlink}
                </Button>
              ) : null}
            </div>
          </Card>
        ))}
      </div>
    </AccountShell>
  )
}

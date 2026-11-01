import { useEffect, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { AuthenticationAPIError } from '../../api'
import {
  deleteTokenVaultConnector,
  listTokenVaultConnectors,
  setTokenVaultConnectorStatus,
} from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { TokenVaultConnector } from '../../types'
import { adminTokenVaultDictionary } from './AdminTokenVaultPage.i18n'

export function AdminTokenVaultPage({
  csrfToken,
  actorUsername,
  connectors: initial,
  justCreated = false,
}: {
  csrfToken: string
  actorUsername?: string
  connectors: TokenVaultConnector[]
  justCreated?: boolean
}) {
  const t = useDictionary(adminTokenVaultDictionary)
  const [connectors, setConnectors] = useState(initial)
  useEffect(() => setConnectors(initial), [initial])
  const [pending, setPending] = useState<{
    connector: TokenVaultConnector
    action: 'delete' | 'disable'
  } | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState(justCreated ? t.createdNotice : '')
  async function refresh() {
    setConnectors(await listTokenVaultConnectors())
  }
  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      await refresh()
      setNotice(success)
      setPending(null)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericError)
    } finally {
      setBusy(false)
    }
  }
  function confirm() {
    if (!pending) return
    void run(
      () =>
        pending.action === 'delete'
          ? deleteTokenVaultConnector(csrfToken, pending.connector.id)
          : setTokenVaultConnectorStatus(csrfToken, pending.connector.id, 'disabled').then(
              () => {},
            ),
      pending.action === 'delete' ? t.deletedNotice : t.updatedNotice,
    )
  }
  return (
    <AdminShell
      active="token-vault"
      actorUsername={actorUsername}
      title={t.title}
      description={t.description}
      list={{ total: connectors.length, displayed: connectors.length, busy, onRefresh: refresh }}
      actions={
        <Button nativeButton={false} render={<Link to="/admin/token-vault/new" />}>
          {t.add}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />
      {connectors.length === 0 ? (
        <Card className="p-6 text-sm text-slate-500">{t.empty}</Card>
      ) : null}
      {connectors.map((connector) => (
        <Card
          key={connector.id}
          className="flex flex-col gap-3 p-4 md:flex-row md:items-center md:justify-between"
        >
          <div className="min-w-0">
            <h2 className="font-semibold">{connector.name}</h2>
            <p className="text-sm text-slate-600">
              {connector.status === 'active' ? t.statusActive : t.statusDisabled} ·{' '}
              {t.rulesCount.replace('{count}', String(connector.rules.length))}
            </p>
            <p className="break-all text-xs text-slate-500">{connector.token_endpoint}</p>
          </div>
          <div className="flex shrink-0 gap-2">
            <Button
              variant={connector.status === 'active' ? 'destructive' : 'outline'}
              disabled={busy}
              onClick={() => {
                if (connector.status === 'active') setPending({ connector, action: 'disable' })
                else
                  void run(
                    () =>
                      setTokenVaultConnectorStatus(csrfToken, connector.id, 'active').then(
                        () => {},
                      ),
                    t.updatedNotice,
                  )
              }}
            >
              {connector.status === 'active' ? t.disable : t.enable}
            </Button>
            <Button
              variant="destructive"
              disabled={busy}
              onClick={() => setPending({ connector, action: 'delete' })}
            >
              {t.delete}
            </Button>
          </div>
        </Card>
      ))}
      {pending ? (
        <ConfirmActionDialog
          title={pending.action === 'delete' ? t.delete : t.disable}
          target={pending.connector.name}
          description={
            pending.action === 'delete'
              ? t.deleteConfirm.replace('{name}', pending.connector.name)
              : t.disableConfirm
          }
          confirmLabel={pending.action === 'delete' ? t.delete : t.disable}
          busy={busy}
          onClose={() => setPending(null)}
          onConfirm={confirm}
        />
      ) : null}
    </AdminShell>
  )
}

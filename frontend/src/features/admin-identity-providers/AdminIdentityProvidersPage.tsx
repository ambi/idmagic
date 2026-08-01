import { IconNetwork, IconPlus } from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  deleteIdentityProviderConnection,
  runIdentityProviderAction,
  tenantURL,
  testIdentityProviderConnection,
  type IdentityProviderConnection,
  type IdentityProviderConnectionTestResult,
} from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import { identityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'
import {
  DeleteConnectionDialog,
  StatusBadge,
  TestResultBanner,
} from './AdminIdentityProvidersShared'

export function AdminIdentityProvidersPage({
  csrfToken,
  actorUsername,
  connections: initialConnections,
}: {
  csrfToken: string
  actorUsername?: string
  connections: IdentityProviderConnection[]
}) {
  const [connections, setConnections] = useState(initialConnections)
  const [busyId, setBusyId] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [testResult, setTestResult] = useState<{
    id: string
    result: IdentityProviderConnectionTestResult
  } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<IdentityProviderConnection | null>(null)
  const t = useDictionary(identityProvidersDictionary)

  async function run(connection: IdentityProviderConnection, action: () => Promise<void>) {
    setBusyId(connection.id)
    setError('')
    try {
      await action()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
    } finally {
      setBusyId('')
    }
  }

  async function toggleStatus(connection: IdentityProviderConnection) {
    const activate = connection.status === 'disabled'
    await run(connection, async () => {
      await runIdentityProviderAction(csrfToken, connection.id, activate ? 'activate' : 'disable')
      setConnections((current) =>
        current.map((item) =>
          item.id === connection.id ? { ...item, status: activate ? 'active' : 'disabled' } : item,
        ),
      )
      setNotice(activate ? t.activated : t.disabled)
    })
  }

  async function test(connection: IdentityProviderConnection) {
    await run(connection, async () => {
      const result = await testIdentityProviderConnection(csrfToken, connection.id)
      setTestResult({ id: connection.id, result })
    })
  }

  async function remove(connection: IdentityProviderConnection) {
    await run(connection, async () => {
      await deleteIdentityProviderConnection(csrfToken, connection.id)
      setConnections((current) => current.filter((item) => item.id !== connection.id))
      setConfirmDelete(null)
      setNotice(t.deleted)
    })
  }

  function requestDelete(connection: IdentityProviderConnection) {
    if (connection.status === 'active') {
      setConfirmDelete(connection)
    } else {
      void remove(connection)
    }
  }

  return (
    <>
      <AdminShell
        active="identity-providers"
        actorUsername={actorUsername}
        title={t.pageTitle}
        description={t.pageDescription}
        actions={
          <Button
            nativeButton={false}
            render={<a href={tenantURL('/admin/identity-providers/new')} />}
          >
            <IconPlus size={17} aria-hidden="true" />
            {t.addConnection}
          </Button>
        }
      >
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {testResult ? (
          <TestResultBanner result={testResult.result} onDismiss={() => setTestResult(null)} />
        ) : null}

        {connections.length === 0 ? (
          <Card className="flex flex-col items-center gap-2 p-10 text-center">
            <IconNetwork size={28} className="text-slate-400" aria-hidden="true" />
            <p className="font-semibold text-slate-800">{t.empty}</p>
            <p className="text-sm text-slate-500">{t.emptyHint}</p>
          </Card>
        ) : (
          <Card className="overflow-hidden">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="px-5 py-3.5">{t.tableHeaderName}</th>
                  <th className="px-5 py-3.5">{t.tableHeaderProtocol}</th>
                  <th className="px-5 py-3.5">{t.tableHeaderStatus}</th>
                  <th className="px-5 py-3.5" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {connections.map((connection) => (
                  <tr key={connection.id} className={cn('bg-white')}>
                    <td className="px-5 py-4">
                      <p className="font-semibold text-slate-900">{connection.display_name}</p>
                      <p className="mt-0.5 truncate text-xs text-slate-500">{connection.issuer}</p>
                    </td>
                    <td className="px-5 py-4 text-slate-600">
                      {connection.protocol.toUpperCase()}
                    </td>
                    <td className="px-5 py-4">
                      <StatusBadge status={connection.status} labels={t} />
                    </td>
                    <td className="px-5 py-4">
                      <AdminPaneActions
                        detailHref={tenantURL(
                          `/admin/identity-providers/${encodeURIComponent(connection.id)}`,
                        )}
                        editHref={tenantURL(
                          `/admin/identity-providers/${encodeURIComponent(connection.id)}/edit`,
                        )}
                        busy={busyId === connection.id}
                        actions={[
                          { label: t.test, onClick: () => void test(connection) },
                          {
                            label: connection.status === 'active' ? t.disable : t.activate,
                            onClick: () => void toggleStatus(connection),
                          },
                          {
                            label: t.delete,
                            onClick: () => requestDelete(connection),
                            tone: 'danger',
                          },
                        ]}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        )}
      </AdminShell>

      {confirmDelete ? (
        <DeleteConnectionDialog
          connection={confirmDelete}
          busy={busyId === confirmDelete.id}
          onClose={() => setConfirmDelete(null)}
          onConfirm={() => void remove(confirmDelete)}
        />
      ) : null}
    </>
  )
}

import { IconPlus, IconServer2, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  deleteMcpResourceServer,
  listMcpResourceServers,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import { editURL, newURL, McpResourceServerStatus } from './AdminMcpResourceServersShared'
import { shellDictionary } from '../../components/shell.i18n'

// AdminMcpResourceServersPage は MCP リソースサーバーの参照専用一覧。作成/編集は専用ルート
// (new / $resourceServerId/edit) に一本化されている (policy #2, wi-314 T014)。
export function AdminMcpResourceServersPage({
  csrfToken,
  actorUsername,
  resourceServers,
}: {
  csrfToken: string
  actorUsername?: string
  resourceServers: McpResourceServer[]
}) {
  const [items, setItems] = useState(resourceServers)
  const [selectedID, setSelectedID] = useState(resourceServers[0]?.id ?? '')
  const selected = items.find((item) => item.id === selectedID) ?? items[0] ?? null
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const dict = useDictionary(adminMcpResourceServersDictionary)
  const common = useDictionary(commonDictionary)
  const access = useDictionary(shellDictionary)
  const [target, setTarget] = useState<McpResourceServer | null>(null)
  const [busy, setBusy] = useState(false)

  async function handleDelete(resourceServer: McpResourceServer) {
    setError('')
    setBusy(true)
    try {
      await deleteMcpResourceServer(csrfToken, resourceServer.id)
      setItems((previous) => previous.filter((item) => item.id !== resourceServer.id))
      setNotice(dict.deletedNotice.replace('{resource}', resourceServer.name))
      setTarget(null)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : dict.deleteFailedError)
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="mcp-resource-servers"
      actorUsername={actorUsername}
      title={dict.pageTitle}
      description={dict.pageDescription}
      list={{
        total: items.length,
        displayed: items.length,
        onRefresh: async () => setItems(await listMcpResourceServers()),
      }}
      actions={
        <Button nativeButton={false} render={<a href={newURL()} />}>
          <IconPlus size={17} aria-hidden="true" />
          {dict.registerResourceServer}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
        <Card className="min-w-0 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50 text-xs font-semibold text-slate-500">
              <tr>
                <th className="px-4 py-3">{dict.nameLabel}</th>
                <th className="px-4 py-3">{dict.resourceLabel}</th>
                <th className="px-4 py-3">{dict.stateLabel}</th>
                <th className="px-4 py-3 text-right">{dict.toolCount}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr
                  key={item.id}
                  onClick={() => setSelectedID(item.id)}
                  className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${selected?.id === item.id ? 'bg-blue-50/60' : ''}`}
                >
                  <td className="px-4 py-3">
                    <button
                      type="button"
                      className="text-left font-semibold text-slate-900"
                      aria-label={dict.selectServer.replace('{name}', item.name)}
                      aria-pressed={selected?.id === item.id}
                      onClick={() => setSelectedID(item.id)}
                    >
                      {item.name}
                    </button>
                  </td>
                  <td className="break-all px-4 py-3 font-mono text-xs text-slate-500">
                    {item.resource}
                  </td>
                  <td className="px-4 py-3">
                    <McpResourceServerStatus state={item.state} />
                  </td>
                  <td className="px-4 py-3 text-right">
                    {(item.access_rules ?? []).filter((rule) => rule.tool).length}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {items.length === 0 ? (
            <div className="flex min-h-40 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
              <IconServer2 size={24} className="text-slate-400" aria-hidden="true" />
              <p className="mt-3">{dict.emptyNotice}</p>
              <p className="mt-1">{dict.emptyHelp}</p>
            </div>
          ) : null}
        </Card>
        <Card className="min-w-0 p-6">
          {selected ? (
            <>
              <div className="flex items-center justify-between gap-2">
                <h2 className="break-all text-base font-semibold">{selected.name}</h2>
                <McpResourceServerStatus state={selected.state} />
              </div>

              <dl className="mt-4 grid gap-2 text-sm">
                <dt className="font-medium">{dict.ruleScopes}</dt>
                <dd className="flex flex-wrap gap-1.5 font-mono text-xs">
                  {selected.scopes.map((scope) => (
                    <span key={scope} className="rounded-full bg-slate-50 px-2 py-0.5">
                      {scope}
                    </span>
                  ))}
                </dd>
                <dt className="font-medium">{dict.clientIdLabel}</dt>
                <dd className="break-all font-mono text-xs">{selected.client_id || '—'}</dd>
              </dl>
              <div className="mt-4 flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  nativeButton={false}
                  render={
                    <a
                      href={tenantURL(
                        (selected.access_rules ?? []).some((rule) => rule.tool)
                          ? `/admin/mcp-resource-servers/${encodeURIComponent(selected.id)}/access`
                          : `/admin/authorization/access?server=${encodeURIComponent(selected.id)}`,
                      )}
                    />
                  }
                >
                  {(selected.access_rules ?? []).some((rule) => rule.tool)
                    ? dict.assignments
                    : access.authorization}
                </Button>
                <Button
                  variant="outline"
                  nativeButton={false}
                  render={<a href={editURL(selected.id)} />}
                >
                  {dict.edit}
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  disabled={busy}
                  aria-label={`${dict.delete}: ${selected.resource}`}
                  onClick={() => {
                    setError('')
                    setTarget(selected)
                  }}
                >
                  <IconTrash size={16} aria-hidden="true" />
                  {dict.delete}
                </Button>
              </div>
            </>
          ) : (
            <p className="text-sm text-slate-500">{dict.selectServerNotice}</p>
          )}
        </Card>
      </div>
      {target ? (
        <ConfirmActionDialog
          title={dict.delete}
          target={target.name || target.resource}
          description={common.deleteResourceDescription}
          confirmLabel={dict.delete}
          busy={busy}
          error={error}
          onClose={() => setTarget(null)}
          onConfirm={() => void handleDelete(target)}
        />
      ) : null}
    </AdminShell>
  )
}

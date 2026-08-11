import { IconPlus, IconServer2, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, deleteMcpResourceServer } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import { editURL, newURL } from './AdminMcpResourceServersShared'

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
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const dict = useDictionary(adminMcpResourceServersDictionary)

  async function handleDelete(resourceServer: McpResourceServer) {
    setError('')
    try {
      await deleteMcpResourceServer(csrfToken, resourceServer.id)
      setItems((previous) => previous.filter((item) => item.id !== resourceServer.id))
      setNotice(dict.deletedNotice.replace('{resource}', resourceServer.resource))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : dict.deleteFailedError)
    }
  }

  return (
    <AdminShell
      active="mcp-resource-servers"
      actorUsername={actorUsername}
      title={dict.pageTitle}
      description={dict.pageDescription}
      actions={
        <Button nativeButton={false} render={<a href={newURL()} />}>
          <IconPlus size={17} aria-hidden="true" />
          {dict.registerResourceServer}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      {items.length === 0 ? (
        <Card className="p-8 text-center text-sm text-slate-500">{dict.emptyNotice}</Card>
      ) : (
        <div className="flex flex-col gap-3">
          {items.map((item) => (
            <Card key={item.id} className="flex flex-col gap-3 p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="break-all font-mono text-sm font-semibold text-slate-900">
                      {item.resource}
                    </p>
                    <span
                      className={
                        item.state === 'Active'
                          ? 'rounded-full bg-emerald-50 px-2 py-0.5 text-[0.68rem] font-bold text-emerald-700'
                          : 'rounded-full bg-slate-100 px-2 py-0.5 text-[0.68rem] font-bold text-slate-500'
                      }
                    >
                      {item.state}
                    </span>
                  </div>
                  <p className="mt-0.5 text-xs leading-5 text-slate-500">{item.name}</p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button
                    variant="outline"
                    nativeButton={false}
                    render={<a href={editURL(item.id)} />}
                  >
                    {dict.edit}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    aria-label={`${dict.delete}: ${item.resource}`}
                    onClick={() => void handleDelete(item)}
                  >
                    <IconTrash size={16} aria-hidden="true" />
                  </Button>
                </div>
              </div>
              <div className="flex items-start gap-2 rounded-lg bg-slate-50 p-2.5 text-xs leading-5 text-slate-600">
                <IconServer2
                  size={15}
                  className="mt-0.5 shrink-0 text-blue-600"
                  aria-hidden="true"
                />
                <div className="flex flex-wrap gap-1.5">
                  {item.scopes.map((scope) => (
                    <span
                      key={scope}
                      className="rounded-full bg-white px-2 py-0.5 font-mono text-slate-700"
                    >
                      {scope}
                    </span>
                  ))}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </AdminShell>
  )
}

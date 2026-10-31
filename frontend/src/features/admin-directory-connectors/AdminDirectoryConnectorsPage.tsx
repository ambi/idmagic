import { IconPlus, IconServer } from '@tabler/icons-react'
import { AdminShell } from '../../components/AdminShell'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useFormatters } from '../../lib/i18n'
import type { DirectoryConnector } from '../../types'
import { adminDirectoryConnectorsDictionary } from './AdminDirectoryConnectorsPage.i18n'
import { connectorStatusLabel, detailURL, newURL } from './presentation'

export function AdminDirectoryConnectorsPage({
  actorUsername,
  connectors,
}: {
  actorUsername?: string
  connectors: DirectoryConnector[]
}) {
  const t = useDictionary(adminDirectoryConnectorsDictionary)
  const { formatDateTime } = useFormatters()

  return (
    <AdminShell
      active="directory-connectors"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button nativeButton={false} render={<a href={newURL()} />}>
          <IconPlus size={17} aria-hidden="true" />
          {t.createConnector}
        </Button>
      }
    >
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">{t.headerName}</th>
              <th className="px-4 py-3">{t.headerStatus}</th>
              <th className="px-4 py-3">{t.headerLastSeen}</th>
              <th className="px-4 py-3">{t.headerCreatedAt}</th>
              <th className="px-4 py-3 text-right" />
            </tr>
          </thead>
          <tbody>
            {connectors.map((connector) => (
              <tr key={connector.id} className="border-t border-slate-100 hover:bg-slate-50">
                <td className="px-4 py-3 font-semibold text-slate-900">{connector.display_name}</td>
                <td className="px-4 py-3 text-slate-600">
                  {connectorStatusLabel(connector.status, t)}
                </td>
                <td className="px-4 py-3 text-xs text-slate-600">
                  {connector.last_seen_at ? formatDateTime(connector.last_seen_at) : '—'}
                </td>
                <td className="px-4 py-3 text-xs text-slate-600">
                  {formatDateTime(connector.created_at)}
                </td>
                <td className="px-4 py-3 text-right">
                  <Button
                    variant="outline"
                    nativeButton={false}
                    render={<a href={detailURL(connector.id)} />}
                  >
                    {t.detail}
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {connectors.length === 0 ? (
          <div className="flex min-h-40 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
            <IconServer size={24} className="text-slate-400" aria-hidden="true" />
            <p className="mt-3">{t.emptyNotice}</p>
          </div>
        ) : null}
      </Card>
    </AdminShell>
  )
}

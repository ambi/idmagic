import { createFileRoute } from '@tanstack/react-router'
import { getDirectoryConnector, listDirectoryIngestionRuns } from '../../../api/admin'
import { AdminDirectoryConnectorDetailPage } from '../../../features/admin-directory-connectors/AdminDirectoryConnectorDetailPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/directory-connectors_/$connectorId')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [connector, runs] = await Promise.all([
      getDirectoryConnector(params.connectorId),
      listDirectoryIngestionRuns(params.connectorId),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      connector,
      runs,
    }
  },
  component: AdminDirectoryConnectorDetailRoute,
})

function AdminDirectoryConnectorDetailRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-directory-connector-detail">
      {/* 再発行と無効化は部品の状態を更新するので、Connector が変わったら作り直す。 */}
      <AdminDirectoryConnectorDetailPage key={data.connector.id} {...data} />
    </PageMarker>
  )
}

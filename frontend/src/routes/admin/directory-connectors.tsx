import { createFileRoute } from '@tanstack/react-router'
import { listDirectoryConnectors } from '../../api/admin'
import { AdminDirectoryConnectorsPage } from '../../features/admin-directory-connectors/AdminDirectoryConnectorsPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/directory-connectors')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      actorUsername: account.preferred_username,
      connectors: await listDirectoryConnectors(),
    }
  },
  component: AdminDirectoryConnectorsRoute,
})

function AdminDirectoryConnectorsRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-directory-connectors">
      <AdminDirectoryConnectorsPage {...data} />
    </PageMarker>
  )
}

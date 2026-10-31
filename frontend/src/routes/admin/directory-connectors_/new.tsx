import { createFileRoute } from '@tanstack/react-router'
import { AdminDirectoryConnectorCreatePage } from '../../../features/admin-directory-connectors/AdminDirectoryConnectorCreatePage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/directory-connectors_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminDirectoryConnectorCreateRoute,
})

function AdminDirectoryConnectorCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-directory-connector-create">
      <AdminDirectoryConnectorCreatePage {...data} />
    </PageMarker>
  )
}

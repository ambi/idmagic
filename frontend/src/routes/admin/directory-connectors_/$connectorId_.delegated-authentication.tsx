import { createFileRoute } from '@tanstack/react-router'
import { getDirectoryConnector } from '../../../api/admin'
import { DelegatedAuthenticationPage } from '../../../features/admin-directory-connectors/DelegatedAuthenticationPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute(
  '/admin/directory-connectors_/$connectorId_/delegated-authentication',
)({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      connector: await getDirectoryConnector(params.connectorId),
    }
  },
  component: DelegatedAuthenticationRoute,
})

function DelegatedAuthenticationRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-directory-delegated-authentication">
      <DelegatedAuthenticationPage key={data.connector.id} {...data} />
    </PageMarker>
  )
}

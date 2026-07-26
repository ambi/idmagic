import { createFileRoute } from '@tanstack/react-router'
import { getAdminIntegrationEndpoints, listWsFedRelyingParties } from '../../../api/admin'
import { AdminEntraFederationPage } from '../../../features/admin-entra-federation/AdminEntraFederationPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/federation/entra')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [relyingParties, integrationEndpoints] = await Promise.all([
      listWsFedRelyingParties(),
      getAdminIntegrationEndpoints(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      relyingParties,
      integrationEndpoints,
    }
  },
  component: AdminEntraFederationRoute,
})

function AdminEntraFederationRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-entra-federation">
      <AdminEntraFederationPage {...data} />
    </PageMarker>
  )
}

import { createFileRoute } from '@tanstack/react-router'
import { AdminEntraFederationAddPage } from '../../../../features/admin-entra-federation/AdminEntraFederationPage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/federation/entra_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminEntraFederationAddRoute,
})

function AdminEntraFederationAddRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-entra-federation-add">
      <AdminEntraFederationAddPage {...data} />
    </PageMarker>
  )
}

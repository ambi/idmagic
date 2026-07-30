import { createFileRoute } from '@tanstack/react-router'
import { listIdentityProviderConnections } from '../../api/admin'
import { AdminIdentityProvidersPage } from '../../features/admin-identity-providers/AdminIdentityProvidersPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/identity-providers')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const connections = await listIdentityProviderConnections()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      connections,
    }
  },
  component: AdminIdentityProvidersRoute,
})

function AdminIdentityProvidersRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-identity-providers">
      <AdminIdentityProvidersPage {...data} />
    </PageMarker>
  )
}

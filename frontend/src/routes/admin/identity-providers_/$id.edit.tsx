import { createFileRoute } from '@tanstack/react-router'
import { AuthenticationAPIError, listIdentityProviderConnections } from '../../../api'
import { AdminIdentityProviderEditPage } from '../../../features/admin-identity-providers/AdminIdentityProviderFormPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/identity-providers_/$id/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const connections = await listIdentityProviderConnections()
    const connection = connections.find((item) => item.id === params.id)
    if (!connection) {
      throw new AuthenticationAPIError('The identity provider does not exist.', 'not_found')
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      connection,
    }
  },
  component: AdminIdentityProviderEditRoute,
})

function AdminIdentityProviderEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-identity-provider-edit">
      <AdminIdentityProviderEditPage {...data} />
    </PageMarker>
  )
}

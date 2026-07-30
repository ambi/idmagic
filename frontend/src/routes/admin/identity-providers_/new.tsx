import { createFileRoute } from '@tanstack/react-router'
import { AdminIdentityProviderCreatePage } from '../../../features/admin-identity-providers/AdminIdentityProviderFormPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/identity-providers_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminIdentityProviderCreateRoute,
})

function AdminIdentityProviderCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-identity-provider-create">
      <AdminIdentityProviderCreatePage {...data} />
    </PageMarker>
  )
}

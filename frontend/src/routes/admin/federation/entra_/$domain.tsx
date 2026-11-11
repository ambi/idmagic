import { createFileRoute } from '@tanstack/react-router'
import { getEntraFederationDomain } from '../../../../api/admin'
import { AdminEntraFederationDomainPage } from '../../../../features/admin-entra-federation/AdminEntraFederationDomainPage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/federation/entra_/$domain')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      detail: await getEntraFederationDomain(params.domain),
    }
  },
  component: AdminEntraFederationDomainRoute,
})

function AdminEntraFederationDomainRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-entra-federation-domain">
      <AdminEntraFederationDomainPage {...data} />
    </PageMarker>
  )
}

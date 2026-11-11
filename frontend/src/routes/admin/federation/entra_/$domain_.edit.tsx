import { createFileRoute } from '@tanstack/react-router'
import { getEntraFederationDomain } from '../../../../api/admin'
import { AdminEntraFederationEditPage } from '../../../../features/admin-entra-federation/AdminEntraFederationPage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/federation/entra_/$domain_/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const detail = await getEntraFederationDomain(params.domain)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      domain: detail.domain,
      sourceAnchorAttribute: detail.profile.source_anchor_attribute,
      replyURL: detail.reply_url,
    }
  },
  component: AdminEntraFederationEditRoute,
})

function AdminEntraFederationEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-entra-federation-edit">
      <AdminEntraFederationEditPage {...data} />
    </PageMarker>
  )
}

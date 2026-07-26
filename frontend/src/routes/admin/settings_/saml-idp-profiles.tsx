import { createFileRoute } from '@tanstack/react-router'
import { listSamlIDPProfiles } from '../../../api/admin'
import { AdminSamlIDPProfilesListPage } from '../../../features/admin-saml-idp-profiles/AdminSamlIDPProfilesListPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/settings_/saml-idp-profiles')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      actorUsername: account.preferred_username,
      profiles: await listSamlIDPProfiles(),
    }
  },
  component: AdminSamlIDPProfilesRoute,
})

function AdminSamlIDPProfilesRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-saml-idp-profiles">
      <AdminSamlIDPProfilesListPage {...data} />
    </PageMarker>
  )
}

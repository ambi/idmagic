import { createFileRoute } from '@tanstack/react-router'
import { AdminSamlIDPProfileCreatePage } from '../../../../features/admin-saml-idp-profiles/AdminSamlIDPProfileCreatePage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/settings_/saml-idp-profiles_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminSamlIDPProfileCreateRoute,
})

function AdminSamlIDPProfileCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-saml-idp-profile-create">
      <AdminSamlIDPProfileCreatePage {...data} />
    </PageMarker>
  )
}

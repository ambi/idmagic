import { createFileRoute } from '@tanstack/react-router'
import { AuthenticationAPIError } from '../../../../api/core'
import { listSamlIDPProfiles } from '../../../../api/admin'
import { AdminSamlIDPProfileDetailPage } from '../../../../features/admin-saml-idp-profiles/AdminSamlIDPProfileDetailPage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/settings_/saml-idp-profiles_/$profileId/')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const entry = (await listSamlIDPProfiles()).find(
      ({ profile }) => profile.profile_id === params.profileId,
    )
    if (!entry) throw new AuthenticationAPIError('SAML IdP profile not found.', 'not_found')
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      entry,
    }
  },
  component: AdminSamlIDPProfileDetailRoute,
})

function AdminSamlIDPProfileDetailRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-saml-idp-profile-detail">
      <AdminSamlIDPProfileDetailPage {...data} />
    </PageMarker>
  )
}

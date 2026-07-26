import { createFileRoute } from '@tanstack/react-router'
import { listSamlIDPProfiles } from '../../../../api/admin'
import { AuthenticationAPIError } from '../../../../api/core'
import { AdminSamlIDPProfileEditPage } from '../../../../features/admin-saml-idp-profiles/AdminSamlIDPProfileEditPage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/settings_/saml-idp-profiles_/$profileId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const entry = (await listSamlIDPProfiles()).find(
      ({ profile }) => profile.profile_id === params.profileId,
    )
    if (!entry) throw new AuthenticationAPIError('SAML IdP profile not found.', 'not_found')
    if (entry.profile.is_default) {
      throw new AuthenticationAPIError(
        'The default SAML IdP profile cannot be changed.',
        'default_idp_profile',
      )
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      entry,
    }
  },
  component: AdminSamlIDPProfileEditRoute,
})

function AdminSamlIDPProfileEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-saml-idp-profile-edit">
      <AdminSamlIDPProfileEditPage {...data} />
    </PageMarker>
  )
}

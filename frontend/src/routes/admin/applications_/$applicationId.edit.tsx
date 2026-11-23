import { createFileRoute } from '@tanstack/react-router'
import { getAdminApplication, getAppSignInPolicy, listSamlIDPProfiles } from '../../../api/admin'
import { AdminApplicationEditPage } from '../../../features/admin-applications/AdminApplicationEditPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/applications_/$applicationId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [detail, samlIDPProfiles, signInPolicy] = await Promise.all([
      getAdminApplication(params.applicationId),
      listSamlIDPProfiles(),
      getAppSignInPolicy(params.applicationId),
    ])
    detail.sign_in_policy = signInPolicy
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      detail,
      samlIDPProfiles,
    }
  },
  component: AdminApplicationEditRoute,
})

function AdminApplicationEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-application-edit">
      <AdminApplicationEditPage {...data} />
    </PageMarker>
  )
}

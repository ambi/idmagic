import { createFileRoute } from '@tanstack/react-router'
import {
  getAdminApplication,
  getAdminIntegrationEndpoints,
  listSamlIDPProfiles,
} from '../../../api/admin'
import { AdminApplicationDetailPage } from '../../../features/admin-applications/AdminApplicationDetailPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/applications_/$applicationId/')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [detail, integrationEndpoints, samlIDPProfiles] = await Promise.all([
      getAdminApplication(params.applicationId),
      getAdminIntegrationEndpoints(),
      listSamlIDPProfiles(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      detail,
      integrationEndpoints,
      samlIDPProfiles,
    }
  },
  component: AdminApplicationDetailRoute,
})

function AdminApplicationDetailRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-application-detail">
      <AdminApplicationDetailPage {...data} />
    </PageMarker>
  )
}

import { createFileRoute } from '@tanstack/react-router'
import { getAdminIntegrationEndpoints } from '../../api/admin'
import { AdminApiTokensPage } from '../../features/admin-settings/AdminApiTokensPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/api-tokens')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      integrationEndpoints: await getAdminIntegrationEndpoints(),
    }
  },
  component: () => (
    <PageMarker kind="admin-api-tokens">
      <AdminApiTokensPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})

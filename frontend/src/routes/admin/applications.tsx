import { createFileRoute } from '@tanstack/react-router'
import { listAdminApplicationsPage } from '../../api/admin'
import { AdminApplicationsPage } from '../../features/admin-applications/AdminApplicationsListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/applications')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const page = await listAdminApplicationsPage()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      applications: page.applications,
      nextCursor: page.nextCursor,
    }
  },
  component: AdminApplicationsRoute,
})

function AdminApplicationsRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-applications">
      <AdminApplicationsPage {...data} />
    </PageMarker>
  )
}

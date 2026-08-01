import { createFileRoute } from '@tanstack/react-router'
import { AdminApplicationCreatePage } from '../../../features/admin-applications/AdminApplicationCreatePage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/applications_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminApplicationCreateRoute,
})

function AdminApplicationCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-application-create">
      <AdminApplicationCreatePage {...data} />
    </PageMarker>
  )
}

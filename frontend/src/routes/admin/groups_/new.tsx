import { createFileRoute } from '@tanstack/react-router'
import { AdminGroupCreatePage } from '../../../features/admin-groups/AdminGroupsPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/groups_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminGroupCreateRoute,
})

function AdminGroupCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-group-create">
      <AdminGroupCreatePage {...data} />
    </PageMarker>
  )
}

import { createFileRoute } from '@tanstack/react-router'
import { listAdminGroupsPage } from '../../api'
import { AdminGroupsPage } from '../../features/admin-groups/AdminGroupsListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/groups')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const page = await listAdminGroupsPage()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      groups: page.groups,
      nextCursor: page.nextCursor,
    }
  },
  component: AdminGroupsRoute,
})

function AdminGroupsRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-groups">
      <AdminGroupsPage {...data} />
    </PageMarker>
  )
}

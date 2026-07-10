import { createFileRoute } from '@tanstack/react-router'
import { getAdminGroup } from '../../../api/admin'
import { AdminGroupEditPage } from '../../../features/admin-groups/AdminGroupsPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/groups_/$groupId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const { group } = await getAdminGroup(params.groupId)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      group,
    }
  },
  component: AdminGroupEditRoute,
})

function AdminGroupEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-group-edit">
      <AdminGroupEditPage {...data} />
    </PageMarker>
  )
}

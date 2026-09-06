import { createFileRoute } from '@tanstack/react-router'
import { AdminGroupMemberImportPage } from '../../../features/admin-groups/AdminGroupMemberImportPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/groups_/$groupId/members/import')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      groupId: params.groupId,
    }
  },
  component: AdminGroupMemberImportRoute,
})

function AdminGroupMemberImportRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-group-member-import">
      <AdminGroupMemberImportPage {...data} />
    </PageMarker>
  )
}

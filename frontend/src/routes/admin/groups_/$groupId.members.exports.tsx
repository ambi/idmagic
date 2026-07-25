import { createFileRoute } from '@tanstack/react-router'
import { groupMemberExports, tenantURL } from '../../../api'
import { DataExportPage } from '../../../features/admin-exports/DataExportPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/groups_/$groupId/members/exports')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      groupId: params.groupId,
    }
  },
  component: AdminGroupMembersExportRoute,
})

function AdminGroupMembersExportRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-group-members-export">
      <DataExportPage
        {...data}
        target="group_members"
        api={groupMemberExports(data.groupId)}
        backPath={tenantURL(`/admin/groups/${encodeURIComponent(data.groupId)}`)}
      />
    </PageMarker>
  )
}

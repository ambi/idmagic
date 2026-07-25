import { createFileRoute } from '@tanstack/react-router'
import { groupExports } from '../../../api'
import { DataExportPage } from '../../../features/admin-exports/DataExportPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/groups_/exports')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminGroupExportRoute,
})

function AdminGroupExportRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-group-export">
      <DataExportPage {...data} target="groups" api={groupExports} />
    </PageMarker>
  )
}

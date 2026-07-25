import { createFileRoute } from '@tanstack/react-router'
import { userExports } from '../../../api'
import { DataExportPage } from '../../../features/admin-exports/DataExportPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/users_/exports')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminUserExportRoute,
})

function AdminUserExportRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-user-export">
      <DataExportPage {...data} target="users" api={userExports} />
    </PageMarker>
  )
}

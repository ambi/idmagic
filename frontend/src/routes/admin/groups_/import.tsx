import { createFileRoute } from '@tanstack/react-router'
import { AdminGroupImportPage } from '../../../features/admin-groups/AdminGroupImportPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/groups_/import')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminGroupImportRoute,
})

function AdminGroupImportRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-group-import">
      <AdminGroupImportPage {...data} />
    </PageMarker>
  )
}

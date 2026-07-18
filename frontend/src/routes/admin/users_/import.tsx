import { createFileRoute } from '@tanstack/react-router'
import { AdminUserImportPage } from '../../../features/admin-users/AdminUserImportPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/users_/import')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminUserImportRoute,
})

function AdminUserImportRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-user-import">
      <AdminUserImportPage {...data} />
    </PageMarker>
  )
}

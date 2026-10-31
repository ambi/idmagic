import { createFileRoute } from '@tanstack/react-router'
import { AdminCrossAppAccessNewIssuePage } from '../../../features/admin-cross-app-access/AdminCrossAppAccessCreatePages'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/cross-app-access_/new-issue')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
    }
  },
  component: AdminCrossAppAccessNewIssueRoute,
})

function AdminCrossAppAccessNewIssueRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-cross-app-access-new-issue">
      <AdminCrossAppAccessNewIssuePage {...data} />
    </PageMarker>
  )
}

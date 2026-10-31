import { createFileRoute } from '@tanstack/react-router'
import { listTokenVaultConnectors } from '../../api/admin'
import { AdminTokenVaultPage } from '../../features/admin-token-vault/AdminTokenVaultPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/token-vault')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      connectors: await listTokenVaultConnectors(),
    }
  },
  component: AdminTokenVaultRoute,
})

function AdminTokenVaultRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-token-vault">
      <AdminTokenVaultPage {...data} />
    </PageMarker>
  )
}

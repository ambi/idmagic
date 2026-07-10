import { createFileRoute } from '@tanstack/react-router'
import { listTenantKeyHealth } from '../../api/admin'
import { SystemKeyHealthPage } from '../../features/admin-keys/SystemKeyHealthPage'
import { requireSystemAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/system/keys')({
  loader: async ({ location }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    const tenants = await listTenantKeyHealth()
    return {
      actorUsername: account.preferred_username,
      tenants,
    }
  },
  component: SystemKeyHealthRoute,
})

function SystemKeyHealthRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-key-health">
      <SystemKeyHealthPage {...data} />
    </PageMarker>
  )
}

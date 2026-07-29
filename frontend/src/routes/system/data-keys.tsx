import { createFileRoute } from '@tanstack/react-router'
import { listTenantDataKeyHealth } from '../../api/admin'
import { SystemDataKeyHealthPage } from '../../features/admin-data-keys/SystemDataKeyHealthPage'
import { requireSystemAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/system/data-keys')({
  loader: async ({ location }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    const tenants = await listTenantDataKeyHealth()
    return {
      actorUsername: account.preferred_username,
      tenants,
    }
  },
  component: SystemDataKeyHealthRoute,
})

function SystemDataKeyHealthRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-data-key-health">
      <SystemDataKeyHealthPage {...data} />
    </PageMarker>
  )
}

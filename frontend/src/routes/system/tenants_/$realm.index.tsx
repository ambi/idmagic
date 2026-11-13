import { createFileRoute } from '@tanstack/react-router'
import { getAdminTenant } from '../../../api'
import { SystemTenantDetailPage } from '../../../features/system-tenants/SystemTenantDetailPage'
import { requireSystemAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/system/tenants_/$realm/')({
  loader: async ({ location, params }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    const tenant = await getAdminTenant(params.realm)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username, tenant }
  },
  component: SystemTenantDetailPageRoute,
})

function SystemTenantDetailPageRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-tenant-detail">
      <SystemTenantDetailPage {...data} />
    </PageMarker>
  )
}

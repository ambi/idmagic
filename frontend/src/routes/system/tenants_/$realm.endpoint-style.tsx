import { createFileRoute } from '@tanstack/react-router'
import { getAdminTenant } from '../../../api'
import { SystemTenantEndpointStylePage } from '../../../features/system-tenants/SystemTenantEndpointStylePage'
import { requireSystemAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/system/tenants_/$realm/endpoint-style')({
  loader: async ({ location, params }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    const tenant = await getAdminTenant(params.realm)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username, tenant }
  },
  component: SystemTenantEndpointStylePageRoute,
})

function SystemTenantEndpointStylePageRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-tenant-endpoint-style">
      <SystemTenantEndpointStylePage {...data} />
    </PageMarker>
  )
}

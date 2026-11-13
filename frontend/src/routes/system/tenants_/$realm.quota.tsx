import { createFileRoute } from '@tanstack/react-router'
import { getAdminTenant } from '../../../api'
import { SystemTenantQuotaPage } from '../../../features/system-tenants/SystemTenantQuotaPage'
import { requireSystemAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/system/tenants_/$realm/quota')({
  loader: async ({ location, params }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    const tenant = await getAdminTenant(params.realm)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username, tenant }
  },
  component: SystemTenantQuotaPageRoute,
})

function SystemTenantQuotaPageRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-tenant-quota">
      <SystemTenantQuotaPage {...data} />
    </PageMarker>
  )
}

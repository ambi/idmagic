import { createFileRoute } from '@tanstack/react-router'
import { getAdminTenant } from '../../../api'
import { SystemTenantEditPage } from '../../../features/system-tenants/SystemTenantEditPage'
import { requireSystemAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/system/tenants_/$realm/edit')({
  loader: async ({ location, params }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    const tenant = await getAdminTenant(params.realm)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username, tenant }
  },
  component: SystemTenantEditPageRoute,
})

function SystemTenantEditPageRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-tenant-edit">
      <SystemTenantEditPage {...data} />
    </PageMarker>
  )
}

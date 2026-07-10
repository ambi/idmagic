import { createFileRoute } from '@tanstack/react-router'
import { request } from '../../api/core'
import { SystemTenantsPage } from '../../features/system-tenants/SystemTenantsPage'
import type { AdminTenant } from '../../types'
import { requireSystemAccount } from '../-guards'
import { PageMarker } from '../-page'

type AdminTenantListResponse = { tenants: AdminTenant[] }

export const Route = createFileRoute('/system/tenants')({
  loader: async ({ location }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    const tenants = await request<AdminTenantListResponse>('/api/admin/tenants')
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      tenants: tenants.tenants,
    }
  },
  component: SystemTenantsRoute,
})

function SystemTenantsRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-tenants">
      <SystemTenantsPage {...data} />
    </PageMarker>
  )
}

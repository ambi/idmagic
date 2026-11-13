import { createFileRoute } from '@tanstack/react-router'
import { SystemTenantCreatePage } from '../../features/system-tenants/SystemTenantCreatePage'
import { requireSystemAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/system/new-tenant')({
  loader: async ({ location }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username }
  },
  component: SystemTenantCreateRoute,
})

function SystemTenantCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-tenant-create">
      <SystemTenantCreatePage {...data} />
    </PageMarker>
  )
}

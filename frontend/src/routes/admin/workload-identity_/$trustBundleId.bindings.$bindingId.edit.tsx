import { createFileRoute, notFound } from '@tanstack/react-router'
import {
  getWorkloadTrustBundle,
  listAdminAgents,
  listAgentWorkloadBindings,
} from '../../../api/admin'
import { AdminWorkloadBindingEditPage } from '../../../features/admin-workload-identity/AdminWorkloadBindingEditPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute(
  '/admin/workload-identity_/$trustBundleId/bindings/$bindingId/edit',
)({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [trustBundle, bindings, agents] = await Promise.all([
      getWorkloadTrustBundle(params.trustBundleId),
      listAgentWorkloadBindings(params.trustBundleId),
      listAdminAgents(),
    ])
    const binding = bindings.find((item) => item.id === params.bindingId)
    if (!binding) throw notFound()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      trustBundle,
      binding,
      agents,
    }
  },
  component: BindingEditRoute,
})

function BindingEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-workload-binding-edit">
      <AdminWorkloadBindingEditPage {...data} />
    </PageMarker>
  )
}

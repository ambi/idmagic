import { createFileRoute } from '@tanstack/react-router'
import { AdminTenantGroupAttributeCreatePage } from '../../../../features/admin-tenants/AdminTenantGroupAttributeCreatePage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/tenant/group_attributes_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username }
  },
  component: () => (
    <PageMarker kind="admin-group-attribute-create">
      <AdminTenantGroupAttributeCreatePage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})

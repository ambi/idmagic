import { createFileRoute } from '@tanstack/react-router'
import { request } from '../../../api/core'
import { AdminTenantGroupAttributesPage } from '../../../features/admin-tenants/AdminTenantGroupAttributesPage'
import type { TenantGroupAttributeSchema } from '../../../types'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/tenant/group_attributes')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const schema = await request<TenantGroupAttributeSchema>(
      '/api/admin/v1/tenant/group_attribute_schema',
    )
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      schema,
    }
  },
  component: AdminTenantGroupAttributesRoute,
})

function AdminTenantGroupAttributesRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-tenant-group-attributes">
      <AdminTenantGroupAttributesPage {...data} />
    </PageMarker>
  )
}

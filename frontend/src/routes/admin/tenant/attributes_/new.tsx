import { createFileRoute } from '@tanstack/react-router'
import { request } from '../../../../api/core'
import { AdminTenantAttributeCreatePage } from '../../../../features/admin-tenants/AdminTenantAttributeCreatePage'
import type { TenantUserAttributeSchema } from '../../../../types'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/tenant/attributes_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const schema = await request<TenantUserAttributeSchema>(
      '/api/admin/tenant/user_attribute_schema',
    )
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      existingAttributes: schema.attributes,
    }
  },
  component: AdminTenantAttributeCreateRoute,
})

function AdminTenantAttributeCreateRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-tenant-attribute-create">
      <AdminTenantAttributeCreatePage {...data} />
    </PageMarker>
  )
}

import { createFileRoute, useRouter } from '@tanstack/react-router'
import { findAuthorizationModel } from '../../../../api/admin'
import { AdminAuthorizationTupleCreatePage } from '../../../../features/admin-authorization/AdminAuthorizationTupleCreatePage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/authorization/relation-tuples_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const published = await findAuthorizationModel()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      resourceTypes: published?.authorization_model.resource_types ?? null,
    }
  },
  component: AdminAuthorizationTupleCreateRoute,
})

function AdminAuthorizationTupleCreateRoute() {
  const data = Route.useLoaderData()
  const router = useRouter()
  return (
    <PageMarker kind="admin-authorization-tuple-create">
      <AdminAuthorizationTupleCreatePage
        {...data}
        onCreated={() =>
          void router.navigate({
            to: '/admin/authorization/relation-tuples',
            state: { authorizationNotice: 'tuple-added' },
          })
        }
      />
    </PageMarker>
  )
}

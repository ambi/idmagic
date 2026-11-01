import { createFileRoute, redirect, useRouter } from '@tanstack/react-router'
import { findAuthorizationModel } from '../../../../api/admin'
import { AdminAuthorizationModelEditPage } from '../../../../features/admin-authorization/AdminAuthorizationModelEditPage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/authorization/model_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    if (await findAuthorizationModel()) throw redirect({ to: '/admin/authorization/model' })
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username }
  },
  component: CreateRoute,
})
function CreateRoute() {
  const router = useRouter()
  return (
    <PageMarker kind="admin-authorization-model-create">
      <AdminAuthorizationModelEditPage
        {...Route.useLoaderData()}
        model={null}
        onPublished={async () => {
          await router.invalidate()
          await router.navigate({
            to: '/admin/authorization/model',
            state: { authorizationNotice: 'published' },
          })
        }}
      />
    </PageMarker>
  )
}

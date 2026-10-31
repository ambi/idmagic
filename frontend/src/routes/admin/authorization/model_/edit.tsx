import { createFileRoute, useRouter } from '@tanstack/react-router'
import { findAuthorizationModel } from '../../../../api/admin'
import { AdminAuthorizationModelEditPage } from '../../../../features/admin-authorization/AdminAuthorizationModelEditPage'
import { requirePortalAccount } from '../../../-guards'
import { PageMarker } from '../../../-page'

export const Route = createFileRoute('/admin/authorization/model_/edit')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const published = await findAuthorizationModel()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      model: published?.authorization_model ?? null,
    }
  },
  component: AdminAuthorizationModelEditRoute,
})

function AdminAuthorizationModelEditRoute() {
  const data = Route.useLoaderData()
  const router = useRouter()
  return (
    <PageMarker kind="admin-authorization-model-edit">
      <AdminAuthorizationModelEditPage
        key={data.model?.version ?? 0}
        {...data}
        onPublished={() => void router.navigate({ to: '/admin/authorization/model' })}
      />
    </PageMarker>
  )
}

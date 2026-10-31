import { createFileRoute } from '@tanstack/react-router'
import { findAuthorizationModel } from '../../../api/admin'
import { AdminAuthorizationModelPage } from '../../../features/admin-authorization/AdminAuthorizationModelPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export function validateModelSearch(search: Record<string, unknown>): { version?: number } {
  const version = search.version
  return typeof version === 'number' && Number.isInteger(version) && version > 0 ? { version } : {}
}

export const Route = createFileRoute('/admin/authorization/model')({
  validateSearch: validateModelSearch,
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const published = await findAuthorizationModel(deps.version)
    return {
      actorUsername: account.preferred_username,
      model: published?.authorization_model ?? null,
      requestedVersion: deps.version,
    }
  },
  component: AdminAuthorizationModelRoute,
})

function AdminAuthorizationModelRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-authorization-model">
      <AdminAuthorizationModelPage {...data} />
    </PageMarker>
  )
}

import { createFileRoute } from '@tanstack/react-router'
import { findAuthorizationModel } from '../../../api/admin'
import { AdminAuthorizationCheckPage } from '../../../features/admin-authorization/AdminAuthorizationCheckPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export function validateCheckSearch(search: Record<string, unknown>): { consistency?: string } {
  return typeof search.consistency === 'string' && search.consistency
    ? { consistency: search.consistency }
    : {}
}

export const Route = createFileRoute('/admin/authorization/check')({
  validateSearch: validateCheckSearch,
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const published = await findAuthorizationModel()
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      // モデルは入力候補に使うだけなので、未登録でも画面を開ける。判定は API が拒否する。
      resourceTypes: published?.authorization_model.resource_types ?? [],
      consistency: deps.consistency ?? '',
    }
  },
  component: AdminAuthorizationCheckRoute,
})

function AdminAuthorizationCheckRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-authorization-check">
      <AdminAuthorizationCheckPage key={data.consistency} {...data} />
    </PageMarker>
  )
}

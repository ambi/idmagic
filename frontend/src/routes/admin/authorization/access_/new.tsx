import { createFileRoute, useRouter } from '@tanstack/react-router'
import { AdminAgentAccessPage } from '../../../../features/admin-authorization/AdminAgentAccessPage'
import { PageMarker } from '../../../-page'
import { loadAgentAccess, validateAccessSearch } from '../-access'

export const Route = createFileRoute('/admin/authorization/access_/new')({
  validateSearch: validateAccessSearch,
  loaderDeps: ({ search }) => search,
  loader: ({ location, deps }) =>
    loadAgentAccess(location.pathname, location.searchStr, deps.agent),
  component: CreateAccessRoute,
})

function CreateAccessRoute() {
  const data = Route.useLoaderData()
  const search = Route.useSearch()
  const router = useRouter()
  return (
    <PageMarker kind="admin-agent-access-create">
      <AdminAgentAccessPage
        {...data}
        mode="create"
        selectedServer={search.server}
        selectedAgent={search.agent}
        onGranted={(selection) =>
          void router.navigate({
            to: '/admin/authorization/access',
            search: selection,
            state: { authorizationNotice: 'tuple-added' },
          })
        }
      />
    </PageMarker>
  )
}

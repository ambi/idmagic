import { createFileRoute } from '@tanstack/react-router'
import { AdminAgentAccessPage } from '../../../features/admin-authorization/AdminAgentAccessPage'
import { PageMarker } from '../../-page'
import { loadAgentAccess, validateAccessSearch } from './-access'
import { useAuthorizationNotice } from './-notice'

export const Route = createFileRoute('/admin/authorization/access')({
  validateSearch: validateAccessSearch,
  loaderDeps: ({ search }) => search,
  loader: ({ location, deps }) =>
    loadAgentAccess(location.pathname, location.searchStr, deps.agent),
  component: AccessRoute,
})

function AccessRoute() {
  const data = Route.useLoaderData()
  const search = Route.useSearch()
  const notice = useAuthorizationNotice()
  return (
    <PageMarker kind="admin-agent-access">
      <AdminAgentAccessPage
        key={`${data.consistency}:${search.server}:${search.agent}`}
        {...data}
        mode="list"
        selectedServer={search.server}
        selectedAgent={search.agent}
        justGranted={notice === 'tuple-added'}
      />
    </PageMarker>
  )
}

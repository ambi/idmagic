import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import { AuthenticationAPIError, listAdminAgentsPage } from '../../api'
import { AdminAgentsPage } from '../../features/admin-agents/AdminAgentsListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/agents')({
  validateSearch: (search: Record<string, unknown>) => ({
    ...(typeof search.cursor === 'string' && search.cursor ? { cursor: search.cursor } : {}),
  }),
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    let cursorReset = false
    let page: Awaited<ReturnType<typeof listAdminAgentsPage>>
    try {
      page = await listAdminAgentsPage(deps)
    } catch (cause) {
      if (
        !deps.cursor ||
        !(cause instanceof AuthenticationAPIError) ||
        cause.code !== 'invalid_request'
      )
        throw cause
      cursorReset = true
      page = await listAdminAgentsPage()
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      agents: page.agents,
      previousCursor: page.previousCursor,
      nextCursor: page.nextCursor,
      cursorReset,
    }
  },
  component: AdminAgentsRoute,
})

function AdminAgentsRoute() {
  const data = Route.useLoaderData()
  const navigate = Route.useNavigate()
  const search = Route.useSearch()
  useEffect(() => {
    if (data.cursorReset && search.cursor) void navigate({ replace: true, search: {} })
  }, [data.cursorReset, navigate, search.cursor])
  return (
    <PageMarker kind="admin-agents">
      <AdminAgentsPage
        {...data}
        key={search.cursor ?? ''}
        onPage={(cursor) => navigate({ search: { cursor } })}
      />
    </PageMarker>
  )
}

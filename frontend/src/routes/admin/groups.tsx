import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import { AuthenticationAPIError, listAdminGroupsPage } from '../../api'
import { AdminGroupsPage } from '../../features/admin-groups/AdminGroupsListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/groups')({
  validateSearch: (search: Record<string, unknown>) => ({
    ...(typeof search.cursor === 'string' && search.cursor ? { cursor: search.cursor } : {}),
  }),
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    let cursorReset = false
    let page: Awaited<ReturnType<typeof listAdminGroupsPage>>
    try {
      page = await listAdminGroupsPage(deps)
    } catch (cause) {
      if (
        !deps.cursor ||
        !(cause instanceof AuthenticationAPIError) ||
        cause.code !== 'invalid_request'
      )
        throw cause
      cursorReset = true
      page = await listAdminGroupsPage()
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      ...page,
      cursorReset,
    }
  },
  component: AdminGroupsRoute,
})

function AdminGroupsRoute() {
  const data = Route.useLoaderData()
  const navigate = Route.useNavigate()
  const search = Route.useSearch()
  useEffect(() => {
    if (data.cursorReset && search.cursor) void navigate({ replace: true, search: {} })
  }, [data.cursorReset, navigate, search.cursor])
  const navigatePage = (cursor: string | null) => navigate({ search: cursor ? { cursor } : {} })
  return (
    <PageMarker kind="admin-groups">
      <AdminGroupsPage
        {...data}
        pagination={data}
        cursor={search.cursor}
        key={search.cursor ?? ''}
        onPage={navigatePage}
      />
    </PageMarker>
  )
}

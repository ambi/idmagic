import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import { AuthenticationAPIError } from '../../api'
import { listAdminApplicationsPage } from '../../api/admin'
import { AdminApplicationsPage } from '../../features/admin-applications/AdminApplicationsListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/applications')({
  validateSearch: (search: Record<string, unknown>) => ({
    ...(typeof search.cursor === 'string' && search.cursor ? { cursor: search.cursor } : {}),
  }),
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    let cursorReset = false
    let page: Awaited<ReturnType<typeof listAdminApplicationsPage>>
    try {
      page = await listAdminApplicationsPage(deps)
    } catch (cause) {
      if (
        !deps.cursor ||
        !(cause instanceof AuthenticationAPIError) ||
        cause.code !== 'invalid_request'
      )
        throw cause
      cursorReset = true
      page = await listAdminApplicationsPage()
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      ...page,
      cursorReset,
    }
  },
  component: AdminApplicationsRoute,
})

function AdminApplicationsRoute() {
  const data = Route.useLoaderData()
  const navigate = Route.useNavigate()
  const search = Route.useSearch()
  useEffect(() => {
    if (data.cursorReset && search.cursor) void navigate({ replace: true, search: {} })
  }, [data.cursorReset, navigate, search.cursor])
  const navigatePage = (cursor: string | null) => navigate({ search: cursor ? { cursor } : {} })
  return (
    <PageMarker kind="admin-applications">
      <AdminApplicationsPage
        {...data}
        pagination={data}
        cursor={search.cursor}
        key={search.cursor ?? ''}
        onPage={navigatePage}
      />
    </PageMarker>
  )
}

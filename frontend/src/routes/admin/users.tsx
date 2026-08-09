import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import { AuthenticationAPIError, listAdminUsersPage } from '../../api'
import { AdminUsersPage } from '../../features/admin-users/AdminUsersListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export function validateAdminUsersSearch(search: Record<string, unknown>) {
  return {
    ...(typeof search.cursor === 'string' && search.cursor ? { cursor: search.cursor } : {}),
    ...(typeof search.query === 'string' && search.query ? { query: search.query } : {}),
    ...(typeof search.status === 'string' &&
    ['all', 'active', 'disabled', 'pending_deletion'].includes(search.status)
      ? { status: search.status }
      : {}),
  }
}

export function adminUsersAPIStatus(status?: string): string | undefined {
  return status === 'all' ? undefined : (status ?? 'active')
}

export const Route = createFileRoute('/admin/users')({
  validateSearch: validateAdminUsersSearch,
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    let cursorReset = false
    const apiStatus = adminUsersAPIStatus(deps.status)
    let page: Awaited<ReturnType<typeof listAdminUsersPage>>
    try {
      page = await listAdminUsersPage({ ...deps, status: apiStatus })
    } catch (cause) {
      if (
        !deps.cursor ||
        !(cause instanceof AuthenticationAPIError) ||
        cause.code !== 'invalid_request'
      )
        throw cause
      cursorReset = true
      page = await listAdminUsersPage({ query: deps.query, status: apiStatus })
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      ...page,
      query: deps.query ?? '',
      status: deps.status ?? 'active',
      cursorReset,
    }
  },
  component: AdminUsersRoute,
})

function AdminUsersRoute() {
  const data = Route.useLoaderData()
  const navigate = Route.useNavigate()
  const search = Route.useSearch()
  useEffect(() => {
    if (!data.cursorReset || !search.cursor) return
    void navigate({
      replace: true,
      search: { query: search.query, status: search.status },
    })
  }, [data.cursorReset, navigate, search.cursor, search.query, search.status])
  useEffect(() => {
    if (search.status !== 'active') return
    void navigate({ replace: true, search: { query: search.query, cursor: search.cursor } })
  }, [navigate, search.cursor, search.query, search.status])
  const navigatePage = (cursor: string | null) => {
    if (cursor) return navigate({ search: { ...search, cursor } })
    const { cursor: _cursor, ...withoutCursor } = search
    return navigate({ search: withoutCursor })
  }
  return (
    <PageMarker kind="admin-users">
      <AdminUsersPage
        {...data}
        pagination={data}
        key={`${search.cursor ?? ''}:${search.query ?? ''}:${search.status ?? ''}`}
        onPage={navigatePage}
        onFilter={(next) => navigate({ search: { ...next } })}
      />
    </PageMarker>
  )
}

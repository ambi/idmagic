import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import { AuthenticationAPIError, getAdminSettings, listAdminUsersPage } from '../../api'
import { AdminUsersPage } from '../../features/admin-users/AdminUsersListPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/users')({
  validateSearch: (search: Record<string, unknown>) => ({
    ...(typeof search.cursor === 'string' && search.cursor ? { cursor: search.cursor } : {}),
    ...(typeof search.query === 'string' && search.query ? { query: search.query } : {}),
    ...(typeof search.status === 'string' &&
    ['active', 'disabled', 'pending_deletion'].includes(search.status)
      ? { status: search.status }
      : {}),
  }),
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    let cursorReset = false
    const settingsPromise = getAdminSettings().catch(() => undefined)
    let page: Awaited<ReturnType<typeof listAdminUsersPage>>
    try {
      page = await listAdminUsersPage(deps)
    } catch (cause) {
      if (
        !deps.cursor ||
        !(cause instanceof AuthenticationAPIError) ||
        cause.code !== 'invalid_request'
      )
        throw cause
      cursorReset = true
      page = await listAdminUsersPage({ query: deps.query, status: deps.status })
    }
    const settings = await settingsPromise
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      users: page.users,
      previousCursor: page.previousCursor,
      nextCursor: page.nextCursor,
      query: deps.query ?? '',
      status: deps.status ?? '',
      usageUsers: settings?.usage?.users,
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
  return (
    <PageMarker kind="admin-users">
      <AdminUsersPage
        {...data}
        key={`${search.cursor ?? ''}:${search.query ?? ''}:${search.status ?? ''}`}
        onPage={(cursor) => navigate({ search: { ...search, cursor } })}
        onFilter={(next) => navigate({ search: { ...next } })}
      />
    </PageMarker>
  )
}

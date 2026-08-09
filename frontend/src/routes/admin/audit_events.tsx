import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import {
  type AdminAuditEventCategory,
  type AdminAuditEventsSearchParams,
  AuthenticationAPIError,
  listAdminAuditEvents,
  listAdminAuditEventSearchOptions,
} from '../../api'
import { AdminAuditEventsPage } from '../../features/admin-audit-events/AdminAuditEventsPage'
import type { AdminAuditEvent } from '../../types'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

const AUDIT_EVENT_CATEGORIES: AdminAuditEventCategory[] = [
  'authentication',
  'success',
  'fail',
  'aggregated',
  'user',
  'group',
  'client',
  'consent',
  'token',
  'tenant',
  'key',
]

// URL query string を検索フォームの正とする (wi-147)。想定外の値は静かに無視し、
// 監査イベント検索を壊さない (壊れた URL でも空条件として動作する)。
export function validateAuditEventsSearch(
  search: Record<string, unknown>,
): AdminAuditEventsSearchParams {
  const result: AdminAuditEventsSearchParams = {}
  if (
    typeof search.category === 'string' &&
    (AUDIT_EVENT_CATEGORIES as string[]).includes(search.category)
  ) {
    result.category = search.category as AdminAuditEventCategory
  }
  if (typeof search.sub === 'string' && search.sub) result.sub = search.sub
  if (typeof search.username === 'string' && search.username) result.username = search.username
  if (typeof search.after === 'string' && search.after) result.after = search.after
  if (typeof search.before === 'string' && search.before) result.before = search.before
  if (typeof search.limit === 'number' && Number.isFinite(search.limit)) {
    result.limit = search.limit
  }
  if (typeof search.cursor === 'string' && search.cursor) result.cursor = search.cursor
  if (search.allTenants === true) result.allTenants = true
  if (Array.isArray(search.filter)) {
    const filter = search.filter.filter((v): v is string => typeof v === 'string')
    if (filter.length > 0) result.filter = filter
  }
  return result
}

export const Route = createFileRoute('/admin/audit_events')({
  validateSearch: validateAuditEventsSearch,
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    // 検索条件 (URL) に起因する取得失敗 (例: 不正な値による 4xx) はページ全体を壊さず、
    // ページ内のエラー表示に留める。認証そのものの失敗は requirePortalAccount 側で扱う (wi-147)。
    let events: AdminAuditEvent[] = []
    let nextCursor: string | null = null
    let previousCursor: string | null = null
    let lastCursor: string | null = null
    let hasFirst = false
    let totalItems = 0
    let totalPages = 0
    let currentPage = 0
    let pageSize = deps.limit ?? 100
    let cursorReset = false
    let searchError = ''
    try {
      const page = await listAdminAuditEvents(deps)
      events = page.events
      previousCursor = page.previousCursor
      nextCursor = page.nextCursor
      lastCursor = page.lastCursor
      hasFirst = page.hasFirst
      totalItems = page.totalItems
      totalPages = page.totalPages
      currentPage = page.currentPage
      pageSize = page.pageSize
    } catch (cause) {
      if (
        deps.cursor &&
        cause instanceof AuthenticationAPIError &&
        cause.code === 'invalid_request'
      ) {
        const { cursor: _cursor, ...withoutCursor } = deps
        const page = await listAdminAuditEvents(withoutCursor)
        events = page.events
        previousCursor = page.previousCursor
        nextCursor = page.nextCursor
        lastCursor = page.lastCursor
        hasFirst = page.hasFirst
        totalItems = page.totalItems
        totalPages = page.totalPages
        currentPage = page.currentPage
        pageSize = page.pageSize
        cursorReset = true
      } else {
        searchError = cause instanceof AuthenticationAPIError ? cause.message : String(cause)
      }
    }
    const searchOptions = await listAdminAuditEventSearchOptions().catch(() => undefined)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      actorRoles: account.roles ?? [],
      actorRealm: account.realm ?? '',
      events,
      previousCursor,
      nextCursor,
      lastCursor,
      hasFirst,
      totalItems,
      totalPages,
      currentPage,
      pageSize,
      search: deps,
      searchOptions,
      initialError: searchError,
      cursorReset,
    }
  },
  component: AdminAuditEventsRoute,
})

function AdminAuditEventsRoute() {
  const data = Route.useLoaderData()
  const navigate = Route.useNavigate()
  const search = Route.useSearch()
  useEffect(() => {
    if (!data.cursorReset || !search.cursor) return
    const { cursor: _cursor, ...withoutCursor } = search
    void navigate({ replace: true, search: withoutCursor })
  }, [data.cursorReset, navigate, search])
  const navigatePage = (cursor: string | null) => {
    if (cursor) return navigate({ search: { ...search, cursor } })
    const { cursor: _cursor, ...withoutCursor } = search
    return navigate({ search: withoutCursor })
  }
  return (
    <PageMarker kind="admin-audit-events">
      <AdminAuditEventsPage
        key={JSON.stringify(search)}
        {...data}
        pagination={data}
        onSearch={(next) => navigate({ search: next })}
        onPage={navigatePage}
      />
    </PageMarker>
  )
}

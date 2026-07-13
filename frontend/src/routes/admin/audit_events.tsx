import { createFileRoute } from '@tanstack/react-router'
import {
  type AdminAuditEventCategory,
  type AdminAuditEventsSearchParams,
  AuthenticationAPIError,
  listAdminAuditEvents,
  listAdminAuditEventSearchOptions,
} from '../../api'
import { AdminAuditEventsPage } from '../../features/admin-audit-events/AdminAuditEventsPage'
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
    let events: Awaited<ReturnType<typeof listAdminAuditEvents>> = []
    let searchError = ''
    try {
      events = await listAdminAuditEvents(deps)
    } catch (cause) {
      searchError = cause instanceof AuthenticationAPIError ? cause.message : String(cause)
    }
    const searchOptions = await listAdminAuditEventSearchOptions().catch(() => undefined)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      actorRoles: account.roles ?? [],
      actorRealm: account.realm ?? '',
      events,
      search: deps,
      searchOptions,
      initialError: searchError,
    }
  },
  component: AdminAuditEventsRoute,
})

function AdminAuditEventsRoute() {
  const data = Route.useLoaderData()
  const navigate = Route.useNavigate()
  return (
    <PageMarker kind="admin-audit-events">
      <AdminAuditEventsPage {...data} onSearch={(next) => navigate({ search: next })} />
    </PageMarker>
  )
}

import { createFileRoute } from '@tanstack/react-router'
import { listAdminConsents, listAdminUsers } from '../../api'
import { request } from '../../api/core'
import { AdminDashboardPage } from '../../features/admin-dashboard/AdminDashboardPage'
import type { AdminOAuth2Client, AdminSettings } from '../../types'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

type AdminOAuth2ClientListResponse = { clients: AdminOAuth2Client[] }

// Dashboard タイル用の一覧 API はどれも大規模テナントで全件取得できない。
// userCount/clientCount は settings.usage (tenant usage summary、専用の集計クエリ) を優先し、
// usage が無い場合のみ読み込み済み件数にフォールバックする。activeUserCount/
// disabledUserCount/grantedConsentCount は breakdown 用の summary endpoint が無いため、
// listAdminUsers/listAdminConsents (PICKER_LIST_LIMIT=200 で capped) から計算する近似値
// ("capped query" — WI Design が dashboard 向けに明示的に許容する方式)。200 件を超える
// テナントではこれらのタイルは総数と一致しない。
export const Route = createFileRoute('/admin/')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [users, clients, consents, settings] = await Promise.all([
      listAdminUsers(),
      request<AdminOAuth2ClientListResponse>('/api/admin/v1/clients?limit=200'),
      listAdminConsents(),
      request<AdminSettings>('/api/admin/v1/settings'),
    ])
    const activeUserCount = users.filter((u) => !u.disabled_at).length
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      actorRoles: account.roles ?? [],
      actorRealm: account.realm ?? '',
      userCount: settings.usage?.users ?? users.length,
      activeUserCount,
      disabledUserCount: users.length - activeUserCount,
      clientCount: settings.usage?.oauth2_clients ?? clients.clients.length,
      grantedConsentCount: consents.filter((c) => c.state === 'granted').length,
      quota: settings.quota,
      usage: settings.usage,
    }
  },
  component: AdminDashboardRoute,
})

function AdminDashboardRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-dashboard">
      <AdminDashboardPage {...data} />
    </PageMarker>
  )
}

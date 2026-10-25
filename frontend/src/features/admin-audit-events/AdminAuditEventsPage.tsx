import {
  type AdminAuditEventSearchOptions,
  type AdminAuditEventsSearchParams,
  adminAuditExports,
  listAdminAuditEvents,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import type { PageNavigationData } from '../../components/ui/page-navigation'
import { useDictionary } from '../../lib/i18n'
import type { AdminAuditEvent } from '../../types'
import { adminAuditEventsDictionary } from './AdminAuditEventsPage.i18n'
import { AuditEventsBrowser } from './AuditEventsBrowser'

// AdminAuditEventsPage はテナント管理コンソールの監査ログ画面。範囲は常に要求先テナントで
// あり、操作者が `system_admin` であっても横断しない。横断はシステムコンソールが持つ
// (REQ-SYSTEM-020)。そのため、このページはロールも realm も見ない。
export function AdminAuditEventsPage({
  csrfToken = '',
  actorUsername,
  events,
  pagination,
  previousCursor = null,
  nextCursor,
  search,
  searchOptions,
  onSearch,
  onPage,
  cursorReset = false,
  initialError,
}: {
  csrfToken?: string
  actorUsername?: string
  events: AdminAuditEvent[]
  pagination?: Omit<PageNavigationData, 'nextCursor' | 'previousCursor'>
  previousCursor?: string | null
  nextCursor: string | null
  search?: AdminAuditEventsSearchParams
  searchOptions?: AdminAuditEventSearchOptions
  onSearch?: (search: AdminAuditEventsSearchParams) => void
  onPage?: (cursor: string | null) => void
  cursorReset?: boolean
  initialError?: string
}) {
  const t = useDictionary(adminAuditEventsDictionary)
  return (
    <AdminShell
      active="audit-events"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      <AuditEventsBrowser
        events={events}
        pagination={pagination}
        previousCursor={previousCursor}
        nextCursor={nextCursor}
        search={search}
        searchOptions={searchOptions}
        listEvents={listAdminAuditEvents}
        exportClient={adminAuditExports}
        csrfToken={csrfToken}
        onSearch={onSearch}
        onPage={onPage}
        cursorReset={cursorReset}
        initialError={initialError}
      />
    </AdminShell>
  )
}

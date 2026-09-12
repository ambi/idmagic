import {
  type AdminAuditEventSearchOptions,
  type AdminAuditEventsSearchParams,
  listSystemAuditEvents,
  systemAuditEventsExportURL,
} from '../../api'
import { SystemShell } from '../../components/SystemShell'
import type { PageNavigationData } from '../../components/ui/page-navigation'
import { useDictionary } from '../../lib/i18n'
import type { AdminAuditEvent } from '../../types'
import { adminAuditEventsDictionary } from './AdminAuditEventsPage.i18n'
import { AuditEventsBrowser } from './AuditEventsBrowser'

// SystemAuditEventsPage はシステムコンソールの監査ログ画面。範囲は全テナントに固定され、
// 切り替える UI を持たない。検索、ページング、エクスポートはいずれも専用のシステム API を
// 呼ぶ (REQ-AUDIT-007 / REQ-SYSTEM-020)。
export function SystemAuditEventsPage({
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
    <SystemShell
      active="audit-events"
      actorUsername={actorUsername}
      title={t.systemPageTitle}
      description={t.systemPageDescription}
    >
      <AuditEventsBrowser
        events={events}
        pagination={pagination}
        previousCursor={previousCursor}
        nextCursor={nextCursor}
        search={search}
        searchOptions={searchOptions}
        listEvents={listSystemAuditEvents}
        exportURL={systemAuditEventsExportURL}
        onSearch={onSearch}
        onPage={onPage}
        cursorReset={cursorReset}
        initialError={initialError}
      />
    </SystemShell>
  )
}

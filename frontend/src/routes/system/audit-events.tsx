import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import {
  AuthenticationAPIError,
  listAdminAuditEventSearchOptions,
  listSystemAuditEvents,
} from '../../api'
import { SystemAuditEventsPage } from '../../features/admin-audit-events/SystemAuditEventsPage'
import type { AdminAuditEvent } from '../../types'
import { validateAuditEventsSearch } from '../admin/audit_events'
import { requireSystemAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/system/audit-events')({
  validateSearch: validateAuditEventsSearch,
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    // 検索条件 (URL) に起因する取得失敗はページ全体を壊さず、ページ内のエラー表示に留める。
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
      const page = await listSystemAuditEvents(deps)
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
      // テナント管理経路で発行したカーソルはシステム経路の続きにならない。持ち込まれた
      // ときは先頭ページへ戻し、画面を行き止まりにしない。
      if (
        deps.cursor &&
        cause instanceof AuthenticationAPIError &&
        cause.code === 'invalid_request'
      ) {
        const { cursor: _cursor, ...withoutCursor } = deps
        const page = await listSystemAuditEvents(withoutCursor)
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
      actorUsername: account.preferred_username,
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
  component: SystemAuditEventsRoute,
})

function SystemAuditEventsRoute() {
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
    <PageMarker kind="system-audit-events">
      <SystemAuditEventsPage
        key={JSON.stringify(search)}
        {...data}
        pagination={data}
        onSearch={(next) => navigate({ search: next })}
        onPage={navigatePage}
      />
    </PageMarker>
  )
}

import { useState } from 'react'
import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAuditEventsPage } from './AdminAuditEventsPage'
import { adminAuditEventsDictionary } from './AdminAuditEventsPage.i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { friendlyEventName } from '../admin-dashboard/AdminDashboardPage.i18n'
import type { AdminAuditEventsSearchParams } from '../../api'
import type { AdminAuditEvent } from '../../types'

const t = adminAuditEventsDictionary.en
const tCommon = commonDictionary.en

const response = (status: number, body: unknown = {}, headers: Record<string, string> = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
  headers: new Headers(headers),
})

const event: AdminAuditEvent = {
  id: 'evt-1',
  tenant_id: 'tenant-1',
  type: 'UserAuthenticated',
  occurred_at: '2026-01-01T00:00:00Z',
  payload: { foo: 'bar' },
}

const previousEvent: AdminAuditEvent = {
  ...event,
  id: 'evt-previous',
  type: 'AuthenticationFailed',
}

function BrowserHistoryHarness() {
  const [routeData, setRouteData] = useState({
    events: [event],
    search: { sub: 'usr_current' } as AdminAuditEventsSearchParams,
  })
  return (
    <>
      <button
        type="button"
        onClick={() => setRouteData({ events: [previousEvent], search: { sub: 'usr_previous' } })}
      >
        Simulate browser back
      </button>
      <AdminAuditEventsPage
        key={JSON.stringify(routeData.search)}
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={routeData.events}
        nextCursor={null}
        search={routeData.search}
      />
    </>
  )
}

describe('locale', () => {
  afterEach(() => restoreGlobals())

  it('renders the audit events page in English by default', async () => {
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[]}
        nextCursor={null}
      />,
    )
    expect(
      screen.getByRole('heading', { name: adminAuditEventsDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(adminAuditEventsDictionary.en.noMatchingEventsNotice),
    ).toBeInTheDocument()
  })

  it('renders the audit events page in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[]}
        nextCursor={null}
      />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: adminAuditEventsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminAuditEventsPage', () => {
  afterEach(() => restoreGlobals())

  it('shows an empty state when a filtered query returns no events', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { events: [] }))),
    )
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
        nextCursor={null}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.filterAction }))

    expect(await screen.findByText(t.noMatchingEventsNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectEventPrompt)).toBeInTheDocument()
  })

  it('shows an error when querying audit events fails', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(500, { message: 'Could not fetch audit events.' }))),
    )
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
        nextCursor={null}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.filterAction }))

    expect(await screen.findByText('Could not fetch audit events.')).toBeInTheDocument()
    expect(screen.getAllByText(friendlyEventName(event.type, 'en')).length).toBeGreaterThan(0)
  })

  it('initializes the search conditions from the search prop (URL query init, wi-147)', async () => {
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
        nextCursor={null}
        search={{ category: 'authentication', sub: 'usr_from_url' }}
      />,
    )

    // wi-147: category / sub は他の条件と同じ1つの検索条件一覧の行として表示される。
    expect(screen.getByDisplayValue('usr_from_url')).toBeInTheDocument()
    const categoryValueSelect = screen.getAllByRole('combobox')[1] as HTMLSelectElement
    expect(categoryValueSelect.value).toBe('authentication')
  })

  it('restores search conditions and loader results when browser history changes the URL', async () => {
    await renderWithRouter(<BrowserHistoryHarness />)

    expect(screen.getByDisplayValue('usr_current')).toBeInTheDocument()
    expect(screen.getAllByText(friendlyEventName(event.type, 'en')).length).toBeGreaterThan(0)

    fireEvent.click(screen.getByRole('button', { name: 'Simulate browser back' }))

    expect(await screen.findByDisplayValue('usr_previous')).toBeInTheDocument()
    expect(screen.queryByDisplayValue('usr_current')).not.toBeInTheDocument()
    expect(screen.getAllByText(friendlyEventName(previousEvent.type, 'en')).length).toBeGreaterThan(
      0,
    )
    expect(screen.queryByText(event.type)).not.toBeInTheDocument()
  })

  it('calls onSearch with the built query on submit (URL update on search execution, wi-147)', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { events: [] }))),
    )
    const onSearch = mock()
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
        nextCursor={null}
        onSearch={onSearch}
      />,
    )

    // 既定行の種類を「ユーザー ID (操作者)」に切り替えてから値を入力する。
    const fieldTypeSelect = screen.getAllByRole('combobox')[0] as HTMLSelectElement
    fireEvent.change(fieldTypeSelect, { target: { value: 'quick.actor_id' } })
    fireEvent.change(screen.getByPlaceholderText(t.actorUserIdFieldPlaceholder), {
      target: { value: 'usr_submitted' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.filterAction }))

    expect(onSearch).toHaveBeenCalledWith(expect.objectContaining({ sub: 'usr_submitted' }))
    await screen.findByText(t.noMatchingEventsNotice)
  })

  it('resolves username via the actor username condition (username query param, wi-147)', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { events: [] }))),
    )
    const onSearch = mock()
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
        nextCursor={null}
        onSearch={onSearch}
      />,
    )

    const fieldTypeSelect = screen.getAllByRole('combobox')[0] as HTMLSelectElement
    fireEvent.change(fieldTypeSelect, { target: { value: 'quick.username' } })
    fireEvent.change(screen.getByPlaceholderText(t.actorUsernameFieldPlaceholder), {
      target: { value: 'alice' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.filterAction }))

    expect(onSearch).toHaveBeenCalledWith(expect.objectContaining({ username: 'alice' }))
    await screen.findByText(t.noMatchingEventsNotice)
  })

  it('navigates to the next addressable page without appending rows', async () => {
    const onPage = mock()
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
        nextCursor="abc"
        search={{ sub: 'usr_current' }}
        onPage={onPage}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: tCommon.nextPage }))

    expect(onPage).toHaveBeenCalledWith('abc')
    expect(await screen.findAllByText(friendlyEventName(event.type, 'en'))).not.toHaveLength(0)
  })
})

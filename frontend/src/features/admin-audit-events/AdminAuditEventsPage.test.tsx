import { useState } from 'react'
import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAuditEventsPage } from './AdminAuditEventsPage'
import { adminAuditEventsDictionary } from './AdminAuditEventsPage.i18n'
import type { AdminAuditEventsSearchParams } from '../../api'
import type { AdminAuditEvent } from '../../types'

const t = adminAuditEventsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
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
        search={routeData.search}
      />
    </>
  )
}

describe('locale', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('renders the audit events page in English by default', async () => {
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[]}
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
      />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: adminAuditEventsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminAuditEventsPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('shows an empty state when a filtered query returns no events', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, { events: [] }))),
    )
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.filterAction }))

    expect(await screen.findByText(t.noMatchingEventsNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectEventPrompt)).toBeInTheDocument()
  })

  it('shows an error when querying audit events fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(500, { message: 'Could not fetch audit events.' }))),
    )
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.filterAction }))

    expect(await screen.findByText('Could not fetch audit events.')).toBeInTheDocument()
    expect(screen.getAllByText(event.type).length).toBeGreaterThan(0)
  })

  it('initializes the search conditions from the search prop (URL query init, wi-147)', async () => {
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
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
    expect(screen.getAllByText(event.type).length).toBeGreaterThan(0)

    fireEvent.click(screen.getByRole('button', { name: 'Simulate browser back' }))

    expect(await screen.findByDisplayValue('usr_previous')).toBeInTheDocument()
    expect(screen.queryByDisplayValue('usr_current')).not.toBeInTheDocument()
    expect(screen.getAllByText(previousEvent.type).length).toBeGreaterThan(0)
    expect(screen.queryByText(event.type)).not.toBeInTheDocument()
  })

  it('calls onSearch with the built query on submit (URL update on search execution, wi-147)', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, { events: [] }))),
    )
    const onSearch = vi.fn()
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
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
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, { events: [] }))),
    )
    const onSearch = vi.fn()
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
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
})

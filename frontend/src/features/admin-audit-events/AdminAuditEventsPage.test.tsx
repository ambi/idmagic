import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAuditEventsPage } from './AdminAuditEventsPage'
import { adminAuditEventsDictionary } from './AdminAuditEventsPage.i18n'
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
})

import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAuditEventsPage } from './AdminAuditEventsPage'
import type { AdminAuditEvent } from '../../types'

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

    fireEvent.click(screen.getByRole('button', { name: '絞り込み' }))

    expect(await screen.findByText('一致するイベントはありません。')).toBeInTheDocument()
    expect(screen.getByText('イベントを選択してください。')).toBeInTheDocument()
  })

  it('shows an error when querying audit events fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(500, { message: '監査イベントを取得できませんでした。' })),
      ),
    )
    await renderWithRouter(
      <AdminAuditEventsPage
        actorUsername="admin"
        actorRoles={[]}
        actorRealm="tenant-1"
        events={[event]}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: '絞り込み' }))

    expect(await screen.findByText('監査イベントを取得できませんでした。')).toBeInTheDocument()
    expect(screen.getAllByText(event.type).length).toBeGreaterThan(0)
  })
})

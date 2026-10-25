import { afterEach, expect, it, mock } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AuditExport, AuditExportClient } from '../../api'
import { AuditExportPanel } from './AuditExportPanel'
import { cleanup } from '@testing-library/react'

afterEach(cleanup)

//spec:covers REQ-AUDIT-011: 作成、履歴更新、取消しと完成後のダウンロードを表示する。
it('creates with the current filters, cancels and offers only completed downloads', async () => {
  const queued: AuditExport = {
    id: 'export-1',
    status: 'queued',
    scope: 'tenant',
    format: 'json',
    requested_by: 'owner',
    created_at: '2026-10-25T00:00:00Z',
    downloadable: false,
  }
  const client: AuditExportClient = {
    start: mock(async () => queued),
    list: mock(async () => ({
      exports: [
        {
          ...queued,
          status: 'succeeded' as const,
          downloadable: true,
          expires_at: '2026-10-26T00:00:00Z',
          events: 1001,
        },
      ],
    })),
    cancel: mock(async () => ({ ...queued, status: 'canceled' as const })),
    downloadURL: (id) => `/exports/${id}/download`,
  }
  await renderWithRouter(
    <AuditExportPanel client={client} csrfToken="csrf" query={() => ({ sub: 'alice' })} />,
  )
  fireEvent.click(screen.getByRole('button', { name: /^Export$/ }))
  await screen.findByText('Queued')
  expect(client.start).toHaveBeenCalledWith('csrf', { sub: 'alice' })
  expect(screen.queryByRole('link', { name: 'Download' })).toBeNull()
  fireEvent.click(screen.getByRole('button', { name: 'Cancel export' }))
  await screen.findByText('Canceled')
  expect(client.cancel).toHaveBeenCalledWith('csrf', 'export-1')
  fireEvent.click(screen.getByRole('button', { name: 'Refresh export history' }))
  await waitFor(() =>
    expect(screen.getByRole('link', { name: 'Download' }).getAttribute('href')).toBe(
      '/exports/export-1/download',
    ),
  )
  expect(screen.queryByRole('button', { name: 'Cancel export' })).toBeNull()
})

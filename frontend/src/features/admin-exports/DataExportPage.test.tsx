import { afterEach, describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { userExports } from '../../api'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { DataExportPage } from './DataExportPage'
import { dataExportDictionary } from './DataExportPage.i18n'

const t = dataExportDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('DataExportPage', () => {
  afterEach(() => restoreGlobals())

  it('offers only allowlisted columns and starts a user export', async () => {
    const queuedJob = {
      id: 'exp-1',
      status: 'queued',
      target: 'user',
      format: 'csv',
      requested_columns: ['preferred_username'],
      requested_by: 'admin',
      created_at: '2026-07-25T00:00:00Z',
      downloadable: false,
    }
    let started = false
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.endsWith('/api/admin/v1/users/exports') && init?.method === 'POST') {
          started = true
          return Promise.resolve(response(202, queuedJob))
        }
        if (url.endsWith('/api/admin/v1/users/exports')) {
          return Promise.resolve(response(200, { exports: started ? [queuedJob] : [] }))
        }
        return Promise.resolve(response(404))
      }),
    )

    await renderWithRouter(<DataExportPage csrfToken="csrf" target="users" api={userExports} />)

    // A sensitive column (password_hash) must never be offered.
    expect(screen.queryByText('password_hash')).toBeNull()
    // The username column is offered and PII columns are badged.
    expect(screen.getByText(t.colUsername)).toBeInTheDocument()
    expect(screen.getAllByText(t.piiBadge).length).toBeGreaterThan(0)

    fireEvent.click(screen.getByRole('button', { name: t.startExport }))

    await waitFor(() => expect(started).toBe(true))
    await waitFor(() => expect(screen.getByText(t.startedNotice)).toBeInTheDocument())
    // The queued export appears with its status.
    await waitFor(() => expect(screen.getByText(t.statusQueued)).toBeInTheDocument())
  })
})

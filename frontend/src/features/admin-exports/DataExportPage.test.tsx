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
        if (url.endsWith('/api/admin/v1/tenant/user_attribute_schema')) {
          return Promise.resolve(
            response(200, {
              tenant_id: 'tenant-1',
              attributes: [
                {
                  key: 'cost_code',
                  label: 'Cost code',
                  type: 'string',
                  multi_valued: false,
                  required: false,
                  editable_by_user: false,
                  visibility: 'private',
                  pii: false,
                },
              ],
              builtin: [
                {
                  key: 'department',
                  label: 'Department',
                  type: 'string',
                  multi_valued: false,
                  required: false,
                  editable_by_user: false,
                  visibility: 'self_readable',
                  pii: false,
                },
              ],
              created_at: '2026-08-10T00:00:00Z',
              updated_at: '2026-08-10T00:00:00Z',
            }),
          )
        }
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
    expect(screen.getByText(t.colRequiredActions)).toBeInTheDocument()
    expect(await screen.findByText('Department')).toBeInTheDocument()
    expect(screen.getByText('(attr:department)')).toBeInTheDocument()
    expect(await screen.findByText('Cost code')).toBeInTheDocument()
    expect(screen.getByText('(custom:cost_code)')).toBeInTheDocument()
    expect(screen.getAllByText(t.piiBadge).length).toBeGreaterThan(0)

    fireEvent.click(screen.getByRole('button', { name: t.startExport }))

    await waitFor(() => expect(started).toBe(true))
    await waitFor(() => expect(screen.getByText(t.startedNotice)).toBeInTheDocument())
    // The queued export appears with its status.
    await waitFor(() => expect(screen.getByText(t.statusQueued)).toBeInTheDocument())
  })
})

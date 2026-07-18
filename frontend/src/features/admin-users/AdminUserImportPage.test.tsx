import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUserImportPage } from './AdminUserImportPage'
import { adminUsersDictionary } from './AdminUsersPage.i18n'

const t = adminUsersDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

describe('AdminUserImportPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  function csvFile(content: string) {
    return new File([content], 'users.csv', { type: 'text/csv' })
  }

  async function selectFile(content: string) {
    fireEvent.change(screen.getByLabelText(t.selectCsvFile), {
      target: { files: [csvFile(content)] },
    })
    await waitFor(() =>
      expect(
        screen.getByText(t.selectedFileLabel.replace('{name}', 'users.csv')),
      ).toBeInTheDocument(),
    )
  }

  it('previews row errors from dry run and applies only after explicit confirmation', async () => {
    const dryRunResult = {
      total_rows: 2,
      accepted_rows: 1,
      rejected_rows: 1,
      errors: [{ row: 3, column: 'email', code: 'invalid_email' }],
    }
    const applyResult = { total_rows: 2, accepted_rows: 1, rejected_rows: 1, errors: [] }
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.endsWith('/api/admin/users/imports') && init?.method === 'POST') {
          const mode = (JSON.parse(String(init.body)) as { mode: string }).mode
          const id = mode === 'apply' ? 'job-apply' : 'job-dry-run'
          return Promise.resolve(response(202, { id, status: 'queued', mode }))
        }
        if (url.endsWith('/api/admin/users/imports/job-dry-run')) {
          return Promise.resolve(
            response(200, { id: 'job-dry-run', status: 'succeeded', result: dryRunResult }),
          )
        }
        if (url.endsWith('/api/admin/users/imports/job-apply')) {
          return Promise.resolve(
            response(200, { id: 'job-apply', status: 'succeeded', result: applyResult }),
          )
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )

    await renderWithRouter(<AdminUserImportPage csrfToken="csrf" />)
    await selectFile('preferred_username,email,name,roles\njiro,not-an-email,Jiro,\n')

    fireEvent.click(screen.getByRole('button', { name: t.runDryRun }))
    expect(await screen.findByText(t.importErrorInvalidEmail)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: t.applyImport }))
    const dialog = await screen.findByRole('dialog')
    expect(fetch).not.toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ body: expect.stringContaining('"apply"') }),
    )

    fireEvent.click(within(dialog).getByRole('button', { name: t.applyImportConfirmButton }))

    expect(
      await screen.findByText(t.importApplySuccessNotice.replace('{count}', '1')),
    ).toBeInTheDocument()
  })

  it('shows a translated message when the CSV is rejected before a job is created', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(400, { error: 'invalid_header' }))),
    )

    await renderWithRouter(<AdminUserImportPage csrfToken="csrf" />)
    await selectFile('preferred_username,email,name,roles,password\njiro,a@b.com,Jiro,,secret\n')

    fireEvent.click(screen.getByRole('button', { name: t.runDryRun }))

    expect(await screen.findByText(t.importErrorInvalidHeader)).toBeInTheDocument()
  })
})

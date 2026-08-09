import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUserImportPage } from './AdminUserImportPage'
import { adminUsersDictionary } from './AdminUsersPage.i18n'

const t = adminUsersDictionary.en

const response = (status: number, body: unknown = {}, headers: Record<string, string> = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  headers: new Headers(headers),
  json: mock().mockResolvedValue(body),
})

const paginationHeaders = (totalItems: number) => ({
  'Pagination-Total-Items': String(totalItems),
  'Pagination-Total-Pages': totalItems === 0 ? '0' : '1',
  'Pagination-Current-Page': totalItems === 0 ? '0' : '1',
  'Pagination-Page-Size': '100',
})

describe('AdminUserImportPage', () => {
  afterEach(() => restoreGlobals())

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
    const previewResult = {
      total_rows: 2,
      created_rows: 1,
      updated_rows: 0,
      unchanged_rows: 0,
      rejected_rows: 1,
      error_total: 1,
    }
    const applyResult = {
      total_rows: 2,
      created_rows: 1,
      updated_rows: 0,
      unchanged_rows: 0,
      rejected_rows: 1,
      error_total: 0,
    }
    let previewBody: BodyInit | null | undefined
    let applyBody: BodyInit | null | undefined
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.endsWith('/api/admin/v1/users/imports') && init?.method === 'POST') {
          previewBody = init.body
          return Promise.resolve(
            response(202, { id: 'job-preview', status: 'queued', mode: 'preview' }),
          )
        }
        if (url.endsWith('/api/admin/v1/users/imports/job-preview/apply')) {
          applyBody = init?.body
          return Promise.resolve(
            response(202, { id: 'job-apply', status: 'queued', mode: 'apply' }),
          )
        }
        if (url.includes('/api/admin/v1/users/imports/job-preview')) {
          return Promise.resolve(
            response(
              200,
              {
                id: 'job-preview',
                status: 'succeeded',
                mode: 'preview',
                result: previewResult,
                errors: [{ row: 3, column: 'email', code: 'invalid_email' }],
              },
              paginationHeaders(1),
            ),
          )
        }
        if (url.includes('/api/admin/v1/users/imports/job-apply')) {
          return Promise.resolve(
            response(
              200,
              {
                id: 'job-apply',
                status: 'succeeded',
                mode: 'apply',
                result: applyResult,
                errors: [],
              },
              paginationHeaders(0),
            ),
          )
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )

    await renderWithRouter(<AdminUserImportPage csrfToken="csrf" />)
    await selectFile('preferred_username,email,name,roles\njiro,not-an-email,Jiro,\n')

    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))
    expect(await screen.findByText(t.importErrorInvalidEmail)).toBeInTheDocument()
    expect(previewBody).toBeInstanceOf(File)

    fireEvent.click(screen.getByRole('button', { name: t.applyImport }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: t.applyImportConfirmButton }))

    expect(
      await screen.findByText(t.importApplySuccessNotice.replace('{count}', '1')),
    ).toBeInTheDocument()
    expect(applyBody).toBeUndefined()
  })

  it('shows a translated message when the CSV is rejected before a job is created', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { error: 'invalid_header' }))),
    )

    await renderWithRouter(<AdminUserImportPage csrfToken="csrf" />)
    await selectFile('preferred_username,email,name,roles,password\njiro,a@b.com,Jiro,,secret\n')

    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))

    expect(await screen.findByText(t.importErrorInvalidHeader)).toBeInTheDocument()
  })
})

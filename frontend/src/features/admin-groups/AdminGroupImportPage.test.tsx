import { afterEach, describe, it, expect, mock } from 'bun:test'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupImportPage } from './AdminGroupImportPage'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

const t = adminGroupsDictionary.en

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

const emptyCounts = {
  total_rows: 0,
  created_rows: 0,
  updated_rows: 0,
  unchanged_rows: 0,
  deleted_rows: 0,
  deleted_memberships: 0,
  rejected_rows: 0,
  error_total: 0,
}

describe('AdminGroupImportPage', () => {
  afterEach(() => restoreGlobals())

  function csvFile(content: string) {
    return new File([content], 'groups.csv', { type: 'text/csv' })
  }

  async function selectFile(content: string) {
    fireEvent.change(screen.getByLabelText(t.selectCsvFile), {
      target: { files: [csvFile(content)] },
    })
    await waitFor(() =>
      expect(
        screen.getByText(t.selectedFileLabel.replace('{name}', 'groups.csv')),
      ).toBeInTheDocument(),
    )
  }

  function stubImport(
    previewBody: unknown,
    applyBody: unknown,
    capture: { preview?: BodyInit | null; apply?: BodyInit | null },
  ) {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.endsWith('/api/admin/v1/groups/imports') && init?.method === 'POST') {
          capture.preview = init.body
          return Promise.resolve(
            response(202, { id: 'job-preview', status: 'queued', mode: 'preview' }),
          )
        }
        if (url.endsWith('/api/admin/v1/groups/imports/job-preview/apply')) {
          capture.apply = init?.body
          return Promise.resolve(
            response(202, { id: 'job-apply', status: 'queued', mode: 'apply' }),
          )
        }
        if (url.includes('/api/admin/v1/groups/imports/job-preview')) {
          return Promise.resolve(response(200, previewBody, paginationHeaders(1)))
        }
        if (url.includes('/api/admin/v1/groups/imports/job-apply')) {
          return Promise.resolve(response(200, applyBody, paginationHeaders(0)))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
  }

  it('previews row errors and applies only after explicit confirmation, without resending the CSV', async () => {
    const capture: { preview?: BodyInit | null; apply?: BodyInit | null } = {}
    stubImport(
      {
        id: 'job-preview',
        status: 'succeeded',
        mode: 'preview',
        result: {
          ...emptyCounts,
          total_rows: 2,
          created_rows: 1,
          rejected_rows: 1,
          error_total: 1,
        },
        errors: [{ row: 3, column: 'membership_type', code: 'immutable_membership_type' }],
      },
      {
        id: 'job-apply',
        status: 'succeeded',
        mode: 'apply',
        result: { ...emptyCounts, total_rows: 2, created_rows: 1, rejected_rows: 1 },
        errors: [],
      },
      capture,
    )

    await renderWithRouter(<AdminGroupImportPage csrfToken="csrf" />)
    await selectFile('name,membership_type\nplatform,dynamic\n')

    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))
    expect(await screen.findByText(t.importErrorImmutableMembershipType)).toBeInTheDocument()
    expect(capture.preview).toBeInstanceOf(File)

    fireEvent.click(screen.getByRole('button', { name: t.applyImport }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: t.applyImportConfirmButton }))

    expect(
      await screen.findByText(t.importApplySuccessNotice.replace('{count}', '1')),
    ).toBeInTheDocument()
    // apply は成功済み preview ID だけを参照する。CSV を再送しない。
    expect(capture.apply).toBeUndefined()
  })

  it('reports deletion counts apart from the other operations and blocks apply until they are acknowledged', async () => {
    const capture: { preview?: BodyInit | null; apply?: BodyInit | null } = {}
    stubImport(
      {
        id: 'job-preview',
        status: 'succeeded',
        mode: 'preview',
        result: {
          ...emptyCounts,
          total_rows: 3,
          updated_rows: 1,
          deleted_rows: 2,
          deleted_memberships: 5,
        },
        errors: [],
      },
      {
        id: 'job-apply',
        status: 'succeeded',
        mode: 'apply',
        result: {
          ...emptyCounts,
          total_rows: 3,
          updated_rows: 1,
          deleted_rows: 2,
          deleted_memberships: 5,
        },
        errors: [],
      },
      capture,
    )

    await renderWithRouter(<AdminGroupImportPage csrfToken="csrf" />)
    await selectFile('id,name,lifecycle_action\ngroup-1,engineering,delete\n')

    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))
    // 削除件数と巻き込まれる membership 件数は、他の操作と分けて独立に表示する。
    const deletionSection = (await screen.findByText(t.importDeletionHeading)).parentElement
    expect(deletionSection).not.toBeNull()
    expect(within(deletionSection as HTMLElement).getByText('2')).toBeInTheDocument()
    expect(within(deletionSection as HTMLElement).getByText('5')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: t.applyImport }))
    const dialog = await screen.findByRole('dialog')
    expect(
      within(dialog).getByText(
        t.applyImportConfirmDeleteWarning.replace('{groups}', '2').replace('{memberships}', '5'),
      ),
    ).toBeInTheDocument()

    // 確認前は適用できない。CSV の 1 列が大量削除を発火しうるため、
    // 明示確認が多層の防御の 1 つになっている。
    const confirm = within(dialog).getByRole('button', { name: t.applyImportConfirmButton })
    expect(confirm).toBeDisabled()
    fireEvent.click(within(dialog).getByRole('button', { name: t.applyImportConfirmButton }))
    expect(capture.apply).toBeUndefined()
    expect(screen.queryByText(t.applyResultTitle)).not.toBeInTheDocument()

    fireEvent.click(within(dialog).getByLabelText(t.applyImportConfirmDeleteAcknowledge))
    fireEvent.click(within(dialog).getByRole('button', { name: t.applyImportConfirmButton }))

    expect(
      await screen.findByText(
        t.importApplyDeletedNotice.replace('{groups}', '2').replace('{memberships}', '5'),
      ),
    ).toBeInTheDocument()
  })

  it('shows a translated message when the CSV is rejected before a job is created', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { error: 'invalid_header' }))),
    )

    await renderWithRouter(<AdminGroupImportPage csrfToken="csrf" />)
    await selectFile('name,password\nplatform,secret\n')

    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))

    expect(await screen.findByText(t.importErrorInvalidHeader)).toBeInTheDocument()
  })
})

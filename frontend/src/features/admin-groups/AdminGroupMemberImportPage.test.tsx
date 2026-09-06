import { afterEach, describe, it, expect, mock } from 'bun:test'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupMemberImportPage } from './AdminGroupMemberImportPage'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

const t = adminGroupsDictionary.en
const GROUP_ID = 'group-engineering'
const BASE = `/api/admin/v1/groups/${GROUP_ID}/members/imports`

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
  group_id: GROUP_ID,
  total_rows: 0,
  added_rows: 0,
  removed_rows: 0,
  unchanged_rows: 0,
  rejected_rows: 0,
  error_total: 0,
}

describe('AdminGroupMemberImportPage', () => {
  afterEach(() => restoreGlobals())

  function csvFile(content: string) {
    return new File([content], 'members.csv', { type: 'text/csv' })
  }

  async function selectFile(content: string) {
    fireEvent.change(screen.getByLabelText(t.selectCsvFile), {
      target: { files: [csvFile(content)] },
    })
    await waitFor(() =>
      expect(
        screen.getByText(t.selectedFileLabel.replace('{name}', 'members.csv')),
      ).toBeInTheDocument(),
    )
  }

  function stubImport(
    previewBody: unknown,
    applyBody: unknown,
    capture: { preview?: BodyInit | null; apply?: BodyInit | null; urls: string[] },
  ) {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        capture.urls.push(url)
        if (url.endsWith(BASE) && init?.method === 'POST') {
          capture.preview = init.body
          return Promise.resolve(
            response(202, { id: 'job-preview', status: 'queued', mode: 'preview' }),
          )
        }
        if (url.endsWith(`${BASE}/job-preview/apply`)) {
          capture.apply = init?.body
          return Promise.resolve(
            response(202, { id: 'job-apply', status: 'queued', mode: 'apply' }),
          )
        }
        if (url.includes(`${BASE}/job-preview`)) {
          return Promise.resolve(response(200, previewBody, paginationHeaders(1)))
        }
        if (url.includes(`${BASE}/job-apply`)) {
          return Promise.resolve(response(200, applyBody, paginationHeaders(0)))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
  }

  it('previews row errors and applies only after explicit confirmation, without resending the CSV', async () => {
    const capture: { preview?: BodyInit | null; apply?: BodyInit | null; urls: string[] } = {
      urls: [],
    }
    stubImport(
      {
        id: 'job-preview',
        status: 'succeeded',
        mode: 'preview',
        result: { ...emptyCounts, total_rows: 2, added_rows: 1, rejected_rows: 1, error_total: 1 },
        errors: [{ row: 3, column: 'membership_state', code: 'invalid_membership_state' }],
      },
      {
        id: 'job-apply',
        status: 'succeeded',
        mode: 'apply',
        result: { ...emptyCounts, total_rows: 2, added_rows: 1, rejected_rows: 1 },
        errors: [],
      },
      capture,
    )

    await renderWithRouter(<AdminGroupMemberImportPage csrfToken="csrf" groupId={GROUP_ID} />)
    await selectFile('user_id,membership_state\nuser-bob,present\nuser-x,maybe\n')

    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))
    expect(
      await screen.findByText(t.membershipImportErrorInvalidMembershipState),
    ).toBeInTheDocument()
    expect(capture.preview).toBeInstanceOf(File)

    fireEvent.click(screen.getByRole('button', { name: t.applyImport }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: t.applyImportConfirmButton }))

    expect(
      await screen.findByText(
        t.membershipImportApplySuccessNotice.replace('{added}', '1').replace('{removed}', '0'),
      ),
    ).toBeInTheDocument()
    // apply は成功済み preview ID だけを参照する。CSV を再送しない。
    expect(capture.apply).toBeUndefined()
    // すべての要求がパスのグループの下にある。別グループへ漏れる経路を持たない。
    for (const url of capture.urls) {
      expect(url).toContain(`/groups/${GROUP_ID}/members/imports`)
    }
  })

  it('reports the release count apart from the other operations and blocks apply until it is acknowledged', async () => {
    const capture: { preview?: BodyInit | null; apply?: BodyInit | null; urls: string[] } = {
      urls: [],
    }
    const counts = {
      ...emptyCounts,
      total_rows: 4,
      added_rows: 1,
      unchanged_rows: 1,
      removed_rows: 2,
    }
    stubImport(
      { id: 'job-preview', status: 'succeeded', mode: 'preview', result: counts, errors: [] },
      { id: 'job-apply', status: 'succeeded', mode: 'apply', result: counts, errors: [] },
      capture,
    )

    await renderWithRouter(<AdminGroupMemberImportPage csrfToken="csrf" groupId={GROUP_ID} />)
    await selectFile('user_id,membership_state\nuser-alice,absent\n')

    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))
    // 解除件数は他の操作と分けて独立に表示する。
    const releaseSection = (await screen.findByText(t.membershipImportReleaseHeading)).parentElement
    expect(releaseSection).not.toBeNull()
    expect(within(releaseSection as HTMLElement).getByText('2')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: t.applyImport }))
    const dialog = await screen.findByRole('dialog')
    expect(
      within(dialog).getByText(t.membershipImportConfirmReleaseWarning.replace('{removed}', '2')),
    ).toBeInTheDocument()

    // 確認前は適用できない。CSV の 1 列が大量の権限剥奪を発火しうるため、
    // 明示確認が多層の防御の 1 つになっている。
    const confirm = within(dialog).getByRole('button', { name: t.applyImportConfirmButton })
    expect(confirm).toBeDisabled()
    fireEvent.click(confirm)
    expect(capture.apply).toBeUndefined()
    expect(screen.queryByText(t.applyResultTitle)).not.toBeInTheDocument()

    fireEvent.click(within(dialog).getByLabelText(t.membershipImportConfirmReleaseAcknowledge))
    fireEvent.click(within(dialog).getByRole('button', { name: t.applyImportConfirmButton }))

    expect(
      await screen.findByText(
        t.membershipImportApplySuccessNotice.replace('{added}', '1').replace('{removed}', '2'),
      ),
    ).toBeInTheDocument()
  })

  it('translates a whole-file refusal the preview job reports', async () => {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.endsWith(BASE) && init?.method === 'POST') {
          return Promise.resolve(
            response(202, { id: 'job-preview', status: 'queued', mode: 'preview' }),
          )
        }
        return Promise.resolve(
          response(
            200,
            {
              id: 'job-preview',
              status: 'succeeded',
              mode: 'preview',
              result: { ...emptyCounts, rejected_rows: 1, error_total: 1 },
              errors: [{ row: 0, column: 'group_id', code: 'dynamic_group' }],
            },
            paginationHeaders(1),
          ),
        )
      }),
    )

    await renderWithRouter(<AdminGroupMemberImportPage csrfToken="csrf" groupId={GROUP_ID} />)
    await selectFile('user_id,membership_state\nuser-bob,present\n')
    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))

    expect(await screen.findByText(t.membershipImportErrorDynamicGroup)).toBeInTheDocument()
  })

  it('shows a translated message when the CSV is rejected before a job is created', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { error: 'csv_too_large' }))),
    )

    await renderWithRouter(<AdminGroupMemberImportPage csrfToken="csrf" groupId={GROUP_ID} />)
    await selectFile('user_id,membership_state\nuser-bob,present\n')
    fireEvent.click(screen.getByRole('button', { name: t.runPreview }))

    expect(await screen.findByText(t.importErrorCsvTooLarge)).toBeInTheDocument()
  })
})

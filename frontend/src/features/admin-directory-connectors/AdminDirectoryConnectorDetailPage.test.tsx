import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { DirectoryConnector, IngestionRunSummary } from '../../types'
import { AdminDirectoryConnectorDetailPage } from './AdminDirectoryConnectorDetailPage'
import { adminDirectoryConnectorsDictionary } from './AdminDirectoryConnectorsPage.i18n'

const t = adminDirectoryConnectorsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
  headers: new Headers(),
})

const connector = (overrides: Partial<DirectoryConnector> = {}): DirectoryConnector => ({
  id: 'connector-1',
  display_name: 'Corp AD',
  status: 'active',
  created_at: '2026-10-31T00:00:00Z',
  key_id: 'kid-1',
  accidental_deletion_count_threshold: 500,
  accidental_deletion_percent_threshold: 20,
  ...overrides,
})

const counts = {
  batches: 1,
  items: 5,
  created: 1,
  updated: 0,
  unchanged: 0,
  deleted: 0,
  failed: 0,
  disabled: 0,
}

const failedRun: IngestionRunSummary = {
  id: 'run-1',
  mode: 'full',
  status: 'completed',
  started_at: '2026-10-31T00:00:00Z',
  counts: { ...counts, failed: 101 },
}

async function renderPage(
  overrides: Partial<DirectoryConnector> = {},
  runs: IngestionRunSummary[] = [],
) {
  return renderWithRouter(
    <AdminDirectoryConnectorDetailPage
      csrfToken="csrf"
      connector={connector(overrides)}
      runs={runs}
    />,
  )
}

afterEach(() => {
  restoreGlobals()
})

describe('AdminDirectoryConnectorDetailPage', () => {
  //spec:covers REQ-SOURCING-023: Active の再発行は確認の前に要求せず、確認の後にトークンを表示すること。
  it('Active の再発行は確認の後に要求し、新しいトークンを表示する', async () => {
    const fetchMock = mock().mockResolvedValue(
      response(201, {
        connector: connector({ status: 'awaiting_enrollment', key_id: undefined }),
        enrollment_token: 'idmagic_dce_new',
      }),
    )
    stubGlobal('fetch', fetchMock)
    await renderPage()

    fireEvent.click(screen.getByRole('button', { name: t.reissue }))
    expect(screen.getByText(t.reissueConfirmMessage)).toBeTruthy()
    expect(fetchMock).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: t.reissueConfirm }))
    expect(await screen.findByText('idmagic_dce_new')).toBeTruthy()
    expect(String(fetchMock.mock.calls[0][0])).toContain(
      '/api/admin/v1/directory-connectors/connector-1/enrollment-tokens',
    )
  })

  //spec:covers REQ-SOURCING-023: 拒否された再発行はエラーを表示し、トークンを表示しないこと。
  it('再発行が拒否されたらエラーを表示してトークンを表示しない', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(500)))
    await renderPage({ status: 'awaiting_enrollment' })

    fireEvent.click(screen.getByRole('button', { name: t.reissue }))
    expect(await screen.findByRole('alert')).toBeTruthy()
    expect(screen.queryByText(t.tokenNotice)).toBeNull()
  })

  //spec:covers REQ-SOURCING-023: 拒否された無効化は、表示中のトークンを残さないこと。
  it('無効化が拒否されたら表示中のトークンを消す', async () => {
    stubGlobal(
      'fetch',
      mock()
        .mockResolvedValueOnce(
          response(201, {
            connector: connector({ status: 'awaiting_enrollment' }),
            enrollment_token: 'idmagic_dce_shown',
          }),
        )
        .mockResolvedValueOnce(response(500)),
    )
    await renderPage({ status: 'awaiting_enrollment' })

    fireEvent.click(screen.getByRole('button', { name: t.reissue }))
    expect(await screen.findByText('idmagic_dce_shown')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: t.disable }))
    fireEvent.click(screen.getByRole('button', { name: t.disableConfirm }))
    // 要素を expect に渡すと、失敗のたびに DOM 全体を整形して 1 回で 1 秒近くかかる。
    await waitFor(() => expect(screen.queryByText(t.disableConfirmMessage) === null).toBe(true))
    expect(screen.queryByText('idmagic_dce_shown')).toBeNull()
    expect(screen.getByRole('alert')).toBeTruthy()
  })

  //spec:covers REQ-SOURCING-023: 無効化は確認を求め、Disabled になると再発行と無効化の操作を消すこと。
  it('無効化は確認の後に要求し、操作を消す', async () => {
    const fetchMock = mock().mockResolvedValue(
      response(200, connector({ status: 'disabled', disabled_at: '2026-10-31T02:00:00Z' })),
    )
    stubGlobal('fetch', fetchMock)
    await renderPage()

    fireEvent.click(screen.getByRole('button', { name: t.disable }))
    expect(fetchMock).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: t.disableConfirm }))

    expect(await screen.findByText(t.disabledAtLabel)).toBeTruthy()
    expect(screen.queryByRole('button', { name: t.reissue })).toBeNull()
    expect(screen.queryByRole('button', { name: t.disable })).toBeNull()
  })

  //spec:covers REQ-SOURCING-024: 期限切れの登録待ちに期限切れと再発行の案内を表示すること。
  it('期限切れの登録トークンを知らせる', async () => {
    await renderPage({
      status: 'awaiting_enrollment',
      enrollment_expires_at: '2020-01-01T00:00:00Z',
    })
    expect(screen.getByText(t.enrollmentExpiredNotice)).toBeTruthy()
  })

  //spec:covers REQ-SOURCING-024: 失敗の項目の外部の不変 ID、説明、バッチの ID と、表示していない数を表示すること。
  it('失敗の一覧を開くと項目と表示しきれない件数を示す', async () => {
    const failures = Array.from({ length: 100 }, (_, i) => ({
      batch_id: `batch-${i}`,
      external_id: `guid-${i}`,
      error: 'username_conflict',
    }))
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { ...failedRun, failures })))
    await renderPage({}, [failedRun])

    fireEvent.click(screen.getByRole('button', { name: t.showFailures }))
    expect(await screen.findByText('guid-0')).toBeTruthy()
    expect(screen.getByText('batch-99')).toBeTruthy()
    expect(screen.getAllByText(t.itemErrorUsernameConflict)).toHaveLength(100)
    expect(screen.getByText(t.failuresHidden.replace('{count}', '1'))).toBeTruthy()
  })

  //spec:covers REQ-SOURCING-024: 実行の取得の失敗を、失敗 0 件のように表示しないこと。
  it('実行を読めないときはエラーを表示する', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(500)))
    await renderPage({}, [failedRun])

    fireEvent.click(screen.getByRole('button', { name: t.showFailures }))
    await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy())
    expect(screen.queryByText(t.headerExternalId)).toBeNull()
  })

  //spec:covers REQ-SOURCING-024: 保留を記録した実行に候補件数と総数を表示すること。
  it('保留の候補件数と総数を表示する', async () => {
    await renderPage({}, [
      {
        ...failedRun,
        status: 'awaiting_approval',
        counts,
        protection: {
          candidate_count: 600,
          active_count: 1000,
          accidental_deletion_count_threshold: 500,
          accidental_deletion_percent_threshold: 20,
          held_at: '2026-10-31T00:00:00Z',
        },
      },
    ])
    expect(screen.getByText(t.runStatusAwaitingApproval)).toBeTruthy()
    expect(
      screen.getByText(t.holdSummary.replace('{candidates}', '600').replace('{active}', '1,000')),
    ).toBeTruthy()
    expect(screen.queryByRole('button', { name: t.showFailures })).toBeNull()
  })
})

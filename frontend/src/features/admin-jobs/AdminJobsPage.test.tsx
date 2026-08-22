import { afterEach, describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen } from '@testing-library/react'
import type { AdminJob } from '../../api'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminJobsPage } from './AdminJobsPage'
import { adminJobsDictionary } from './AdminJobsPage.i18n'

const t = adminJobsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
  headers: new Headers(),
})

function job(overrides: Partial<AdminJob> = {}): AdminJob {
  return {
    id: 'job-1',
    tenant_id: 'acme',
    kind: 'user_import_apply',
    lane: 'bulk',
    status: 'queued',
    attempts: 0,
    max_attempts: 3,
    run_at: '2026-03-01T00:00:00Z',
    created_at: '2026-03-01T00:00:00Z',
    updated_at: '2026-03-01T00:00:00Z',
    ...overrides,
  }
}

function renderPage(jobs: AdminJob[], roles: string[] = ['admin'], realm = 'acme') {
  return renderWithRouter(
    <AdminJobsPage
      actorUsername="admin"
      actorRoles={roles}
      actorRealm={realm}
      jobs={jobs}
      kinds={[...new Set(jobs.map((j) => j.kind))]}
      csrfToken="csrf"
    />,
  )
}

describe('AdminJobsPage', () => {
  afterEach(() => restoreGlobals())

  it('lists jobs with status, lane, and attempts', async () => {
    await renderPage([
      job(),
      job({ id: 'job-2', status: 'failed', kind: 'noop_echo', attempts: 3 }),
    ])

    // 状態と レーンの語は絞り込みの選択肢にも現れるので、表の中だけを見る。
    const rows = screen.getAllByRole('row').slice(1)
    expect(rows).toHaveLength(2)
    expect(rows[0]).toHaveTextContent('user_import_apply')
    expect(rows[0]).toHaveTextContent(t.statusQueued)
    expect(rows[0]).toHaveTextContent(t.laneBulk)
    expect(rows[0]).toHaveTextContent('0/3')
    expect(rows[1]).toHaveTextContent(t.statusFailed)
    expect(rows[1]).toHaveTextContent('3/3')
  })

  it('shows an empty state when nothing matches', async () => {
    await renderPage([])
    expect(screen.getByText(t.noMatchingJobsNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectJobPrompt)).toBeInTheDocument()
  })

  // REQ-JOBS-014: ハンドラーの入出力は表示せず、その旨を運用者へ明示する。
  it('states that the handler payload is deliberately not shown', async () => {
    await renderPage([job()])
    expect(screen.getByText(t.payloadOmittedNotice)).toBeInTheDocument()
  })

  it('shows the failure reason of a dead-lettered job', async () => {
    await renderPage([job({ status: 'failed', attempts: 3, error: 'destination unreachable' })])
    expect(screen.getByText('destination unreachable')).toBeInTheDocument()
  })

  // REQ-JOBS-013: 終端に達していないジョブには取り消しを出す。
  it('offers cancel while the job is still running', async () => {
    await renderPage([job({ status: 'running' })])
    expect(screen.getByRole('button', { name: t.cancelAction })).toBeInTheDocument()
  })

  // REQ-JOBS-013: 終端に達したジョブには取り消しを出さない。
  it('does not offer cancel once the job has finished', async () => {
    await renderPage([job({ id: 'job-done', status: 'succeeded' })])
    expect(screen.queryByRole('button', { name: t.cancelAction })).not.toBeInTheDocument()
  })

  it('cancels a job and reflects the new status', async () => {
    stubGlobal(
      'confirm',
      mock(() => true),
    )
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, job({ status: 'canceled' })))),
    )
    await renderPage([job()])

    fireEvent.click(screen.getByRole('button', { name: t.cancelAction }))

    expect(await screen.findByText(t.cancelSucceededNotice)).toBeInTheDocument()
    expect(screen.getAllByText(t.statusCanceled).length).toBeGreaterThan(0)
  })

  it('does not cancel when the operator declines the confirmation', async () => {
    stubGlobal(
      'confirm',
      mock(() => false),
    )
    const fetchMock = mock(() => Promise.resolve(response(200, job())))
    stubGlobal('fetch', fetchMock)
    await renderPage([job()])

    fireEvent.click(screen.getByRole('button', { name: t.cancelAction }))

    expect(fetchMock).not.toHaveBeenCalled()
  })

  // 全テナント横断は system_admin かつ制御面テナントの経路でのみ提示する。
  it('hides the cross-tenant toggle from a tenant administrator', async () => {
    await renderPage([job()])
    expect(screen.queryByLabelText(t.crossTenantLabel)).not.toBeInTheDocument()
  })

  it('offers the cross-tenant toggle to a system_admin on the control plane', async () => {
    await renderPage([job()], ['admin', 'system_admin'], 'default')
    expect(screen.getByLabelText(t.crossTenantLabel)).toBeInTheDocument()
  })
})

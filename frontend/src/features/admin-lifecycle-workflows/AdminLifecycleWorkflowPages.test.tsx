import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminLifecycleWorkflow } from '../../types'
import { AdminLifecycleWorkflowCreatePage } from './AdminLifecycleWorkflowEditorPage'
import { AdminLifecycleWorkflowsPage } from './AdminLifecycleWorkflowsPage'
import { adminLifecycleWorkflowsDictionary } from './AdminLifecycleWorkflowsPage.i18n'

const wf = adminLifecycleWorkflowsDictionary.ja

const workflow: AdminLifecycleWorkflow = {
  id: 'workflow-1',
  name: '入社処理',
  status: 'draft',
  current_revision: 1,
  trigger: { kind: 'user_created' },
  actions: [{ kind: 'send_email', template_key: 'welcome' }],
  created_at: '2026-07-16T00:00:00Z',
  updated_at: '2026-07-16T00:00:00Z',
}

describe('lifecycle workflow page separation', () => {
  afterEach(() => vi.restoreAllMocks())

  it('一覧画面には作成・編集フォームを置かず、専用画面へのリンクを表示する', async () => {
    await renderWithRouter(
      <AdminLifecycleWorkflowsPage csrfToken="csrf" actorUsername="admin" workflows={[workflow]} />,
      { locale: 'ja' },
    )

    expect(screen.queryByLabelText('名前')).not.toBeInTheDocument()
    expect(screen.queryByText('トリガー（いつ実行するか）')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: wf.addWorkflow })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows/new',
    )
    expect(screen.getByRole('link', { name: '編集' })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows/workflow-1/edit',
    )
  })

  it('状態に関係なくワークフローを削除し、一覧から取り除く', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(new Response(null, { status: 204 }))
    await renderWithRouter(
      <AdminLifecycleWorkflowsPage csrfToken="csrf" actorUsername="admin" workflows={[workflow]} />,
      { locale: 'ja' },
    )

    fireEvent.click(screen.getByRole('button', { name: '削除' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/admin/lifecycle_workflows/workflow-1',
        expect.objectContaining({ method: 'DELETE' }),
      )
      expect(screen.queryByText('入社処理')).not.toBeInTheDocument()
    })
  })

  it('専用作成画面にフォームと一覧へ戻る導線を表示する', async () => {
    await renderWithRouter(
      <AdminLifecycleWorkflowCreatePage
        csrfToken="csrf"
        actorUsername="admin"
        groups={[]}
        applications={[]}
      />,
      { locale: 'ja' },
    )

    expect(screen.getByRole('heading', { name: 'ワークフローを作成' })).toBeInTheDocument()
    expect(screen.getByLabelText('名前')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'ワークフロー一覧へ戻る' })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows',
    )
  })
})

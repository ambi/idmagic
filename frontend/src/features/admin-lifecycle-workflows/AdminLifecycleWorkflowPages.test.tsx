import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, spyOn, jest } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminLifecycleWorkflow } from '../../types'
import { AdminLifecycleWorkflowCreatePage } from './AdminLifecycleWorkflowEditorPage'
import { AdminLifecycleWorkflowsPage } from './AdminLifecycleWorkflowsPage'
import { adminLifecycleWorkflowsDictionary } from './AdminLifecycleWorkflowsPage.i18n'
import { workflowFormDictionary } from './WorkflowDefinitionForm.i18n'

const wf = adminLifecycleWorkflowsDictionary.ja
const form = workflowFormDictionary.ja

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
  afterEach(() => jest.restoreAllMocks())

  it('一覧画面には作成・編集フォームを置かず、専用画面へのリンクを表示する', async () => {
    await renderWithRouter(
      <AdminLifecycleWorkflowsPage csrfToken="csrf" actorUsername="admin" workflows={[workflow]} />,
      { locale: 'ja' },
    )

    expect(screen.queryByLabelText('名前')).not.toBeInTheDocument()
    expect(screen.queryByText('トリガー（いつ実行するか）')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: wf.addWorkflow })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows/new',
    )
    expect(screen.getByRole('button', { name: '編集' })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows/workflow-1/edit',
    )
  })

  it('状態に関係なくワークフローを削除し、一覧から取り除く', async () => {
    const fetchMock = spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(null, { status: 204 }),
    )
    await renderWithRouter(
      <AdminLifecycleWorkflowsPage csrfToken="csrf" actorUsername="admin" workflows={[workflow]} />,
      { locale: 'ja' },
    )

    fireEvent.click(screen.getByRole('button', { name: '削除' }))

    expect(fetchMock).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog', { name: 'ワークフローの削除' })).toHaveTextContent('入社処理')

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '削除する' }))
      await new Promise((resolve) => setTimeout(resolve, 0))
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/admin/lifecycle_workflows/workflow-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
    expect(screen.queryByText('入社処理')).not.toBeInTheDocument()
  })

  it('実行前確認の対象ユーザーを説明付きアプリ内ダイアログで入力する', async () => {
    const promptSpy = spyOn(window, 'prompt')
    const fetchMock = spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(Response.json({ runs: [] }))
      .mockResolvedValueOnce(
        Response.json({
          steps: [{ action_kind: 'send_email', would_change: 'would_change' }],
        }),
      )
    await renderWithRouter(
      <AdminLifecycleWorkflowsPage csrfToken="csrf" actorUsername="admin" workflows={[workflow]} />,
      { locale: 'ja' },
    )

    fireEvent.click(screen.getByRole('button', { name: wf.history }))
    await screen.findByRole('heading', {
      name: wf.executionHeading.replace('{name}', workflow.name),
    })
    fireEvent.click(screen.getByRole('button', { name: wf.dryRunButton }))

    const dialog = screen.getByRole('dialog', { name: '実行前確認' })
    expect(dialog).toHaveTextContent(
      'このワークフローを適用した場合の処理内容を確認するユーザーを指定します。',
    )
    fireEvent.change(screen.getByLabelText('対象ユーザー ID'), {
      target: { value: 'user-123' },
    })
    fireEvent.click(screen.getByRole('button', { name: '確認する' }))

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/admin/lifecycle_workflows/workflow-1/dry_run',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ target_user_id: 'user-123' }),
        }),
      ),
    )
    expect(promptSpy).not.toHaveBeenCalled()
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

    expect(screen.getByRole('heading', { name: form.createTitle })).toBeInTheDocument()
    expect(screen.getByLabelText(form.nameLabel)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: form.backToList })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows',
    )
  })
})

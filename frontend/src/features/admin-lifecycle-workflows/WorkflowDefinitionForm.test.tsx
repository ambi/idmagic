import { screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminLifecycleWorkflow } from '../../types'
import {
  WorkflowDefinitionForm,
  validateWorkflowDraft,
  workflowActionLabel,
  workflowInput,
  workflowStatusLabel,
  workflowTriggerLabel,
} from './WorkflowDefinitionForm'

describe('WorkflowDefinitionForm', () => {
  it('機械向けの値ではなく日本語の意味を表示する', async () => {
    await renderWithRouter(
      <WorkflowDefinitionForm groups={[]} applications={[]} busy={false} onSubmit={vi.fn()} />,
      { locale: 'ja' },
    )

    expect(screen.getByText('トリガー（いつ実行するか）')).toBeInTheDocument()
    expect(screen.getByText('アクション（何を行うか）')).toBeInTheDocument()
    expect(screen.getByText('ユーザーが作成されたとき')).toBeInTheDocument()
    expect(screen.getByText('アクションの種類を選択')).toBeInTheDocument()
    expect(screen.queryByText('user_created')).not.toBeInTheDocument()
    expect(screen.queryByText('send_email')).not.toBeInTheDocument()
    expect(screen.queryByText('current_revision')).not.toBeInTheDocument()
  })

  it('必須設定が不足していると日本語で案内する', async () => {
    const onSubmit = vi.fn()
    await renderWithRouter(
      <WorkflowDefinitionForm groups={[]} applications={[]} busy={false} onSubmit={onSubmit} />,
      { locale: 'ja' },
    )

    expect(screen.getByRole('button', { name: '下書きを作成' })).toBeDisabled()
    expect(screen.getByRole('status')).toHaveTextContent('ワークフロー名を入力してください。')
    expect(screen.getByRole('status')).toHaveTextContent('アクション 1の種類を選択してください。')
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('既存定義を日本語の表示名で復元する', async () => {
    const workflow: AdminLifecycleWorkflow = {
      id: 'workflow-1',
      name: '退職処理',
      status: 'draft',
      current_revision: 2,
      trigger: {
        kind: 'user_status_changed',
        from_status: 'active',
        to_status: 'disabled',
      },
      actions: [{ kind: 'disable_user', reason: '退職' }],
      created_at: '2026-07-16T00:00:00Z',
      updated_at: '2026-07-16T00:00:00Z',
    }

    await renderWithRouter(
      <WorkflowDefinitionForm
        workflow={workflow}
        groups={[]}
        applications={[]}
        busy={false}
        onSubmit={vi.fn()}
      />,
      { locale: 'ja' },
    )

    expect(screen.getByDisplayValue('退職処理')).toBeInTheDocument()
    expect(screen.getByText('ユーザー状態が変更されたとき')).toBeInTheDocument()
    expect(screen.getAllByText('ユーザーを無効化').length).toBeGreaterThan(0)
    expect(screen.getByDisplayValue('退職')).toBeInTheDocument()
  })
})

describe('workflow definition mapping', () => {
  it('選択したトリガーとアクション順をAPI入力へ変換する', () => {
    const draft: Parameters<typeof workflowInput>[0] = {
      name: ' 入社処理 ',
      description: ' 初期設定 ',
      triggerKind: 'user_attributes_changed',
      watchedAttributes: 'department, job_title',
      fromStatus: '',
      toStatus: '',
      actions: [
        { key: 'action-1', kind: 'add_group_member', group_id: 'group-1' },
        { key: 'action-2', kind: 'send_email', template_key: 'welcome' },
      ],
    }

    expect(validateWorkflowDraft(draft)).toEqual([])
    expect(workflowInput(draft)).toEqual({
      name: '入社処理',
      description: '初期設定',
      trigger: {
        kind: 'user_attributes_changed',
        watched_attributes: ['department', 'job_title'],
      },
      actions: [
        { kind: 'add_group_member', group_id: 'group-1' },
        { kind: 'send_email', template_key: 'welcome' },
      ],
    })
  })

  it('状態・トリガー・アクションを利用者向け表示へ変換する', () => {
    expect(workflowStatusLabel('draft')).toBe('下書き')
    expect(workflowTriggerLabel('user_created')).toBe('ユーザーが作成されたとき')
    expect(workflowActionLabel('send_email')).toBe('メールを送信')
  })
})

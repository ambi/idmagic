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
import { workflowFormDictionary } from './WorkflowDefinitionForm.i18n'

const ja = workflowFormDictionary.ja
const en = workflowFormDictionary.en

describe('WorkflowDefinitionForm', () => {
  it('機械向けの値ではなく利用者向けの表示名を出す', async () => {
    await renderWithRouter(
      <WorkflowDefinitionForm groups={[]} applications={[]} busy={false} onSubmit={vi.fn()} />,
      { locale: 'ja' },
    )

    expect(screen.getByText(ja.triggerSectionLabel)).toBeInTheDocument()
    expect(screen.getByText(ja.actionsSectionLabel)).toBeInTheDocument()
    expect(screen.getByText(ja.trigger_user_created)).toBeInTheDocument()
    expect(screen.getByText(ja.actionKindPlaceholder)).toBeInTheDocument()
    expect(screen.queryByText('user_created')).not.toBeInTheDocument()
    expect(screen.queryByText('send_email')).not.toBeInTheDocument()
    expect(screen.queryByText('current_revision')).not.toBeInTheDocument()
  })

  it('英語ロケールでは英語で表示する', async () => {
    await renderWithRouter(
      <WorkflowDefinitionForm groups={[]} applications={[]} busy={false} onSubmit={vi.fn()} />,
      { locale: 'en' },
    )

    expect(screen.getByText(en.triggerSectionLabel)).toBeInTheDocument()
    expect(screen.getByText(en.trigger_user_created)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: en.createDraft })).toBeInTheDocument()
  })

  it('必須設定が不足しているとロケールに沿った文言で案内する', async () => {
    const onSubmit = vi.fn()
    await renderWithRouter(
      <WorkflowDefinitionForm groups={[]} applications={[]} busy={false} onSubmit={onSubmit} />,
      { locale: 'ja' },
    )

    expect(screen.getByRole('button', { name: ja.createDraft })).toBeDisabled()
    expect(screen.getByRole('status')).toHaveTextContent(ja.errNameRequired)
    expect(screen.getByRole('status')).toHaveTextContent(ja.errActionKind.replace('{index}', '1'))
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('既存定義を表示名で復元する', async () => {
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
    expect(screen.getByText(ja.trigger_user_status_changed)).toBeInTheDocument()
    expect(screen.getAllByText(ja.action_disable_user).length).toBeGreaterThan(0)
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

    expect(validateWorkflowDraft(draft, ja)).toEqual([])
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

  it('状態・トリガー・アクションを辞書の表示名へ変換する', () => {
    expect(workflowStatusLabel('draft', ja)).toBe(ja.status_draft)
    expect(workflowTriggerLabel('user_created', ja)).toBe(ja.trigger_user_created)
    expect(workflowActionLabel('send_email', en)).toBe(en.action_send_email)
  })
})

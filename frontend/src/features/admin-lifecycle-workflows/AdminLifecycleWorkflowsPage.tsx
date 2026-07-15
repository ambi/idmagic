import { useState } from 'react'
import {
  deleteLifecycleWorkflow,
  dryRunLifecycleWorkflow,
  listLifecycleWorkflowRuns,
  retryLifecycleWorkflowRun,
  setLifecycleWorkflowState,
} from '../../api/admin'
import { tenantURL } from '../../api/core'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import type { AdminLifecycleWorkflow, WorkflowRun } from '../../types'
import {
  workflowActionLabel,
  workflowStatusLabel,
  workflowTriggerLabel,
} from './WorkflowDefinitionForm'

function runStatusLabel(status: string): string {
  return (
    {
      queued: '実行待ち',
      running: '実行中',
      succeeded: '成功',
      partially_failed: '一部失敗',
      failed: '失敗',
      canceled: '中止',
    }[status] ?? '不明'
  )
}

function stepOutcomeLabel(outcome: string): string {
  return (
    {
      pending: '未実行',
      changed: '変更あり',
      no_op: '変更なし',
      failed: '失敗',
      canceled: '中止',
    }[outcome] ?? '不明'
  )
}

function dryRunOutcomeLabel(outcome: string): string {
  return (
    {
      would_change: '変更予定',
      no_op: '変更なし',
      blocked: '実行不可',
    }[outcome] ?? '判定不明'
  )
}

export function AdminLifecycleWorkflowsPage({
  csrfToken,
  actorUsername,
  workflows: initial,
}: {
  csrfToken: string
  actorUsername?: string
  workflows: AdminLifecycleWorkflow[]
}) {
  const [workflows, setWorkflows] = useState(initial)
  const [error, setError] = useState('')
  const [runs, setRuns] = useState<WorkflowRun[]>([])
  const [selected, setSelected] = useState<AdminLifecycleWorkflow | null>(null)
  const [dryRun, setDryRun] = useState<string[]>([])
  async function toggle(workflow: AdminLifecycleWorkflow) {
    try {
      const state = workflow.status === 'enabled' ? 'disable' : 'enable'
      const next = await setLifecycleWorkflowState(
        csrfToken,
        workflow.id,
        state,
        workflow.current_revision,
      )
      setWorkflows(workflows.map((item) => (item.id === next.id ? next : item)))
      if (selected?.id === next.id) setSelected(next)
      setError('')
    } catch {
      setError('状態を変更できませんでした。')
    }
  }
  async function selectWorkflow(workflow: AdminLifecycleWorkflow) {
    try {
      setSelected(workflow)
      setRuns(await listLifecycleWorkflowRuns(workflow.id))
      setDryRun([])
    } catch {
      setError('実行履歴を取得できませんでした。')
    }
  }
  async function dryRunWorkflow() {
    if (!selected) return
    const target = window.prompt('対象ユーザー ID')?.trim()
    if (!target) return
    try {
      const result = await dryRunLifecycleWorkflow(csrfToken, selected.id, target)
      setDryRun(
        result.steps.map(
          (step) =>
            `${workflowActionLabel(step.action_kind as Parameters<typeof workflowActionLabel>[0])}: ${dryRunOutcomeLabel(step.would_change)}`,
        ),
      )
    } catch {
      setError('実行前確認を実行できませんでした。')
    }
  }
  async function retry(run: WorkflowRun) {
    try {
      const next = await retryLifecycleWorkflowRun(csrfToken, run.id)
      setRuns(runs.map((item) => (item.id === next.id ? next : item)))
    } catch {
      setError('再実行を開始できませんでした。')
    }
  }
  async function deleteWorkflow(workflow: AdminLifecycleWorkflow) {
    if (!window.confirm(`「${workflow.name}」を削除しますか？`)) return
    try {
      await deleteLifecycleWorkflow(csrfToken, workflow.id, workflow.current_revision)
      setWorkflows((current) => current.filter((item) => item.id !== workflow.id))
      if (selected?.id === workflow.id) {
        setSelected(null)
        setRuns([])
        setDryRun([])
      }
      setError('')
    } catch {
      setError('ワークフローを削除できませんでした。')
    }
  }
  return (
    <AdminShell
      active="workflows"
      actorUsername={actorUsername}
      title="ライフサイクルワークフロー"
      description="ユーザーの変化をきっかけに、自動で行う処理を管理します。"
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="mb-6 flex justify-end">
        <Button asChild>
          <a href={tenantURL('/admin/lifecycle-workflows/new')}>新規作成</a>
        </Button>
      </div>
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left">
            <tr>
              <th className="p-3">名前</th>
              <th className="p-3">状態</th>
              <th className="p-3">トリガー</th>
              <th className="p-3">アクション</th>
              <th className="p-3" />
            </tr>
          </thead>
          <tbody>
            {workflows.map((workflow) => (
              <tr
                className="cursor-pointer border-t hover:bg-slate-50"
                key={workflow.id}
                onClick={() => selectWorkflow(workflow)}
              >
                <td className="p-3 font-medium">
                  {workflow.name}
                  <p className="text-xs text-slate-500">第{workflow.current_revision}版</p>
                </td>
                <td className="p-3">{workflowStatusLabel(workflow.status)}</td>
                <td className="p-3">{workflowTriggerLabel(workflow.trigger.kind)}</td>
                <td className="p-3">
                  {workflow.actions.map((action) => workflowActionLabel(action.kind)).join(' → ')}
                </td>
                <td className="flex gap-2 p-3">
                  <Button asChild variant="outline">
                    <a
                      href={tenantURL(
                        `/admin/lifecycle-workflows/${encodeURIComponent(workflow.id)}/edit`,
                      )}
                      onClick={(event) => event.stopPropagation()}
                    >
                      編集
                    </a>
                  </Button>
                  <Button
                    variant="outline"
                    onClick={(event) => {
                      event.stopPropagation()
                      selectWorkflow(workflow)
                    }}
                  >
                    履歴
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={(event) => {
                      event.stopPropagation()
                      deleteWorkflow(workflow)
                    }}
                  >
                    削除
                  </Button>
                  <Button
                    variant="outline"
                    disabled={workflow.status === 'archived'}
                    onClick={(event) => {
                      event.stopPropagation()
                      toggle(workflow)
                    }}
                  >
                    {workflow.status === 'archived'
                      ? '操作不可'
                      : workflow.status === 'enabled'
                        ? '無効化'
                        : '有効化'}
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
      {selected ? (
        <Card className="mt-6 p-4">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">{selected.name} の実行状況</h2>
            <Button variant="outline" onClick={dryRunWorkflow}>
              実行前確認
            </Button>
          </div>
          {dryRun.length ? (
            <ul className="mt-3 list-disc pl-5 text-sm">
              {dryRun.map((line) => (
                <li key={line}>{line}</li>
              ))}
            </ul>
          ) : null}
          <div className="mt-4 space-y-2">
            {runs.map((run) => (
              <div className="rounded border p-3 text-sm" key={run.id}>
                <div className="flex items-center justify-between">
                  <span>
                    {runStatusLabel(run.status)} · 第{run.revision}版
                  </span>
                  {run.status === 'failed' || run.status === 'partially_failed' ? (
                    <Button variant="outline" onClick={() => retry(run)}>
                      再実行
                    </Button>
                  ) : null}
                </div>
                <p className="mt-1 text-xs text-slate-500">
                  {run.steps
                    .map(
                      (step) =>
                        `${workflowActionLabel(step.action_kind as Parameters<typeof workflowActionLabel>[0])}: ${stepOutcomeLabel(step.outcome)}${step.error_code ? `（${step.error_code}）` : ''}`,
                    )
                    .join(' · ')}
                </p>
              </div>
            ))}
          </div>
        </Card>
      ) : null}
    </AdminShell>
  )
}

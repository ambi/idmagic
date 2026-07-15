import { useState } from 'react'
import {
  createLifecycleWorkflow,
  dryRunLifecycleWorkflow,
  listLifecycleWorkflowRuns,
  retryLifecycleWorkflowRun,
  setLifecycleWorkflowState,
} from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import type { AdminLifecycleWorkflow, WorkflowRun } from '../../types'

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
  const [name, setName] = useState('')
  const [runs, setRuns] = useState<WorkflowRun[]>([])
  const [selected, setSelected] = useState<AdminLifecycleWorkflow | null>(null)
  const [dryRun, setDryRun] = useState<string[]>([])
  async function create() {
    try {
      const workflow = await createLifecycleWorkflow(csrfToken, {
        name,
        trigger: { kind: 'user_created' },
        actions: [{ kind: 'send_email', template_key: 'welcome' }],
      })
      setWorkflows([...workflows, workflow])
      setName('')
      setError('')
    } catch {
      setError('ワークフローを作成できませんでした。')
    }
  }
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
      setDryRun(result.steps.map((step) => `${step.action_kind}: ${step.would_change}`))
    } catch {
      setError('dry-run を実行できませんでした。')
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
  return (
    <AdminShell
      active="workflows"
      actorUsername={actorUsername}
      title="ライフサイクルワークフロー"
      description="構造化されたユーザー変更後 action を管理します。"
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="mb-6 flex gap-2 p-4">
        <Input
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="ワークフロー名"
        />
        <Button disabled={!name.trim()} onClick={create}>
          作成
        </Button>
      </Card>
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left">
            <tr>
              <th className="p-3">名前</th>
              <th className="p-3">状態</th>
              <th className="p-3">トリガー</th>
              <th className="p-3">Action</th>
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
                  <p className="text-xs text-slate-500">revision {workflow.current_revision}</p>
                </td>
                <td className="p-3">{workflow.status}</td>
                <td className="p-3">{workflow.trigger.kind}</td>
                <td className="p-3">{workflow.actions.map((action) => action.kind).join(', ')}</td>
                <td className="p-3">
                  <Button
                    variant="outline"
                    onClick={(event) => {
                      event.stopPropagation()
                      toggle(workflow)
                    }}
                  >
                    {workflow.status === 'enabled' ? '無効化' : '有効化'}
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
              dry-run
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
                    {run.status} · revision {run.revision}
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
                        `${step.action_kind}: ${step.outcome}${step.error_code ? ` (${step.error_code})` : ''}`,
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

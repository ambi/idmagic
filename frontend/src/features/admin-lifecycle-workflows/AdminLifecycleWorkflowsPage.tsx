import { IconPlus } from '@tabler/icons-react'
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
import { useDictionary } from '../../lib/i18n'
import type {
  AdminLifecycleWorkflow,
  WorkflowActionKind,
  WorkflowRun,
  WorkflowTrigger,
} from '../../types'
import { adminLifecycleWorkflowsDictionary } from './AdminLifecycleWorkflowsPage.i18n'

type WorkflowsDictionary = (typeof adminLifecycleWorkflowsDictionary)['ja']

function statusLabel(status: AdminLifecycleWorkflow['status'], t: WorkflowsDictionary): string {
  return t[`status_${status}` as const] ?? status
}

function triggerLabel(kind: WorkflowTrigger['kind'], t: WorkflowsDictionary): string {
  return t[`trigger_${kind}` as const] ?? t.trigger_unknown
}

function actionLabel(kind: WorkflowActionKind, t: WorkflowsDictionary): string {
  return t[`action_${kind}` as const] ?? t.action_unknown
}

function runStatusLabel(status: string, t: WorkflowsDictionary): string {
  return t[`runStatus_${status}` as keyof WorkflowsDictionary] ?? t.runStatus_unknown
}

function stepOutcomeLabel(outcome: string, t: WorkflowsDictionary): string {
  return t[`stepOutcome_${outcome}` as keyof WorkflowsDictionary] ?? t.stepOutcome_unknown
}

function dryRunOutcomeLabel(outcome: string, t: WorkflowsDictionary): string {
  return t[`dryRunOutcome_${outcome}` as keyof WorkflowsDictionary] ?? t.dryRunOutcome_unknown
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
  const t = useDictionary(adminLifecycleWorkflowsDictionary)
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
      setError(t.stateChangeError)
    }
  }
  async function selectWorkflow(workflow: AdminLifecycleWorkflow) {
    try {
      setSelected(workflow)
      setRuns(await listLifecycleWorkflowRuns(workflow.id))
      setDryRun([])
    } catch {
      setError(t.runsFetchError)
    }
  }
  async function dryRunWorkflow() {
    if (!selected) return
    const target = window.prompt(t.dryRunPrompt)?.trim()
    if (!target) return
    try {
      const result = await dryRunLifecycleWorkflow(csrfToken, selected.id, target)
      setDryRun(
        result.steps.map(
          (step) =>
            `${actionLabel(step.action_kind as WorkflowActionKind, t)}: ${dryRunOutcomeLabel(step.would_change, t)}`,
        ),
      )
    } catch {
      setError(t.dryRunError)
    }
  }
  async function retry(run: WorkflowRun) {
    try {
      const next = await retryLifecycleWorkflowRun(csrfToken, run.id)
      setRuns(runs.map((item) => (item.id === next.id ? next : item)))
    } catch {
      setError(t.retryError)
    }
  }
  async function deleteWorkflow(workflow: AdminLifecycleWorkflow) {
    if (!window.confirm(t.deleteConfirm.replace('{name}', workflow.name))) return
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
      setError(t.deleteError)
    }
  }
  return (
    <AdminShell
      active="workflows"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="mb-6 flex justify-end">
        <Button asChild>
          <a href={tenantURL('/admin/lifecycle-workflows/new')}>
            <IconPlus size={16} aria-hidden="true" />
            {t.addWorkflow}
          </a>
        </Button>
      </div>
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left">
            <tr>
              <th className="p-3">{t.tableHeaderName}</th>
              <th className="p-3">{t.tableHeaderStatus}</th>
              <th className="p-3">{t.tableHeaderTrigger}</th>
              <th className="p-3">{t.tableHeaderActions}</th>
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
                  <p className="text-xs text-slate-500">
                    {t.revisionLabel.replace('{revision}', String(workflow.current_revision))}
                  </p>
                </td>
                <td className="p-3">{statusLabel(workflow.status, t)}</td>
                <td className="p-3">{triggerLabel(workflow.trigger.kind, t)}</td>
                <td className="p-3">
                  {workflow.actions.map((action) => actionLabel(action.kind, t)).join(' → ')}
                </td>
                <td className="flex gap-2 p-3">
                  <Button asChild variant="outline">
                    <a
                      href={tenantURL(
                        `/admin/lifecycle-workflows/${encodeURIComponent(workflow.id)}/edit`,
                      )}
                      onClick={(event) => event.stopPropagation()}
                    >
                      {t.edit}
                    </a>
                  </Button>
                  <Button
                    variant="outline"
                    onClick={(event) => {
                      event.stopPropagation()
                      selectWorkflow(workflow)
                    }}
                  >
                    {t.history}
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={(event) => {
                      event.stopPropagation()
                      deleteWorkflow(workflow)
                    }}
                  >
                    {t.delete}
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
                      ? t.toggleUnavailable
                      : workflow.status === 'enabled'
                        ? t.disable
                        : t.enable}
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
            <h2 className="font-semibold">{t.executionHeading.replace('{name}', selected.name)}</h2>
            <Button variant="outline" onClick={dryRunWorkflow}>
              {t.dryRunButton}
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
                    {runStatusLabel(run.status, t)} ·{' '}
                    {t.revisionLabel.replace('{revision}', String(run.revision))}
                  </span>
                  {run.status === 'failed' || run.status === 'partially_failed' ? (
                    <Button variant="outline" onClick={() => retry(run)}>
                      {t.retry}
                    </Button>
                  ) : null}
                </div>
                <p className="mt-1 text-xs text-slate-500">
                  {run.steps
                    .map(
                      (step) =>
                        `${actionLabel(step.action_kind as WorkflowActionKind, t)}: ${stepOutcomeLabel(step.outcome, t)}${step.error_code ? `（${step.error_code}）` : ''}`,
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

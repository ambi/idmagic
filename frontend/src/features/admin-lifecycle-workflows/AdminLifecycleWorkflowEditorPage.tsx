import { IconArrowLeft } from '@tabler/icons-react'
import { useState } from 'react'
import {
  createLifecycleWorkflow,
  updateLifecycleWorkflow,
  type LifecycleWorkflowInput,
} from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import type { AdminApplication, AdminGroup, AdminLifecycleWorkflow } from '../../types'
import { tenantURL } from '../../api/core'
import { useDictionary } from '../../lib/i18n'
import { WorkflowDefinitionForm } from './WorkflowDefinitionForm'
import { workflowFormDictionary } from './WorkflowDefinitionForm.i18n'

function EditorLayout({
  actorUsername,
  title,
  description,
  error,
  children,
}: {
  actorUsername?: string
  title: string
  description: string
  error: string
  children: React.ReactNode
}) {
  const t = useDictionary(workflowFormDictionary)
  return (
    <AdminShell
      active="workflows"
      actorUsername={actorUsername}
      title={title}
      description={description}
    >
      <div className="mb-6 flex items-center gap-3">
        <a
          href={tenantURL('/admin/lifecycle-workflows')}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToList}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <span className="text-sm text-slate-600">{t.backToList}</span>
      </div>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="max-w-4xl">{children}</div>
    </AdminShell>
  )
}

export function AdminLifecycleWorkflowCreatePage({
  csrfToken,
  actorUsername,
  groups,
  applications,
}: {
  csrfToken: string
  actorUsername?: string
  groups: AdminGroup[]
  applications: AdminApplication[]
}) {
  const t = useDictionary(workflowFormDictionary)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function create(input: LifecycleWorkflowInput) {
    setBusy(true)
    setError('')
    try {
      const workflow = await createLifecycleWorkflow(csrfToken, input)
      window.location.assign(
        tenantURL(`/admin/lifecycle-workflows/${encodeURIComponent(workflow.id)}/edit`),
      )
    } catch {
      setError(t.createError)
      setBusy(false)
    }
  }

  return (
    <EditorLayout
      actorUsername={actorUsername}
      title={t.createTitle}
      description={t.createDescription}
      error={error}
    >
      <WorkflowDefinitionForm
        groups={groups}
        applications={applications}
        busy={busy}
        onSubmit={create}
        onCancel={() => window.location.assign(tenantURL('/admin/lifecycle-workflows'))}
      />
    </EditorLayout>
  )
}

export function AdminLifecycleWorkflowEditPage({
  csrfToken,
  actorUsername,
  initialWorkflow,
  groups,
  applications,
}: {
  csrfToken: string
  actorUsername?: string
  initialWorkflow: AdminLifecycleWorkflow
  groups: AdminGroup[]
  applications: AdminApplication[]
}) {
  const t = useDictionary(workflowFormDictionary)
  const [workflow, setWorkflow] = useState(initialWorkflow)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function update(input: LifecycleWorkflowInput) {
    setBusy(true)
    setError('')
    try {
      const updated = await updateLifecycleWorkflow(csrfToken, workflow.id, {
        ...input,
        expected_revision: workflow.current_revision,
      })
      setWorkflow(updated)
    } catch {
      setError(t.updateError)
    } finally {
      setBusy(false)
    }
  }

  return (
    <EditorLayout
      actorUsername={actorUsername}
      title={t.editTitle}
      description={t.editDescription
        .replace('{name}', workflow.name)
        .replace('{revision}', String(workflow.current_revision))}
      error={error}
    >
      <WorkflowDefinitionForm
        workflow={workflow}
        groups={groups}
        applications={applications}
        busy={busy}
        onSubmit={update}
        onCancel={() => window.location.assign(tenantURL('/admin/lifecycle-workflows'))}
      />
    </EditorLayout>
  )
}

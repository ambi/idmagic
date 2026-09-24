import { IconRefresh } from '@tabler/icons-react'
import { useCallback, useEffect, useState } from 'react'
import {
  getAdminApplicationProvisioningTask,
  listAdminApplicationProvisioningTasks,
  retryAdminApplicationProvisioningTask,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Select } from '../../components/ui/select'
import { useDictionary, useLocale } from '../../lib/i18n'
import { messageOf, SectionTitle } from './AdminApplicationsShared'
import {
  provisioningDictionary,
  type ProvisioningDictionary,
} from './AdminApplicationProvisioning.i18n'
import { formatDate } from './AdminApplicationProvisioningShared'
import type { ProvisioningTask, ProvisioningTaskStatus } from '../../types'

function taskStatusOptions(t: ProvisioningDictionary) {
  return [
    { value: '', label: t.taskStatusAll },
    { value: 'pending', label: t.taskStatusPending },
    { value: 'in_flight', label: t.taskStatusInFlight },
    { value: 'succeeded', label: t.taskStatusSucceeded },
    { value: 'dead_letter', label: t.taskStatusDeadLetter },
  ]
}

function operationLabel(op: ProvisioningTask['operation'], t: ProvisioningDictionary): string {
  switch (op) {
    case 'create':
      return t.operationCreate
    case 'update':
      return t.operationUpdate
    case 'deactivate':
      return t.operationDeactivate
    case 'delete':
      return t.operationDelete
    case 'membership_add':
      return t.operationMembershipAdd
    case 'membership_remove':
      return t.operationMembershipRemove
  }
}

function taskStatusBadge(status: ProvisioningTaskStatus): string {
  switch (status) {
    case 'succeeded':
      return 'bg-emerald-50 text-emerald-700'
    case 'dead_letter':
      return 'bg-red-50 text-red-700'
    case 'in_flight':
      return 'bg-blue-50 text-blue-700'
    default:
      return 'bg-slate-100 text-slate-500'
  }
}

export function TasksPanel({
  csrfToken,
  applicationID,
}: {
  csrfToken: string
  applicationID: string
}) {
  const t = useDictionary(provisioningDictionary)
  const { locale } = useLocale()
  const [statusFilter, setStatusFilter] = useState('')
  const [tasks, setTasks] = useState<ProvisioningTask[]>([])
  const [selected, setSelected] = useState<ProvisioningTask | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [retrying, setRetrying] = useState(false)

  const reload = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const status = statusFilter === '' ? undefined : (statusFilter as ProvisioningTaskStatus)
      const list = await listAdminApplicationProvisioningTasks(applicationID, status)
      setTasks(list)
    } catch (cause) {
      setError(messageOf(cause, t.tasksLoadFailedError))
    } finally {
      setLoading(false)
    }
  }, [applicationID, statusFilter, t.tasksLoadFailedError])

  useEffect(() => {
    void reload()
  }, [reload])

  async function selectTask(d: ProvisioningTask) {
    setSelected(d)
    try {
      setSelected(await getAdminApplicationProvisioningTask(applicationID, d.id))
    } catch {
      // 一覧の値のまま表示を継続する
    }
  }

  async function retry() {
    if (!selected) return
    setRetrying(true)
    setError('')
    try {
      const updated = await retryAdminApplicationProvisioningTask(
        csrfToken,
        applicationID,
        selected.id,
      )
      setSelected(updated)
      await reload()
    } catch (cause) {
      setError(messageOf(cause, t.retryFailedError))
    } finally {
      setRetrying(false)
    }
  }

  return (
    <Card className="grid gap-4 p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <SectionTitle>{t.tasksHeading}</SectionTitle>
        <div className="flex items-center gap-2">
          <Select
            value={statusFilter}
            onValueChange={setStatusFilter}
            options={taskStatusOptions(t)}
            aria-label={t.taskStatusFilterLabel}
          />
          <Button
            type="button"
            variant="ghost"
            aria-label={t.tasksReloadAria}
            onClick={() => void reload()}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
        </div>
      </div>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <TasksTable
          tasks={tasks}
          loading={loading}
          selected={selected}
          onSelect={(d) => void selectTask(d)}
          locale={locale}
          t={t}
        />
        <TaskDetailCard
          selected={selected}
          retrying={retrying}
          onRetry={() => void retry()}
          locale={locale}
          t={t}
        />
      </div>
    </Card>
  )
}

function TasksTable({
  tasks,
  loading,
  selected,
  onSelect,
  locale,
  t,
}: {
  tasks: ProvisioningTask[]
  loading: boolean
  selected: ProvisioningTask | null
  onSelect: (d: ProvisioningTask) => void
  locale: string
  t: ProvisioningDictionary
}) {
  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-3 py-2">{t.tableHeaderUpdatedAt}</th>
            <th className="px-3 py-2">{t.tableHeaderSource}</th>
            <th className="px-3 py-2">{t.tableHeaderOperation}</th>
            <th className="px-3 py-2">{t.tableHeaderStatus}</th>
          </tr>
        </thead>
        <tbody>
          {!loading && tasks.length === 0 ? (
            <tr>
              <td colSpan={4} className="px-3 py-8 text-center text-xs text-slate-500">
                {t.tasksEmptyNotice}
              </td>
            </tr>
          ) : null}
          {tasks.map((d) => (
            <tr
              key={d.id}
              onClick={() => onSelect(d)}
              className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${
                selected?.id === d.id ? 'bg-blue-50/60' : ''
              }`}
            >
              <td className="px-3 py-2 font-mono text-xs">
                {formatDate(d.updated_at, locale, t.unknownDate)}
              </td>
              <td className="px-3 py-2 font-mono text-xs">
                {d.source_type === 'user' ? t.sourceTypeUser : t.sourceTypeGroup}:{d.source_id}
              </td>
              <td className="px-3 py-2 text-xs">{operationLabel(d.operation, t)}</td>
              <td className="px-3 py-2">
                <span
                  className={`rounded px-2 py-0.5 text-xs font-medium ${taskStatusBadge(d.status)}`}
                >
                  {taskStatusOptions(t).find((o) => o.value === d.status)?.label ?? d.status}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function TaskDetailCard({
  selected,
  retrying,
  onRetry,
  locale,
  t,
}: {
  selected: ProvisioningTask | null
  retrying: boolean
  onRetry: () => void
  locale: string
  t: ProvisioningDictionary
}) {
  return (
    <Card className="p-4">
      <h3 className="text-sm font-semibold text-slate-700">{t.taskDetailHeading}</h3>
      {selected ? (
        <dl className="mt-3 grid grid-cols-[110px_minmax(0,1fr)] gap-y-2 text-xs">
          <dt className="text-slate-500">{t.taskIdLabel}</dt>
          <dd className="break-all font-mono">{selected.id}</dd>
          <dt className="text-slate-500">{t.tableHeaderStatus}</dt>
          <dd>{taskStatusOptions(t).find((o) => o.value === selected.status)?.label}</dd>
          <dt className="text-slate-500">{t.taskSourceVersionLabel}</dt>
          <dd className="font-mono">{selected.source_version}</dd>
          {selected.job_id ? (
            <>
              <dt className="text-slate-500">{t.taskJobIdLabel}</dt>
              <dd className="break-all font-mono">{selected.job_id}</dd>
            </>
          ) : null}
          {selected.last_error ? (
            <>
              <dt className="text-slate-500">{t.taskLastErrorLabel}</dt>
              <dd className="break-all text-red-700">{selected.last_error}</dd>
            </>
          ) : null}
          {selected.completed_at ? (
            <>
              <dt className="text-slate-500">{t.taskCompletedAtLabel}</dt>
              <dd>{formatDate(selected.completed_at, locale, t.unknownDate)}</dd>
            </>
          ) : null}
        </dl>
      ) : (
        <p className="mt-3 text-xs text-slate-500">{t.selectTaskPrompt}</p>
      )}
      {selected && selected.status === 'dead_letter' ? (
        <Button
          type="button"
          variant="outline"
          className="mt-3"
          disabled={retrying}
          onClick={onRetry}
        >
          {retrying ? t.retrying : t.retryButton}
        </Button>
      ) : null}
    </Card>
  )
}

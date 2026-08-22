import { IconRefresh, IconSearch, IconX } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  type AdminJob,
  type AdminJobLane,
  type AdminJobQuery,
  type AdminJobStatus,
  AuthenticationAPIError,
  cancelAdminJob,
  listAdminJobs,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import { adminJobsDictionary, type AdminJobsDictionary } from './AdminJobsPage.i18n'

const DEFAULT_REALM = 'default'

const STATUSES: AdminJobStatus[] = ['queued', 'running', 'succeeded', 'failed', 'canceled']
const LANES: AdminJobLane[] = ['latency_sensitive', 'default', 'bulk']

// 終端に達したジョブは取り消せない。UI で隠すのは認可判定ではないので、
// サーバー側も同じ条件を fail-closed で確かめる。
function isCancelable(job: AdminJob): boolean {
  return job.status === 'queued' || job.status === 'running'
}

function statusLabel(status: AdminJobStatus, t: AdminJobsDictionary): string {
  return {
    queued: t.statusQueued,
    running: t.statusRunning,
    succeeded: t.statusSucceeded,
    failed: t.statusFailed,
    canceled: t.statusCanceled,
  }[status]
}

function laneLabel(lane: AdminJobLane, t: AdminJobsDictionary): string {
  return {
    latency_sensitive: t.laneLatencySensitive,
    default: t.laneDefault,
    bulk: t.laneBulk,
  }[lane]
}

// 状態は運用者が一覧を走査して異常を拾うための最初の手がかりなので、色で区別する。
const STATUS_BADGE: Record<AdminJobStatus, string> = {
  queued: 'bg-slate-100 text-slate-700',
  running: 'bg-blue-50 text-blue-700',
  succeeded: 'bg-emerald-50 text-emerald-700',
  failed: 'bg-rose-50 text-rose-700',
  canceled: 'bg-amber-50 text-amber-700',
}

export function AdminJobsPage({
  actorUsername,
  actorRoles,
  actorRealm,
  jobs: initial,
  nextCursor: initialNextCursor,
  kinds,
  csrfToken,
  initialError,
}: {
  actorUsername?: string
  actorRoles: string[]
  actorRealm: string
  jobs: AdminJob[]
  nextCursor?: string
  // kinds は登録済みの JobKind 一覧。UI で固定せず、読み込んだジョブから導く。
  kinds: string[]
  csrfToken: string
  initialError?: string
}) {
  const [jobs, setJobs] = useState(initial)
  const [nextCursor, setNextCursor] = useState(initialNextCursor)
  const [selected, setSelected] = useState<AdminJob | null>(initial[0] ?? null)
  const [status, setStatus] = useState<AdminJobStatus | ''>('')
  const [kind, setKind] = useState('')
  const [lane, setLane] = useState<AdminJobLane | ''>('')
  const [allTenants, setAllTenants] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(initialError ?? '')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminJobsDictionary)
  const { locale } = useLocale()

  const canCrossTenant = actorRoles.includes('system_admin') && actorRealm === DEFAULT_REALM

  function buildQuery(cursor?: string): AdminJobQuery {
    return {
      status: status ? [status] : undefined,
      kind: kind ? [kind] : undefined,
      lane: lane || undefined,
      allTenants: canCrossTenant && allTenants,
      cursor,
    }
  }

  async function handleQuery(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const page = await listAdminJobs(buildQuery())
      setJobs(page.jobs)
      setNextCursor(page.next_cursor)
      setSelected(page.jobs[0] ?? null)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.jobsFetchFailedError)
    } finally {
      setBusy(false)
    }
  }

  async function handleLoadMore() {
    if (!nextCursor) return
    setBusy(true)
    setError('')
    try {
      const page = await listAdminJobs(buildQuery(nextCursor))
      setJobs((current) => [...current, ...page.jobs])
      setNextCursor(page.next_cursor)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.jobsFetchFailedError)
    } finally {
      setBusy(false)
    }
  }

  async function handleCancel(job: AdminJob) {
    if (!window.confirm(t.cancelConfirm)) return
    setBusy(true)
    setError('')
    try {
      const canceled = await cancelAdminJob(csrfToken, job.id)
      setJobs((current) => current.map((j) => (j.id === canceled.id ? canceled : j)))
      setSelected(canceled)
      setNotice(t.cancelSucceededNotice)
    } catch (cause) {
      // 409 は「すでに終わっていた」ことの通知であり、失敗ではなく事実として伝える。
      if (cause instanceof AuthenticationAPIError && cause.code === 'job_not_cancelable') {
        setError(t.cancelNotAllowedError)
      } else {
        setError(cause instanceof AuthenticationAPIError ? cause.message : t.cancelFailedError)
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="jobs"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <Card className="p-5">
        <form onSubmit={handleQuery} className="grid gap-4 lg:grid-cols-4">
          <Field label={t.statusFieldLabel}>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value as AdminJobStatus | '')}
              className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
            >
              <option value="">{t.anyOption}</option>
              {STATUSES.map((value) => (
                <option key={value} value={value}>
                  {statusLabel(value, t)}
                </option>
              ))}
            </select>
          </Field>
          <Field label={t.kindFieldLabel}>
            <select
              value={kind}
              onChange={(e) => setKind(e.target.value)}
              className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
            >
              <option value="">{t.anyOption}</option>
              {kinds.map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </Field>
          <Field label={t.laneFieldLabel}>
            <select
              value={lane}
              onChange={(e) => setLane(e.target.value as AdminJobLane | '')}
              className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
            >
              <option value="">{t.anyOption}</option>
              {LANES.map((value) => (
                <option key={value} value={value}>
                  {laneLabel(value, t)}
                </option>
              ))}
            </select>
          </Field>
          <div className="flex items-end">
            <Button type="submit" disabled={busy}>
              <IconSearch size={16} aria-hidden="true" />
              {t.filterAction}
            </Button>
          </div>
        </form>
        {canCrossTenant ? (
          <label className="mt-4 inline-flex items-center gap-2 text-sm text-slate-700">
            <input
              type="checkbox"
              checked={allTenants}
              onChange={(e) => setAllTenants(e.target.checked)}
              className="size-4 rounded border-slate-300"
            />
            {t.crossTenantLabel}
          </label>
        ) : null}
      </Card>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_420px]">
        <Card className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="px-4 py-3">{t.tableHeaderKind}</th>
                  <th className="px-4 py-3">{t.tableHeaderStatus}</th>
                  <th className="px-4 py-3">{t.tableHeaderLane}</th>
                  <th className="px-4 py-3">{t.tableHeaderAttempts}</th>
                  <th className="px-4 py-3">{t.tableHeaderCreatedAt}</th>
                </tr>
              </thead>
              <tbody>
                {jobs.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-12 text-center text-sm text-slate-500">
                      {t.noMatchingJobsNotice}
                    </td>
                  </tr>
                ) : null}
                {jobs.map((job) => (
                  <tr
                    key={job.id}
                    onClick={() => setSelected(job)}
                    className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${
                      selected?.id === job.id ? 'bg-blue-50/60' : ''
                    }`}
                  >
                    <td className="px-4 py-3 font-mono text-xs">{job.kind}</td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded px-2 py-0.5 text-xs font-medium ${STATUS_BADGE[job.status]}`}
                      >
                        {statusLabel(job.status, t)}
                      </span>
                    </td>
                    <td className="px-4 py-3">{laneLabel(job.lane, t)}</td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {job.attempts}/{job.max_attempts}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {formatDate(job.created_at, locale)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {nextCursor ? (
            <div className="border-t border-slate-100 p-3 text-center">
              <Button type="button" variant="ghost" onClick={handleLoadMore} disabled={busy}>
                <IconRefresh size={14} aria-hidden="true" />
                {t.loadMore}
              </Button>
            </div>
          ) : null}
        </Card>

        <Card className="p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-slate-700">{t.detailHeading}</h2>
            {selected && isCancelable(selected) ? (
              <Button
                variant="ghost"
                onClick={() => handleCancel(selected)}
                disabled={busy}
                className="text-rose-700"
              >
                <IconX size={14} aria-hidden="true" />
                {t.cancelAction}
              </Button>
            ) : null}
          </div>
          {selected ? (
            <>
              <dl className="mt-4 grid grid-cols-[130px_minmax(0,1fr)] gap-y-2 text-xs">
                <dt className="text-slate-500">{t.idLabel}</dt>
                <dd className="break-all font-mono">{selected.id}</dd>
                <dt className="text-slate-500">{t.tableHeaderKind}</dt>
                <dd className="font-mono">{selected.kind}</dd>
                <dt className="text-slate-500">{t.tableHeaderStatus}</dt>
                <dd>{statusLabel(selected.status, t)}</dd>
                <dt className="text-slate-500">{t.tableHeaderLane}</dt>
                <dd>{laneLabel(selected.lane, t)}</dd>
                <dt className="text-slate-500">{t.tableHeaderTenant}</dt>
                <dd className="break-all font-mono">{selected.tenant_id}</dd>
                <dt className="text-slate-500">{t.attemptsLabel}</dt>
                <dd className="font-mono">
                  {selected.attempts}/{selected.max_attempts}
                </dd>
                <dt className="text-slate-500">{t.createdAtLabel}</dt>
                <dd>{formatDate(selected.created_at, locale)}</dd>
                <dt className="text-slate-500">{t.runAtLabel}</dt>
                <dd>{formatDate(selected.run_at, locale)}</dd>
                <dt className="text-slate-500">{t.updatedAtLabel}</dt>
                <dd>{formatDate(selected.updated_at, locale)}</dd>
                {selected.lease_owner ? (
                  <>
                    <dt className="text-slate-500">{t.leaseOwnerLabel}</dt>
                    <dd className="break-all font-mono">{selected.lease_owner}</dd>
                  </>
                ) : null}
                {selected.lease_expires_at ? (
                  <>
                    <dt className="text-slate-500">{t.leaseExpiresAtLabel}</dt>
                    <dd>{formatDate(selected.lease_expires_at, locale)}</dd>
                  </>
                ) : null}
                {selected.progress ? (
                  <>
                    <dt className="text-slate-500">{t.progressLabel}</dt>
                    <dd>
                      {selected.progress.percent !== undefined
                        ? `${selected.progress.percent}%`
                        : null}
                      {selected.progress.message ? ` ${selected.progress.message}` : null}
                    </dd>
                  </>
                ) : null}
              </dl>
              {selected.error ? (
                <div className="mt-4">
                  <Label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                    {t.errorLabel}
                  </Label>
                  <pre className="mt-1 max-h-48 overflow-auto whitespace-pre-wrap rounded-md bg-rose-50 p-3 text-xs text-rose-900">
                    {selected.error}
                  </pre>
                </div>
              ) : null}
              <p className="mt-4 text-xs text-slate-500">{t.payloadOmittedNotice}</p>
            </>
          ) : (
            <p className="mt-4 text-sm text-slate-500">{t.selectJobPrompt}</p>
          )}
        </Card>
      </div>
    </AdminShell>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid gap-1.5">
      <Label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
        {label}
      </Label>
      {children}
    </div>
  )
}

function formatDate(value: string, locale: 'ja' | 'en'): string {
  try {
    return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
  } catch {
    return value
  }
}

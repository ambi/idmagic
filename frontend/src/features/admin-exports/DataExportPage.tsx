import { IconArrowLeft, IconClock, IconDownload, IconFileExport, IconX } from '@tabler/icons-react'
import { useEffect, useMemo, useState } from 'react'
import { type ExportCollection, getTenantUserAttributeSchema, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { DataExportJob, DataExportStatus } from '../../types'
import { dataExportDictionary } from './DataExportPage.i18n'
import { EXPORT_COLUMNS, type ExportColumn, type ExportTarget } from './dataExportColumns'

const EXPORT_POLL_INTERVAL_MS = 2000

function isTerminal(status: DataExportStatus): boolean {
  return (
    status === 'succeeded' || status === 'failed' || status === 'canceled' || status === 'expired'
  )
}

function statusLabel(t: typeof dataExportDictionary.ja, status: DataExportStatus): string {
  switch (status) {
    case 'queued':
      return t.statusQueued
    case 'running':
      return t.statusRunning
    case 'succeeded':
      return t.statusSucceeded
    case 'failed':
      return t.statusFailed
    case 'canceled':
      return t.statusCanceled
    case 'expired':
      return t.statusExpired
  }
}

function exportErrorLabel(t: typeof dataExportDictionary.ja, code: string | undefined): string {
  switch (code) {
    case 'export_too_large':
      return t.errorTooLarge
    case undefined:
      return ''
    default:
      return t.errorGeneric
  }
}

function formatDateTime(value: string | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

export function DataExportPage({
  csrfToken,
  actorUsername,
  target,
  api,
  backPath,
}: {
  csrfToken: string
  actorUsername?: string
  target: ExportTarget
  api: ExportCollection
  backPath?: string
}) {
  const t = useDictionary(dataExportDictionary)
  const baseColumns = EXPORT_COLUMNS[target]
  const listPath = backPath || tenantURL(target === 'users' ? '/admin/users' : '/admin/groups')
  const [customColumns, setCustomColumns] = useState<ExportColumn[]>([])
  const columns = useMemo(() => [...baseColumns, ...customColumns], [baseColumns, customColumns])
  const [selected, setSelected] = useState<Set<string>>(
    () => new Set(baseColumns.map((c) => c.key)),
  )
  const [exports, setExports] = useState<DataExportJob[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const hasPendingExport = useMemo(() => exports.some((e) => !isTerminal(e.status)), [exports])

  async function refresh() {
    try {
      setExports(await api.list())
    } catch {
      setError(t.genericError)
    }
  }

  // 初回ロードと、未終端エクスポートがある間の polling。
  // biome-ignore lint/correctness/useExhaustiveDependencies: 初回マウント時のみ取得する
  useEffect(() => {
    void refresh()
  }, [])

  // User CSV の custom:<key> は固定 allowlist ではなく tenant の実効 schema から解決する。
  // biome-ignore lint/correctness/useExhaustiveDependencies: target の画面マウント時だけ解決する
  useEffect(() => {
    if (target !== 'users') return
    void getTenantUserAttributeSchema()
      .then((schema) => {
        const resolved = schema.attributes.map((attribute) => ({
          key: `custom:${attribute.key}`,
          label: `${attribute.label} (custom:${attribute.key})`,
          pii: attribute.pii,
        }))
        setCustomColumns(resolved)
        setSelected((previous) => {
          const next = new Set(previous)
          for (const column of resolved) next.add(column.key)
          return next
        })
      })
      .catch(() => setError(t.genericError))
  }, [target])

  // biome-ignore lint/correctness/useExhaustiveDependencies: hasPendingExport の変化でのみ polling を張り直す
  useEffect(() => {
    if (!hasPendingExport) return
    const timer = setInterval(() => void refresh(), EXPORT_POLL_INTERVAL_MS)
    return () => clearInterval(timer)
  }, [hasPendingExport])

  function toggleColumn(key: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  async function startExport() {
    const chosen = columns.map((c) => c.key).filter((key) => selected.has(key))
    if (chosen.length === 0) {
      setError(t.noColumnsSelected)
      return
    }
    setBusy(true)
    setError('')
    setNotice('')
    try {
      const job = await api.start(csrfToken, { columns: chosen })
      setExports((prev) => [job, ...prev])
      setNotice(t.startedNotice)
    } catch {
      setError(t.genericError)
    } finally {
      setBusy(false)
    }
  }

  async function cancelExport(exportId: string) {
    setError('')
    try {
      await api.cancel(csrfToken, exportId)
      setNotice(t.canceledNotice)
      await refresh()
    } catch {
      setError(t.genericError)
    }
  }

  return (
    <AdminShell
      active={target === 'group_members' ? 'groups' : target}
      actorUsername={actorUsername}
      title={
        target === 'users'
          ? t.pageTitleUsers
          : target === 'groups'
            ? t.pageTitleGroups
            : t.pageTitleGroupMembers
      }
      description={t.pageDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={
            target === 'users'
              ? t.backToUsers
              : target === 'groups'
                ? t.backToGroups
                : t.backToGroupDetails
          }
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">
          {target === 'users'
            ? t.pageTitleUsers
            : target === 'groups'
              ? t.pageTitleGroups
              : t.pageTitleGroupMembers}
        </h1>
      </div>

      <div className="mt-6 grid max-w-4xl gap-6">
        {error && <Alert>{error}</Alert>}
        {notice && <Alert variant="success">{notice}</Alert>}

        {target === 'users' && (
          <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-xs leading-5 text-slate-600">
            <p>{t.userTransferPolicyNotice}</p>
            <p>{t.userRoundTripNotice}</p>
          </div>
        )}

        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <div className="grid gap-4 p-6">
            <h2 className="text-sm font-semibold text-slate-900">{t.selectColumnsHeading}</h2>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              {columns.map((column) => (
                <label
                  key={column.key}
                  className="flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700"
                >
                  <input
                    type="checkbox"
                    checked={selected.has(column.key)}
                    onChange={() => toggleColumn(column.key)}
                  />
                  <span>{column.label ?? (column.labelKey ? t[column.labelKey] : column.key)}</span>
                  {column.pii && (
                    <span className="ml-auto rounded bg-amber-50 px-1.5 py-0.5 text-[10px] font-semibold text-amber-700">
                      {t.piiBadge}
                    </span>
                  )}
                </label>
              ))}
            </div>
            <p className="text-xs text-slate-500">{t.piiNotice}</p>
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button type="button" disabled={busy} onClick={() => void startExport()}>
              <IconFileExport size={16} aria-hidden="true" />
              {busy ? t.exporting : t.startExport}
            </Button>
          </div>
        </Card>

        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
            <h2 className="text-sm font-semibold text-slate-900">{t.exportsListHeading}</h2>
            <Button type="button" variant="outline" onClick={() => void refresh()}>
              {t.refresh}
            </Button>
          </div>
          {exports.length === 0 ? (
            <p className="px-6 py-6 text-sm text-slate-500">{t.emptyExports}</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
                  <tr>
                    <th className="px-4 py-2">{t.statusHeader}</th>
                    <th className="px-4 py-2">{t.columnsHeader}</th>
                    <th className="px-4 py-2">{t.rowsHeader}</th>
                    <th className="px-4 py-2">{t.createdHeader}</th>
                    <th className="px-4 py-2">{t.expiresHeader}</th>
                    <th className="px-4 py-2">{t.actionsHeader}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {exports.map((job) => (
                    <tr key={job.id}>
                      <td className="px-4 py-2">
                        <span className="inline-flex items-center gap-1">
                          {!isTerminal(job.status) && (
                            <IconClock size={14} className="animate-pulse" aria-hidden="true" />
                          )}
                          {statusLabel(t, job.status)}
                        </span>
                        {job.status === 'failed' && job.error_code && (
                          <span className="ml-2 text-xs text-rose-600">
                            {exportErrorLabel(t, job.error_code)}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-2 text-xs text-slate-500">
                        {job.requested_columns.length}
                      </td>
                      <td className="px-4 py-2">{job.total_rows ?? '—'}</td>
                      <td className="px-4 py-2 text-xs text-slate-500">
                        {formatDateTime(job.created_at)}
                      </td>
                      <td className="px-4 py-2 text-xs text-slate-500">
                        {formatDateTime(job.expires_at)}
                      </td>
                      <td className="px-4 py-2">
                        {job.downloadable ? (
                          <Button
                            variant="outline"
                            nativeButton={false}
                            render={<a href={api.fileURL(job.id)} />}
                          >
                            <IconDownload size={14} aria-hidden="true" />
                            {t.download}
                          </Button>
                        ) : isTerminal(job.status) ? null : (
                          <Button
                            type="button"
                            variant="ghost"
                            onClick={() => void cancelExport(job.id)}
                          >
                            <IconX size={14} aria-hidden="true" />
                            {t.cancel}
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      </div>
    </AdminShell>
  )
}

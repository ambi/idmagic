import { useEffect, useState } from 'react'
import type { AdminAuditEventsSearchParams, AuditExport, AuditExportClient } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import { adminAuditEventsDictionary } from './AdminAuditEventsPage.i18n'

export function AuditExportPanel({
  client,
  csrfToken,
  query,
}: {
  client: AuditExportClient
  csrfToken: string
  query: () => AdminAuditEventsSearchParams
}) {
  const t = useDictionary(adminAuditEventsDictionary)
  const [exports, setExports] = useState<AuditExport[]>([])
  const [opened, setOpened] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const active = exports.some((job) => job.status === 'queued' || job.status === 'running')
  useEffect(() => {
    if (!opened || !active) return
    let disposed = false
    const timer = setInterval(() => {
      void client
        .list()
        .then((result) => {
          if (!disposed) setExports(result.exports)
        })
        .catch(() => {
          if (!disposed) setError(t.exportFailed)
        })
    }, 1000)
    return () => {
      disposed = true
      clearInterval(timer)
    }
  }, [client, opened, active, t.exportFailed])

  async function action(operation: () => Promise<void>) {
    setBusy(true)
    setError('')
    try {
      await operation()
      setOpened(true)
    } catch {
      setError(t.exportFailed)
    } finally {
      setBusy(false)
    }
  }
  const status = {
    queued: t.exportQueued,
    running: t.exportRunning,
    succeeded: t.exportSucceeded,
    failed: t.exportFailure,
    canceled: t.exportCanceled,
    expired: t.exportExpired,
  }
  return (
    <Card className="space-y-3 p-4">
      <div className="flex gap-2">
        <Button
          type="button"
          disabled={busy}
          onClick={() =>
            action(async () => {
              const job = await client.start(csrfToken, query())
              setExports((current) => [job, ...current.filter((item) => item.id !== job.id)])
            })
          }
        >
          {t.exportAction}
        </Button>
        <Button
          type="button"
          variant="ghost"
          disabled={busy}
          onClick={() =>
            action(async () => {
              setExports((await client.list()).exports)
            })
          }
        >
          {t.exportHistory}
        </Button>
      </div>
      {error ? <Alert>{error}</Alert> : null}
      {opened ? (
        <div aria-live="polite" className="space-y-3">
          <p className="text-sm text-slate-600">{t.exportHint}</p>
          {exports.map((job) => (
            <div
              key={job.id}
              data-audit-export-id={job.id}
              data-export-status={job.status}
              data-export-scope={job.scope}
              className="flex flex-wrap items-center gap-3 text-sm"
            >
              <time dateTime={job.created_at}>{new Date(job.created_at).toLocaleString()}</time>
              <span>{status[job.status]}</span>
              {job.events !== undefined ? (
                <span>
                  {job.events.toLocaleString()} {t.exportEvents}
                </span>
              ) : null}
              {job.error_code ? (
                <span>
                  {job.error_code === 'event_limit_exceeded' ||
                  job.error_code === 'byte_limit_exceeded'
                    ? t.exportLimit
                    : t.exportFailed}
                </span>
              ) : null}
              {job.downloadable && job.status === 'succeeded' ? (
                <a className="text-blue-700 underline" href={client.downloadURL(job.id)} download>
                  {t.exportDownload}
                </a>
              ) : null}
              {job.expires_at ? (
                <span>
                  {t.exportExpiry}: {new Date(job.expires_at).toLocaleString()}
                </span>
              ) : null}
              {job.status === 'queued' || job.status === 'running' ? (
                <Button
                  type="button"
                  variant="ghost"
                  disabled={busy}
                  onClick={() =>
                    action(async () => {
                      const updated = await client.cancel(csrfToken, job.id)
                      setExports((current) =>
                        current.map((item) => (item.id === job.id ? updated : item)),
                      )
                    })
                  }
                >
                  {t.exportCancel}
                </Button>
              ) : null}
            </div>
          ))}
        </div>
      ) : null}
    </Card>
  )
}

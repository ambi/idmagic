import { Fragment, useEffect, useState } from 'react'
import { AuthenticationAPIError, getDirectoryIngestionRun } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useFormatters } from '../../lib/i18n'
import type { IngestionRunDetail, IngestionRunSummary } from '../../types'
import { type Dictionary, hiddenFailureCount, itemErrorLabel, runStatusLabel } from './presentation'

const CELL = 'whitespace-nowrap px-3 py-2.5'

export function IngestionRunsCard({
  connectorID,
  runs,
  t,
}: {
  connectorID: string
  runs: IngestionRunSummary[]
  t: Dictionary
}) {
  const [openRunID, setOpenRunID] = useState('')
  const { formatDateTime, formatNumber } = useFormatters()

  return (
    <Card className="overflow-hidden">
      <div className="p-4">
        <h2 className="text-base font-semibold text-slate-950">{t.runsHeading}</h2>
        <p className="mt-1 text-sm text-slate-600">{t.runsDescription}</p>
      </div>
      {runs.length === 0 ? (
        <p className="border-t border-slate-100 p-4 text-sm text-slate-500">{t.runsEmptyNotice}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-left text-xs font-semibold text-slate-500">
              <tr>
                <th className={CELL}>{t.headerMode}</th>
                <th className={CELL}>{t.headerRunStatus}</th>
                <th className={CELL}>{t.headerStartedAt}</th>
                <th className={CELL}>{t.headerCompletedAt}</th>
                <th className={CELL}>{t.headerItems}</th>
                <th className={CELL}>{t.headerCreated}</th>
                <th className={CELL}>{t.headerUpdated}</th>
                <th className={CELL}>{t.headerUnchanged}</th>
                <th className={CELL}>{t.headerDeleted}</th>
                <th className={CELL}>{t.headerFailed}</th>
                <th className={CELL}>{t.headerDisabledUsers}</th>
                <th className={CELL} />
              </tr>
            </thead>
            <tbody>
              {runs.map((run) => (
                <Fragment key={run.id}>
                  <tr className="border-t border-slate-100 text-slate-700">
                    <td className={CELL}>{run.mode === 'full' ? t.modeFull : t.modeIncremental}</td>
                    <td className={CELL}>
                      {runStatusLabel(run.status, t)}
                      {run.protection ? (
                        <p className="text-xs text-amber-700">
                          {t.holdSummary
                            .replace('{candidates}', formatNumber(run.protection.candidate_count))
                            .replace('{active}', formatNumber(run.protection.active_count))}
                        </p>
                      ) : null}
                    </td>
                    <td className={CELL}>{formatDateTime(run.started_at)}</td>
                    <td className={CELL}>
                      {run.completed_at ? formatDateTime(run.completed_at) : '—'}
                    </td>
                    <td className={CELL}>{formatNumber(run.counts.items)}</td>
                    <td className={CELL}>{formatNumber(run.counts.created)}</td>
                    <td className={CELL}>{formatNumber(run.counts.updated)}</td>
                    <td className={CELL}>{formatNumber(run.counts.unchanged)}</td>
                    <td className={CELL}>{formatNumber(run.counts.deleted)}</td>
                    <td
                      className={`${CELL} ${run.counts.failed > 0 ? 'font-semibold text-rose-700' : ''}`}
                    >
                      {formatNumber(run.counts.failed)}
                    </td>
                    <td className={CELL}>{formatNumber(run.counts.disabled)}</td>
                    <td className={`${CELL} text-right`}>
                      {run.counts.failed > 0 ? (
                        <Button
                          type="button"
                          variant="outline"
                          onClick={() => setOpenRunID(openRunID === run.id ? '' : run.id)}
                        >
                          {openRunID === run.id ? t.hideFailures : t.showFailures}
                        </Button>
                      ) : null}
                    </td>
                  </tr>
                  {openRunID === run.id ? (
                    <tr className="border-t border-slate-100 bg-slate-50">
                      <td colSpan={12} className="p-4">
                        <RunFailures connectorID={connectorID} runID={run.id} t={t} />
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  )
}

function RunFailures({
  connectorID,
  runID,
  t,
}: {
  connectorID: string
  runID: string
  t: Dictionary
}) {
  const [state, setState] = useState<
    | { kind: 'loading' }
    | { kind: 'failed'; message: string }
    | { kind: 'ready'; run: IngestionRunDetail }
  >({ kind: 'loading' })
  const { formatNumber } = useFormatters()

  // 実行は最大 50 件あるので、開いた実行だけを読む。
  useEffect(() => {
    let current = true
    getDirectoryIngestionRun(connectorID, runID).then(
      (run) => {
        if (current) setState({ kind: 'ready', run })
      },
      (cause: unknown) => {
        if (!current) return
        setState({
          kind: 'failed',
          message: cause instanceof AuthenticationAPIError ? cause.message : t.failuresLoadError,
        })
      },
    )
    return () => {
      current = false
    }
  }, [connectorID, runID, t.failuresLoadError])

  if (state.kind === 'loading') return <p className="text-sm text-slate-500">{t.failuresLoading}</p>
  if (state.kind === 'failed') return <Alert variant="destructive">{state.message}</Alert>

  const hidden = hiddenFailureCount(state.run)
  return (
    <div className="grid gap-2">
      <table className="w-full text-left text-xs">
        <thead className="font-semibold text-slate-500">
          <tr>
            <th className="py-1.5 pr-4">{t.headerExternalId}</th>
            <th className="py-1.5 pr-4">{t.headerError}</th>
            <th className="py-1.5">{t.headerBatchId}</th>
          </tr>
        </thead>
        <tbody>
          {(state.run.failures ?? []).map((failure) => (
            <tr
              key={`${failure.batch_id}/${failure.external_id}`}
              className="border-t border-slate-200 text-slate-700"
            >
              <td className="break-all py-1.5 pr-4 font-mono">{failure.external_id}</td>
              <td className="py-1.5 pr-4">{itemErrorLabel(failure.error, t)}</td>
              <td className="break-all py-1.5 font-mono">{failure.batch_id}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {hidden > 0 ? (
        <p className="text-xs text-amber-700">
          {t.failuresHidden.replace('{count}', formatNumber(hidden))}
        </p>
      ) : null}
    </div>
  )
}

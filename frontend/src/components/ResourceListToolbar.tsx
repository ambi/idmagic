import { IconRefresh } from '@tabler/icons-react'
import { useState } from 'react'
import { useDictionary, useFormatters } from '../lib/i18n'
import { commonDictionary } from '../lib/i18n/common.i18n'
import { Alert } from './ui/alert'
import { Button } from './ui/button'
import { Toast } from './ui/toast'

export type ResourceListOptions = {
  total?: number
  displayed?: number
  onRefresh: () => Promise<unknown>
  busy?: boolean
}

export function ResourceListToolbar({
  total,
  displayed,
  onRefresh,
  busy = false,
}: ResourceListOptions) {
  const t = useDictionary(commonDictionary)
  const { formatNumber } = useFormatters()
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  async function refresh() {
    setRefreshing(true)
    setError('')
    setNotice('')
    try {
      await onRefresh()
      setNotice(t.listRefreshed)
    } catch {
      setError(t.refreshFailed)
    } finally {
      setRefreshing(false)
    }
  }
  return (
    <div className="space-y-3">
      <section
        className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-slate-200 bg-white px-4 py-3"
        aria-label={t.listSummary}
      >
        <div className="flex items-baseline gap-3 text-sm">
          {total !== undefined ? (
            <span className="font-semibold text-slate-800">
              {t.totalCount.replace('{count}', formatNumber(total))}
            </span>
          ) : null}
          {displayed !== undefined ? (
            <span className="text-slate-500">
              {t.displayedCount.replace('{count}', formatNumber(displayed))}
            </span>
          ) : null}
        </div>
        <Button
          type="button"
          variant="outline"
          disabled={busy || refreshing}
          onClick={() => void refresh()}
        >
          <IconRefresh
            size={16}
            aria-hidden="true"
            className={refreshing ? 'animate-spin' : undefined}
          />
          {refreshing ? t.refreshing : t.refresh}
        </Button>
      </section>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />
    </div>
  )
}

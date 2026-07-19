import { IconRefresh } from '@tabler/icons-react'
import { useCallback, useEffect, useState } from 'react'
import {
  getAdminApplicationProvisioningDelivery,
  listAdminApplicationProvisioningDeliveries,
  retryAdminApplicationProvisioningDelivery,
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
import type { ProvisioningDelivery, ProvisioningDeliveryStatus } from '../../types'

function deliveryStatusOptions(t: ProvisioningDictionary) {
  return [
    { value: '', label: t.deliveryStatusAll },
    { value: 'pending', label: t.deliveryStatusPending },
    { value: 'in_flight', label: t.deliveryStatusInFlight },
    { value: 'succeeded', label: t.deliveryStatusSucceeded },
    { value: 'dead_letter', label: t.deliveryStatusDeadLetter },
  ]
}

function operationLabel(op: ProvisioningDelivery['operation'], t: ProvisioningDictionary): string {
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

function deliveryStatusBadge(status: ProvisioningDeliveryStatus): string {
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

export function DeliveriesPanel({
  csrfToken,
  applicationID,
}: {
  csrfToken: string
  applicationID: string
}) {
  const t = useDictionary(provisioningDictionary)
  const { locale } = useLocale()
  const [statusFilter, setStatusFilter] = useState('')
  const [deliveries, setDeliveries] = useState<ProvisioningDelivery[]>([])
  const [selected, setSelected] = useState<ProvisioningDelivery | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [retrying, setRetrying] = useState(false)

  const reload = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const status = statusFilter === '' ? undefined : (statusFilter as ProvisioningDeliveryStatus)
      const list = await listAdminApplicationProvisioningDeliveries(applicationID, status)
      setDeliveries(list)
    } catch (cause) {
      setError(messageOf(cause, t.deliveriesLoadFailedError))
    } finally {
      setLoading(false)
    }
  }, [applicationID, statusFilter, t.deliveriesLoadFailedError])

  useEffect(() => {
    void reload()
  }, [reload])

  async function selectDelivery(d: ProvisioningDelivery) {
    setSelected(d)
    try {
      setSelected(await getAdminApplicationProvisioningDelivery(applicationID, d.id))
    } catch {
      // 一覧の値のまま表示を継続する
    }
  }

  async function retry() {
    if (!selected) return
    setRetrying(true)
    setError('')
    try {
      const updated = await retryAdminApplicationProvisioningDelivery(
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
        <SectionTitle>{t.deliveriesHeading}</SectionTitle>
        <div className="flex items-center gap-2">
          <Select
            value={statusFilter}
            onValueChange={setStatusFilter}
            options={deliveryStatusOptions(t)}
            aria-label={t.deliveryStatusFilterLabel}
          />
          <Button
            type="button"
            variant="ghost"
            aria-label={t.deliveriesReloadAria}
            onClick={() => void reload()}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
        </div>
      </div>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <DeliveriesTable
          deliveries={deliveries}
          loading={loading}
          selected={selected}
          onSelect={(d) => void selectDelivery(d)}
          locale={locale}
          t={t}
        />
        <DeliveryDetailCard
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

function DeliveriesTable({
  deliveries,
  loading,
  selected,
  onSelect,
  locale,
  t,
}: {
  deliveries: ProvisioningDelivery[]
  loading: boolean
  selected: ProvisioningDelivery | null
  onSelect: (d: ProvisioningDelivery) => void
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
          {!loading && deliveries.length === 0 ? (
            <tr>
              <td colSpan={4} className="px-3 py-8 text-center text-xs text-slate-500">
                {t.deliveriesEmptyNotice}
              </td>
            </tr>
          ) : null}
          {deliveries.map((d) => (
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
                  className={`rounded px-2 py-0.5 text-xs font-medium ${deliveryStatusBadge(d.status)}`}
                >
                  {deliveryStatusOptions(t).find((o) => o.value === d.status)?.label ?? d.status}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function DeliveryDetailCard({
  selected,
  retrying,
  onRetry,
  locale,
  t,
}: {
  selected: ProvisioningDelivery | null
  retrying: boolean
  onRetry: () => void
  locale: string
  t: ProvisioningDictionary
}) {
  return (
    <Card className="p-4">
      <h3 className="text-sm font-semibold text-slate-700">{t.deliveryDetailHeading}</h3>
      {selected ? (
        <dl className="mt-3 grid grid-cols-[110px_minmax(0,1fr)] gap-y-2 text-xs">
          <dt className="text-slate-500">{t.deliveryIdLabel}</dt>
          <dd className="break-all font-mono">{selected.id}</dd>
          <dt className="text-slate-500">{t.tableHeaderStatus}</dt>
          <dd>{deliveryStatusOptions(t).find((o) => o.value === selected.status)?.label}</dd>
          <dt className="text-slate-500">{t.deliverySourceVersionLabel}</dt>
          <dd className="font-mono">{selected.source_version}</dd>
          {selected.job_id ? (
            <>
              <dt className="text-slate-500">{t.deliveryJobIdLabel}</dt>
              <dd className="break-all font-mono">{selected.job_id}</dd>
            </>
          ) : null}
          {selected.last_error ? (
            <>
              <dt className="text-slate-500">{t.deliveryLastErrorLabel}</dt>
              <dd className="break-all text-red-700">{selected.last_error}</dd>
            </>
          ) : null}
          {selected.completed_at ? (
            <>
              <dt className="text-slate-500">{t.deliveryCompletedAtLabel}</dt>
              <dd>{formatDate(selected.completed_at, locale, t.unknownDate)}</dd>
            </>
          ) : null}
        </dl>
      ) : (
        <p className="mt-3 text-xs text-slate-500">{t.selectDeliveryPrompt}</p>
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

import { IconDownload, IconPlus, IconRefresh, IconSearch, IconTrash } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  type AdminAuditEventQuery,
  adminAuditEventsExportURL,
  AuthenticationAPIError,
  listAdminAuditEvents,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminAuditEvent } from '../../types'
import {
  adminAuditEventsDictionary,
  type AdminAuditEventsDictionary,
} from './AdminAuditEventsPage.i18n'

const DEFAULT_REALM = 'default'

type EventKind = 'success' | 'fail' | 'aggregated'
type AuditFilterField =
  | 'actor.username'
  | 'client.ip'
  | 'event.type'
  | 'outcome'
  | 'target.id'
  | 'session.id'
  | 'transaction.id'

type AuditFilterRow = {
  id: number
  field: AuditFilterField
  value: string
}

function auditFilterFields(
  t: AdminAuditEventsDictionary,
): Array<{ value: AuditFilterField; label: string; placeholder: string }> {
  return [
    {
      value: 'actor.username',
      label: t.filterFieldUsername,
      placeholder: t.filterFieldUsernamePlaceholder,
    },
    { value: 'client.ip', label: t.filterFieldIp, placeholder: t.filterFieldIpPlaceholder },
    {
      value: 'event.type',
      label: t.filterFieldEventType,
      placeholder: t.filterFieldEventTypePlaceholder,
    },
    { value: 'outcome', label: t.filterFieldOutcome, placeholder: t.filterFieldOutcomePlaceholder },
    {
      value: 'target.id',
      label: t.filterFieldTargetUser,
      placeholder: t.filterFieldTargetUserPlaceholder,
    },
    {
      value: 'session.id',
      label: t.filterFieldSession,
      placeholder: t.filterFieldSessionPlaceholder,
    },
    {
      value: 'transaction.id',
      label: t.filterFieldTransaction,
      placeholder: t.filterFieldTransactionPlaceholder,
    },
  ]
}

const FAIL_TYPES = new Set([
  'AuthenticationFailed',
  'AuthenticationStepFailed',
  'MfaChallengeFailed',
])
const AGGREGATED_TYPES = new Set(['AuthenticationEventAggregated', 'LoginThrottled'])
const AUTH_TYPES = new Set([
  'UserAuthenticated',
  'AuthenticationStepCompleted',
  'MfaChallengeIssued',
  'MfaChallengeSucceeded',
  'BackupCodeConsumed',
  'SessionStarted',
  'SessionRefreshed',
  'SessionEnded',
  'FederatedAuthenticated',
  'FederationLinked',
  'FederationUnlinked',
  'SessionImpersonationStarted',
  'SessionImpersonationEnded',
  ...FAIL_TYPES,
  ...AGGREGATED_TYPES,
])

function authEventKind(type: string): EventKind | null {
  if (!AUTH_TYPES.has(type)) return null
  if (FAIL_TYPES.has(type)) return 'fail'
  if (AGGREGATED_TYPES.has(type)) return 'aggregated'
  return 'success'
}

const KIND_BADGE: Record<EventKind, string> = {
  success: 'bg-emerald-50 text-emerald-700',
  fail: 'bg-rose-50 text-rose-700',
  aggregated: 'bg-amber-50 text-amber-700',
}

function kindLabel(kind: EventKind, t: AdminAuditEventsDictionary): string {
  return { success: t.kindSuccess, fail: t.kindFail, aggregated: t.kindAggregated }[kind]
}

export function AdminAuditEventsPage({
  actorUsername,
  actorRoles,
  actorRealm,
  events: initial,
}: {
  actorUsername?: string
  actorRoles: string[]
  actorRealm: string
  events: AdminAuditEvent[]
}) {
  const [events, setEvents] = useState(initial)
  const [selected, setSelected] = useState<AdminAuditEvent | null>(initial[0] ?? null)
  const [category, setCategory] = useState<'' | AdminAuditEventQuery['category']>('')
  const [sub, setSub] = useState('')
  const [after, setAfter] = useState('')
  const [before, setBefore] = useState('')
  const [limit, setLimit] = useState('100')
  const [filters, setFilters] = useState<AuditFilterRow[]>([
    { id: 1, field: 'actor.username', value: '' },
  ])
  const [allTenants, setAllTenants] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminAuditEventsDictionary)
  const { locale } = useLocale()

  const canCrossTenant = actorRoles.includes('system_admin') && actorRealm === DEFAULT_REALM

  function buildQuery(): AdminAuditEventQuery {
    const parsedLimit = limit.trim() ? Number.parseInt(limit, 10) : undefined
    return {
      category: category || undefined,
      sub: sub.trim() || undefined,
      after: after.trim() ? new Date(after).toISOString() : undefined,
      before: before.trim() ? new Date(before).toISOString() : undefined,
      limit: Number.isFinite(parsedLimit) ? parsedLimit : undefined,
      allTenants: canCrossTenant && allTenants,
      filter: filters
        .map((filter) => {
          const value = filter.value.trim()
          return value ? `${filter.field}:eq:${value}` : ''
        })
        .filter(Boolean),
    }
  }

  function addFilter() {
    setFilters((current) => [...current, { id: Date.now(), field: 'actor.username', value: '' }])
  }

  function updateFilter(id: number, patch: Partial<Omit<AuditFilterRow, 'id'>>) {
    setFilters((current) => current.map((row) => (row.id === id ? { ...row, ...patch } : row)))
  }

  function removeFilter(id: number) {
    setFilters((current) =>
      current.length === 1
        ? [{ ...current[0], value: '' }]
        : current.filter((row) => row.id !== id),
    )
  }

  async function handleQuery(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const next = await listAdminAuditEvents(buildQuery())
      setEvents(next)
      setSelected(next[0] ?? null)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : t.auditEventsFetchFailedError,
      )
    } finally {
      setBusy(false)
    }
  }

  function handleExport() {
    window.open(adminAuditEventsExportURL(buildQuery()), '_blank')
  }

  return (
    <AdminShell
      active="audit-events"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      <Card className="p-5">
        <form onSubmit={handleQuery} className="grid gap-4 lg:grid-cols-3">
          <Field label={t.eventCategoryFieldLabel}>
            <select
              value={category ?? ''}
              onChange={(e) =>
                setCategory((e.target.value || '') as '' | AdminAuditEventQuery['category'])
              }
              className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
            >
              <option value="">{t.allEventsOption}</option>
              <optgroup label={t.authenticationGroupLabel}>
                <option value="authentication">{t.authenticationAllOption}</option>
                <option value="success">{t.kindSuccess}</option>
                <option value="fail">{t.kindFail}</option>
                <option value="aggregated">{t.authAggregatedOption}</option>
              </optgroup>
              <optgroup label={t.adminOperationsGroupLabel}>
                <option value="user">{t.userManagementOption}</option>
                <option value="group">{t.groupManagementOption}</option>
                <option value="client">{t.clientManagementOption}</option>
                <option value="consent">{t.consentOption}</option>
                <option value="token">{t.tokenFlowOption}</option>
                <option value="tenant">{t.tenantManagementOption}</option>
                <option value="key">{t.signingKeyOption}</option>
              </optgroup>
            </select>
          </Field>
          <Field label={t.targetUserSubFieldLabel}>
            <Input
              value={sub}
              onChange={(e) => setSub(e.target.value)}
              placeholder={t.filterFieldTargetUserPlaceholder}
            />
          </Field>
          <Field label={t.startDateFieldLabel}>
            <Input type="datetime-local" value={after} onChange={(e) => setAfter(e.target.value)} />
          </Field>
          <Field label={t.endDateFieldLabel}>
            <Input
              type="datetime-local"
              value={before}
              onChange={(e) => setBefore(e.target.value)}
            />
          </Field>
          <Field label={t.maxCountFieldLabel}>
            <Input
              type="number"
              min={1}
              max={1000}
              value={limit}
              onChange={(e) => setLimit(e.target.value)}
            />
          </Field>
          <div className="grid gap-3 lg:col-span-3">
            <div className="flex items-center justify-between gap-3">
              <Label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                {t.searchAttributesLabel}
              </Label>
              <Button type="button" variant="ghost" onClick={addFilter} disabled={busy}>
                <IconPlus size={16} aria-hidden="true" />
                {t.addCondition}
              </Button>
            </div>
            <div className="grid gap-2">
              {filters.map((filter) => {
                const fields = auditFilterFields(t)
                const selectedField =
                  fields.find((field) => field.value === filter.field) ?? fields[0]
                return (
                  <div
                    key={filter.id}
                    className="grid gap-2 rounded-md border border-slate-200 p-2 sm:grid-cols-[190px_minmax(0,1fr)_40px]"
                  >
                    <select
                      value={filter.field}
                      onChange={(e) =>
                        updateFilter(filter.id, {
                          field: e.target.value as AuditFilterField,
                        })
                      }
                      className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
                    >
                      {fields.map((field) => (
                        <option key={field.value} value={field.value}>
                          {field.label}
                        </option>
                      ))}
                    </select>
                    <Input
                      value={filter.value}
                      onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                      placeholder={selectedField.placeholder}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      className="h-9 w-9 px-0"
                      onClick={() => removeFilter(filter.id)}
                      disabled={busy}
                      aria-label={t.removeConditionAria}
                    >
                      <IconTrash size={16} aria-hidden="true" />
                    </Button>
                  </div>
                )
              })}
            </div>
          </div>
          <div className="flex items-end gap-2 lg:col-span-3">
            <Button type="submit" disabled={busy}>
              <IconSearch size={16} aria-hidden="true" />
              {t.filterAction}
            </Button>
            <Button type="button" variant="ghost" onClick={handleExport} disabled={busy}>
              <IconDownload size={16} aria-hidden="true" />
              {t.exportAction}
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
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-3">{t.tableHeaderOccurredAt}</th>
                <th className="px-4 py-3">{t.tableHeaderType}</th>
                <th className="px-4 py-3">{t.tableHeaderTenant}</th>
              </tr>
            </thead>
            <tbody>
              {events.length === 0 ? (
                <tr>
                  <td colSpan={3} className="px-4 py-12 text-center text-sm text-slate-500">
                    {t.noMatchingEventsNotice}
                  </td>
                </tr>
              ) : null}
              {events.map((e) => (
                <tr
                  key={e.id}
                  onClick={() => setSelected(e)}
                  className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${
                    selected?.id === e.id ? 'bg-blue-50/60' : ''
                  }`}
                >
                  <td className="px-4 py-3 font-mono text-xs">
                    {formatDate(e.occurred_at, locale)}
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-2">
                      {authEventKind(e.type) ? (
                        <span
                          className={`rounded px-2 py-0.5 text-xs font-medium ${KIND_BADGE[authEventKind(e.type) as EventKind]}`}
                        >
                          {kindLabel(authEventKind(e.type) as EventKind, t)}
                        </span>
                      ) : null}
                      {e.type}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">{e.tenant_id}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>

        <Card className="p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-slate-700">{t.payloadHeading}</h2>
            {selected ? (
              <Button
                variant="ghost"
                onClick={() =>
                  navigator.clipboard?.writeText(JSON.stringify(selected.payload, null, 2))
                }
                aria-label={t.copyPayloadAria}
              >
                <IconRefresh size={14} aria-hidden="true" />
                {t.copy}
              </Button>
            ) : null}
          </div>
          {selected ? (
            <>
              <dl className="mt-4 grid grid-cols-[80px_minmax(0,1fr)] gap-y-2 text-xs">
                <dt className="text-slate-500">{t.idLabel}</dt>
                <dd className="break-all font-mono">{selected.id}</dd>
                <dt className="text-slate-500">{t.tableHeaderType}</dt>
                <dd>{selected.type}</dd>
                <dt className="text-slate-500">{t.tableHeaderTenant}</dt>
                <dd className="font-mono">{selected.tenant_id}</dd>
                <dt className="text-slate-500">{t.dateTimeLabel}</dt>
                <dd>{formatDate(selected.occurred_at, locale)}</dd>
              </dl>
              <pre className="mt-4 max-h-[420px] overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-50">
                {JSON.stringify(selected.payload, null, 2)}
              </pre>
            </>
          ) : (
            <p className="mt-4 text-sm text-slate-500">{t.selectEventPrompt}</p>
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

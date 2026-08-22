import { IconDownload, IconPlus, IconRefresh, IconSearch, IconTrash } from '@tabler/icons-react'
import { type FormEvent, useEffect, useState } from 'react'
import {
  type AdminAuditEventCategory,
  type AdminAuditEventSearchOptions,
  type AdminAuditEventsSearchParams,
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
import { PageNavigation, type PageNavigationData } from '../../components/ui/page-navigation'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import type { AdminAuditEvent } from '../../types'
import { friendlyEventName } from '../admin-dashboard/AdminDashboardPage.i18n'
import {
  adminAuditEventsDictionary,
  type AdminAuditEventsDictionary,
} from './AdminAuditEventsPage.i18n'

const DEFAULT_REALM = 'default'

// wi-147: 「誰を検索するか」を含むすべての検索条件を1つの一覧に統一する。quick.* は
// registry の filter attribute ではなく、トップレベル query param (category/sub/username) へ
// 変換される疑似フィールド。それ以外は既存の registry allowlist の filter[] へ変換される。
type AuditFilterField =
  | 'quick.category'
  | 'quick.actor_id'
  | 'quick.username'
  | 'actor.username'
  | 'client.ip'
  | 'event.type'
  | 'outcome'
  | 'target.id'
  | 'session.id'
  | 'actor.type'
  | 'agent.id'
  | 'delegation.actor'
  | 'delegation.depth'
  | 'delegation.mode'

// wi-377: 委譲の軸。エージェントが利用者を代行した操作を、利用者本人の操作と区別して引く。
const DELEGATION_FIELDS: AuditFilterField[] = [
  'actor.type',
  'agent.id',
  'delegation.actor',
  'delegation.depth',
  'delegation.mode',
]

type AuditFilterRow = {
  id: number
  field: AuditFilterField
  value: string
}

function auditFilterFields(
  t: AdminAuditEventsDictionary,
): Array<{ value: AuditFilterField; label: string; placeholder?: string }> {
  return [
    { value: 'quick.category', label: t.eventCategoryFieldLabel },
    {
      value: 'quick.actor_id',
      label: t.actorUserIdFieldLabel,
      placeholder: t.actorUserIdFieldPlaceholder,
    },
    {
      value: 'quick.username',
      label: t.actorUsernameFieldLabel,
      placeholder: t.actorUsernameFieldPlaceholder,
    },
    {
      value: 'actor.username',
      label: t.filterFieldUsername,
      placeholder: t.filterFieldUsernamePlaceholder,
    },
    { value: 'client.ip', label: t.filterFieldIp, placeholder: t.filterFieldIpPlaceholder },
    { value: 'event.type', label: t.filterFieldEventType },
    { value: 'outcome', label: t.filterFieldOutcome },
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
    { value: 'actor.type', label: t.filterFieldActorType },
    {
      value: 'agent.id',
      label: t.filterFieldAgentId,
      placeholder: t.filterFieldAgentIdPlaceholder,
    },
    {
      value: 'delegation.actor',
      label: t.filterFieldDelegationActor,
      placeholder: t.filterFieldDelegationActorPlaceholder,
    },
    {
      value: 'delegation.depth',
      label: t.filterFieldDelegationDepth,
      placeholder: t.filterFieldDelegationDepthPlaceholder,
    },
    { value: 'delegation.mode', label: t.filterFieldDelegationMode },
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

type EventKind = 'success' | 'fail' | 'aggregated'

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

const DEFAULT_OUTCOME_CHOICES = ['success', 'failure']
const DEFAULT_ACTOR_TYPE_CHOICES = ['user', 'agent']
const DEFAULT_DELEGATION_MODE_CHOICES = ['direct', 'autonomous', 'on_behalf_of']

function outcomeLabel(value: string, t: AdminAuditEventsDictionary): string {
  if (value === 'success') return t.outcomeSuccessOption
  if (value === 'failure') return t.outcomeFailureOption
  return value
}

function actorTypeLabel(value: string, t: AdminAuditEventsDictionary): string {
  if (value === 'user') return t.actorTypeUserOption
  if (value === 'agent') return t.actorTypeAgentOption
  return value
}

function delegationModeLabel(value: string, t: AdminAuditEventsDictionary): string {
  if (value === 'direct') return t.delegationModeDirectOption
  if (value === 'autonomous') return t.delegationModeAutonomousOption
  if (value === 'on_behalf_of') return t.delegationModeOnBehalfOfOption
  return value
}

// 監査イベントの payload から委譲チェーンを読み出す。チェーンを持たないイベント
// (この軸が追加される前の記録を含む) では undefined を返す。
type DelegationChain = {
  participants: string[]
  depth?: number
  mode?: string
  agentId?: string
}

function delegationChainOf(event: AdminAuditEvent | null): DelegationChain | undefined {
  const payload = event?.payload as Record<string, unknown> | undefined
  if (!payload) return undefined
  const raw = payload.actorChain
  const participants = Array.isArray(raw)
    ? raw.filter((v): v is string => typeof v === 'string')
    : []
  const depth = typeof payload.delegationDepth === 'number' ? payload.delegationDepth : undefined
  const mode = typeof payload.delegationMode === 'string' ? payload.delegationMode : undefined
  const agentId = typeof payload.agentId === 'string' ? payload.agentId : undefined
  if (participants.length === 0 && depth === undefined && mode === undefined) return undefined
  return { participants, depth, mode, agentId }
}

// datetime-local 入力は timezone を持たないローカル時刻表記を要求する。API 用 ISO 文字列
// (buildQuery が new Date(after).toISOString() で生成) を、URL 経由で受け取った検索条件から
// フォームへ復元する際はローカル時刻へ戻す (wi-147)。
function isoToDatetimeLocal(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// URL query (category / sub / username / filter[]) を検索条件一覧の行へ復元する (wi-147)。
// トップレベル query param もすべて同じ一覧の行として表示し、UI 上の置き場所を1つに統一する。
function filtersFromSearch(search?: AdminAuditEventsSearchParams): AuditFilterRow[] {
  const rows: AuditFilterRow[] = []
  let nextID = 1
  if (search?.category) rows.push({ id: nextID++, field: 'quick.category', value: search.category })
  if (search?.sub) rows.push({ id: nextID++, field: 'quick.actor_id', value: search.sub })
  if (search?.username) rows.push({ id: nextID++, field: 'quick.username', value: search.username })
  for (const raw of search?.filter ?? []) {
    const firstColon = raw.indexOf(':')
    const secondColon = raw.indexOf(':', firstColon + 1)
    const field = (firstColon >= 0 ? raw.slice(0, firstColon) : raw) as AuditFilterField
    const value = secondColon >= 0 ? raw.slice(secondColon + 1) : ''
    rows.push({ id: nextID++, field, value })
  }
  if (rows.length === 0) rows.push({ id: 1, field: 'quick.category', value: '' })
  return rows
}

export function AdminAuditEventsPage({
  actorUsername,
  actorRoles,
  actorRealm,
  events: initial,
  pagination,
  previousCursor = null,
  nextCursor: initialNextCursor,
  search,
  searchOptions,
  onSearch,
  onPage,
  cursorReset = false,
  initialError,
}: {
  actorUsername?: string
  actorRoles: string[]
  actorRealm: string
  events: AdminAuditEvent[]
  pagination?: Omit<PageNavigationData, 'nextCursor' | 'previousCursor'>
  previousCursor?: string | null
  nextCursor: string | null
  search?: AdminAuditEventsSearchParams
  searchOptions?: AdminAuditEventSearchOptions
  onSearch?: (search: AdminAuditEventsSearchParams) => void
  onPage?: (cursor: string | null) => void
  cursorReset?: boolean
  // 初期表示 (loader) 側の取得失敗。URL の検索条件が壊れていてもページ自体は表示し、
  // ページ内のエラー表示に留める (wi-147)。
  initialError?: string
}) {
  const [events, setEvents] = useState(initial)
  const [selected, setSelected] = useState<AdminAuditEvent | null>(initial[0] ?? null)
  const [after, setAfter] = useState(isoToDatetimeLocal(search?.after))
  const [before, setBefore] = useState(isoToDatetimeLocal(search?.before))
  const [limit, setLimit] = useState(search?.limit !== undefined ? String(search.limit) : '100')
  const [filters, setFilters] = useState<AuditFilterRow[]>(filtersFromSearch(search))
  const [allTenants, setAllTenants] = useState(search?.allTenants ?? false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(initialError ?? '')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminAuditEventsDictionary)
  const tCommon = useDictionary(commonDictionary)
  const { locale } = useLocale()

  useEffect(() => {
    if (cursorReset) setNotice(tCommon.cursorResetNotice)
  }, [cursorReset, tCommon.cursorResetNotice])

  const canCrossTenant = actorRoles.includes('system_admin') && actorRealm === DEFAULT_REALM
  const eventTypeChoices = searchOptions?.event_types ?? []
  const outcomeChoices = searchOptions?.outcomes ?? DEFAULT_OUTCOME_CHOICES
  const actorTypeChoices = searchOptions?.actor_types ?? DEFAULT_ACTOR_TYPE_CHOICES
  const delegationModeChoices = searchOptions?.delegation_modes ?? DEFAULT_DELEGATION_MODE_CHOICES
  const delegationSelected = filters.some((row) => DELEGATION_FIELDS.includes(row.field))
  const delegationChain = delegationChainOf(selected)

  function buildQuery(): AdminAuditEventsSearchParams {
    const parsedLimit = limit.trim() ? Number.parseInt(limit, 10) : undefined
    const query: AdminAuditEventsSearchParams = {
      after: after.trim() ? new Date(after).toISOString() : undefined,
      before: before.trim() ? new Date(before).toISOString() : undefined,
      limit: Number.isFinite(parsedLimit) ? parsedLimit : undefined,
      allTenants: canCrossTenant && allTenants,
    }
    const filterExprs: string[] = []
    for (const row of filters) {
      const value = row.value.trim()
      if (!value) continue
      switch (row.field) {
        case 'quick.category':
          query.category = value as AdminAuditEventCategory
          break
        case 'quick.actor_id':
          query.sub = value
          break
        case 'quick.username':
          query.username = value
          break
        default:
          filterExprs.push(`${row.field}:eq:${value}`)
      }
    }
    query.filter = filterExprs
    return query
  }

  function addFilter() {
    setFilters((current) => [...current, { id: Date.now(), field: 'event.type', value: '' }])
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
    const query = buildQuery()
    onSearch?.(query)
    try {
      const next = await listAdminAuditEvents(query)
      setEvents(next.events)
      setSelected(next.events[0] ?? null)
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
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <Card className="p-5">
        <form onSubmit={handleQuery} className="grid gap-4 lg:grid-cols-3">
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
            <p className="text-xs text-slate-500">{t.searchAttributesHint}</p>
            {delegationSelected ? (
              <p className="text-xs text-slate-500">{t.delegationAxesHint}</p>
            ) : null}
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
                          value: '',
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
                    {filter.field === 'quick.category' ? (
                      <select
                        value={filter.value}
                        onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
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
                    ) : filter.field === 'event.type' ? (
                      <select
                        value={filter.value}
                        onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                        className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
                      >
                        <option value="">{t.filterFieldAnyOption}</option>
                        {eventTypeChoices.map((choice) => (
                          <option key={choice} value={choice}>
                            {choice}
                          </option>
                        ))}
                      </select>
                    ) : filter.field === 'outcome' ? (
                      <select
                        value={filter.value}
                        onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                        className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
                      >
                        <option value="">{t.filterFieldAnyOption}</option>
                        {outcomeChoices.map((choice) => (
                          <option key={choice} value={choice}>
                            {outcomeLabel(choice, t)}
                          </option>
                        ))}
                      </select>
                    ) : filter.field === 'actor.type' ? (
                      <select
                        value={filter.value}
                        onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                        className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
                      >
                        <option value="">{t.filterFieldAnyOption}</option>
                        {actorTypeChoices.map((choice) => (
                          <option key={choice} value={choice}>
                            {actorTypeLabel(choice, t)}
                          </option>
                        ))}
                      </select>
                    ) : filter.field === 'delegation.mode' ? (
                      <select
                        value={filter.value}
                        onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                        className="h-9 rounded-md border border-slate-300 bg-white px-3 text-sm"
                      >
                        <option value="">{t.filterFieldAnyOption}</option>
                        {delegationModeChoices.map((choice) => (
                          <option key={choice} value={choice}>
                            {delegationModeLabel(choice, t)}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <Input
                        className="h-9"
                        value={filter.value}
                        onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                        placeholder={selectedField.placeholder}
                      />
                    )}
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
                      {friendlyEventName(e.type, locale)}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">{e.tenant_id}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {onPage ? (
            <PageNavigation
              {...pagination}
              previousCursor={previousCursor}
              nextCursor={initialNextCursor}
              onNavigate={onPage}
            />
          ) : null}
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
                <dd>{friendlyEventName(selected.type, locale)}</dd>
                <dt className="text-slate-500">{t.tableHeaderTenant}</dt>
                <dd className="font-mono">{selected.tenant_id}</dd>
                <dt className="text-slate-500">{t.dateTimeLabel}</dt>
                <dd>{formatDate(selected.occurred_at, locale)}</dd>
              </dl>
              {delegationChain ? (
                <DelegationChainView
                  chain={delegationChain}
                  t={t}
                  onSearchParticipant={(participant) =>
                    setFilters([{ id: Date.now(), field: 'delegation.actor', value: participant }])
                  }
                />
              ) : null}
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

// wi-377: 委譲チェーンを展開して表示する。参加者は外側 (現在の行為者) から内側へ並び、
// 最後が代行された主体である。参加者をクリックすると、その参加者がチェーンのどの段に
// いても引ける delegation.actor での絞り込みに切り替える。
function DelegationChainView({
  chain,
  t,
  onSearchParticipant,
}: {
  chain: DelegationChain
  t: AdminAuditEventsDictionary
  onSearchParticipant: (participant: string) => void
}) {
  return (
    <section className="mt-4 rounded-md border border-slate-200 p-3">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
        {t.delegationChainHeading}
      </h3>
      <dl className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-600">
        {chain.mode ? (
          <div className="flex gap-1">
            <dt className="text-slate-500">{t.delegationChainModeLabel}</dt>
            <dd>{delegationModeLabel(chain.mode, t)}</dd>
          </div>
        ) : null}
        {chain.depth !== undefined ? (
          <div className="flex gap-1">
            <dt className="text-slate-500">{t.delegationChainDepthLabel}</dt>
            <dd className="font-mono">{chain.depth}</dd>
          </div>
        ) : null}
        {chain.agentId ? (
          <div className="flex gap-1">
            <dt className="text-slate-500">{t.delegationChainAgentLabel}</dt>
            <dd className="break-all font-mono">{chain.agentId}</dd>
          </div>
        ) : null}
      </dl>
      {chain.participants.length === 0 ? (
        <p className="mt-2 text-xs text-slate-500">{t.delegationChainAbsentNotice}</p>
      ) : (
        <ol className="mt-2 flex flex-wrap items-center gap-1 text-xs">
          {chain.participants.map((participant, index) => (
            <li key={participant} className="flex items-center gap-1">
              {index > 0 ? (
                <span className="text-slate-400">{t.delegationChainActingFor}</span>
              ) : null}
              <button
                type="button"
                onClick={() => onSearchParticipant(participant)}
                aria-label={`${t.delegationChainSearchAria}: ${participant}`}
                className="break-all rounded bg-slate-100 px-2 py-0.5 font-mono hover:bg-slate-200"
              >
                {participant}
              </button>
            </li>
          ))}
        </ol>
      )}
    </section>
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

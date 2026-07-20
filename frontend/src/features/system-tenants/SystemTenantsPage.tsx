import { IconBan, IconCheck, IconPlus, IconRefresh } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  createAdminTenant,
  listAdminTenants,
  setAdminTenantDisabled,
  updateAdminTenant,
  updateAdminTenantQuota,
} from '../../api'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'

export function SystemTenantsPage({
  csrfToken,
  actorUsername,
  tenants: initial,
}: {
  csrfToken: string
  actorUsername?: string
  tenants: AdminTenant[]
}) {
  const [tenants, setTenants] = useState(initial)
  const [selected, setSelected] = useState<AdminTenant | null>(initial[0] ?? null)
  const [showCreate, setShowCreate] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(systemTenantsDictionary)

  async function refresh(preferredID?: string) {
    const next = await listAdminTenants()
    setTenants(next)
    const match = preferredID ? next.find((tenant) => tenant.id === preferredID) : null
    setSelected(match ?? next[0] ?? null)
  }

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(success)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.tenantActionFailedError)
    } finally {
      setBusy(false)
    }
  }

  async function handleCreate(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = e.currentTarget
    const data = new FormData(form)
    await run(async () => {
      const created = await createAdminTenant(csrfToken, {
        realm: String(data.get('realm') ?? ''),
        display_name: String(data.get('display_name') ?? ''),
      })
      form.reset()
      setShowCreate(false)
      await refresh(created.id)
    }, t.tenantCreatedNotice)
  }

  async function handleToggleDisabled(target: AdminTenant) {
    const disabled = target.status === 'active'
    await run(
      async () => {
        await setAdminTenantDisabled(csrfToken, target.realm, disabled)
        await refresh(target.id)
      },
      disabled ? t.tenantDisabledNotice : t.tenantReenabledNotice,
    )
  }

  return (
    <SystemShell
      active="tenants"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <>
          <Button
            variant="outline"
            className="size-9 px-0"
            aria-label={t.reloadListAriaLabel}
            onClick={() => run(() => refresh(selected?.id), t.listRefreshedNotice)}
            disabled={busy}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
          <Button onClick={() => setShowCreate(true)} disabled={busy}>
            <IconPlus size={16} aria-hidden="true" />
            {t.newTenant}
          </Button>
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_420px]">
        <TenantTable
          tenants={tenants}
          selectedID={selected?.id}
          busy={busy}
          onSelect={setSelected}
          onToggleDisabled={handleToggleDisabled}
        />

        <TenantDetailCard
          tenant={selected}
          csrfToken={csrfToken}
          busy={busy}
          onSaved={(id) => run(() => refresh(id), t.tenantUpdatedNotice)}
        />
      </div>

      {showCreate ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 px-4">
          <Card className="w-full max-w-md p-6">
            <h2 className="text-base font-semibold text-slate-900">{t.newTenant}</h2>
            <form onSubmit={handleCreate} className="mt-4 grid gap-4">
              <div className="grid gap-1.5">
                <Label htmlFor="tenant-realm">{t.realmLabel}</Label>
                <Input
                  id="tenant-realm"
                  name="realm"
                  required
                  pattern="^[a-z0-9][a-z0-9-]{0,62}$"
                  placeholder="acme"
                />
                <p className="text-xs text-slate-500">{t.realmHelp}</p>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="tenant-display">{t.displayNameLabel}</Label>
                <Input id="tenant-display" name="display_name" required placeholder="Acme Inc." />
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setShowCreate(false)}
                  disabled={busy}
                >
                  {t.cancel}
                </Button>
                <Button type="submit" disabled={busy}>
                  {t.create}
                </Button>
              </div>
            </form>
          </Card>
        </div>
      ) : null}
    </SystemShell>
  )
}

export function TenantTable({
  tenants,
  selectedID,
  busy,
  onSelect,
  onToggleDisabled,
}: {
  tenants: AdminTenant[]
  selectedID?: string
  busy: boolean
  onSelect: (tenant: AdminTenant) => void
  onToggleDisabled: (tenant: AdminTenant) => void
}) {
  const t = useDictionary(systemTenantsDictionary)
  return (
    <Card className="overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3">Realm</th>
            <th className="px-4 py-3">{t.tableHeaderDisplayName}</th>
            <th className="px-4 py-3">{t.tableHeaderStatus}</th>
            <th className="px-4 py-3" />
          </tr>
        </thead>
        <tbody>
          {tenants.map((tenant) => (
            <tr
              key={tenant.id}
              onClick={() => onSelect(tenant)}
              className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${selectedID === tenant.id ? 'bg-blue-50/60' : ''}`}
            >
              <td className="px-4 py-3 font-mono text-xs">{tenant.realm}</td>
              <td className="px-4 py-3">{tenant.display_name}</td>
              <td className="px-4 py-3">
                <StatusBadge status={tenant.status} />
              </td>
              <td className="px-4 py-3 text-right">
                {tenant.realm !== 'default' ? (
                  <Button
                    variant="ghost"
                    className={
                      tenant.status === 'active'
                        ? 'text-rose-700 hover:bg-rose-50'
                        : 'text-emerald-700 hover:bg-emerald-50'
                    }
                    disabled={busy}
                    onClick={(event) => {
                      event.stopPropagation()
                      onToggleDisabled(tenant)
                    }}
                  >
                    {tenant.status === 'active' ? (
                      <IconBan size={14} aria-hidden="true" />
                    ) : (
                      <IconCheck size={14} aria-hidden="true" />
                    )}
                    {tenant.status === 'active' ? t.disable : t.enable}
                  </Button>
                ) : null}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )
}

export function TenantDetailCard({
  tenant,
  csrfToken,
  busy,
  onSaved,
}: {
  tenant: AdminTenant | null
  csrfToken: string
  busy: boolean
  onSaved: (id: string) => void
}) {
  const t = useDictionary(systemTenantsDictionary)
  const { locale } = useLocale()
  if (!tenant) {
    return (
      <Card className="p-5">
        <p className="text-sm text-slate-500">{t.selectTenantPrompt}</p>
      </Card>
    )
  }
  return (
    <Card className="p-5">
      <h2 className="text-sm font-semibold text-slate-700">{t.detailsHeading}</h2>
      <dl className="mt-4 grid grid-cols-[110px_minmax(0,1fr)] gap-y-3 text-sm">
        <dt className="text-slate-500">Realm</dt>
        <dd className="font-mono text-xs">{tenant.realm}</dd>
        <dt className="text-slate-500">{t.idLabel}</dt>
        <dd className="font-mono text-xs">{tenant.id}</dd>
        <dt className="text-slate-500">{t.displayNameLabel}</dt>
        <dd>{tenant.display_name}</dd>
        <dt className="text-slate-500">{t.tableHeaderStatus}</dt>
        <dd>
          <StatusBadge status={tenant.status} />
        </dd>
        <dt className="text-slate-500">{t.createdLabel}</dt>
        <dd>{formatDate(tenant.created_at, locale)}</dd>
        {tenant.disabled_at ? (
          <>
            <dt className="text-slate-500">{t.disabledLabel}</dt>
            <dd>{formatDate(tenant.disabled_at, locale)}</dd>
          </>
        ) : null}
        {tenant.usage ? (
          <>
            <dt className="text-slate-500 mt-2">{t.quotaHeading}</dt>
            <dd className="col-span-2 mt-2">
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  Users: {tenant.usage?.users ?? 0}{' '}
                  {tenant.quota?.users ? `/ ${tenant.quota.users}` : ''}
                </div>
                <div>
                  Groups: {tenant.usage?.groups ?? 0}{' '}
                  {tenant.quota?.groups ? `/ ${tenant.quota.groups}` : ''}
                </div>
                <div>
                  Apps: {tenant.usage?.applications ?? 0}{' '}
                  {tenant.quota?.applications ? `/ ${tenant.quota.applications}` : ''}
                </div>
                <div>
                  Clients: {tenant.usage?.oauth2_clients ?? 0}{' '}
                  {tenant.quota?.oauth2_clients ? `/ ${tenant.quota.oauth2_clients}` : ''}
                </div>
              </div>
            </dd>
          </>
        ) : null}
      </dl>
      <TenantEditor tenant={tenant} csrfToken={csrfToken} busy={busy} onSaved={onSaved} />
    </Card>
  )
}

function TenantEditor({
  tenant,
  csrfToken,
  busy,
  onSaved,
}: {
  tenant: AdminTenant
  csrfToken: string
  busy: boolean
  onSaved: (id: string) => void
}) {
  const [displayName, setDisplayName] = useState(tenant.display_name)
  const [minLength, setMinLength] = useState(
    tenant.password_policy_override?.min_length?.toString() ?? '',
  )
  const [maxLength, setMaxLength] = useState(
    tenant.password_policy_override?.max_length?.toString() ?? '',
  )
  const [historyDepth, setHistoryDepth] = useState(
    tenant.password_policy_override?.history_depth?.toString() ?? '',
  )
  const [qUsers, setQUsers] = useState(tenant.quota?.users?.toString() ?? '')
  const [qGroups, setQGroups] = useState(tenant.quota?.groups?.toString() ?? '')
  const [qAgents, setQAgents] = useState(tenant.quota?.agents?.toString() ?? '')
  const [qApps, setQApps] = useState(tenant.quota?.applications?.toString() ?? '')
  const [qClients, setQClients] = useState(tenant.quota?.oauth2_clients?.toString() ?? '')
  const [qSessions, setQSessions] = useState(tenant.quota?.active_sessions?.toString() ?? '')
  const [qConsents, setQConsents] = useState(tenant.quota?.consents?.toString() ?? '')
  const [qJobs, setQJobs] = useState(tenant.quota?.active_jobs?.toString() ?? '')
  const [qAudits, setQAudits] = useState(tenant.quota?.audit_events_retained?.toString() ?? '')
  const [qArtifacts, setQArtifacts] = useState(
    tenant.quota?.export_artifacts_bytes?.toString() ?? '',
  )

  const [saving, setSaving] = useState(false)
  const [editError, setEditError] = useState('')
  const t = useDictionary(systemTenantsDictionary)

  async function handleSave(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSaving(true)
    setEditError('')
    try {
      const policy: AdminTenant['password_policy_override'] = {}
      if (minLength.trim()) policy.min_length = Number.parseInt(minLength, 10)
      if (maxLength.trim()) policy.max_length = Number.parseInt(maxLength, 10)
      if (historyDepth.trim()) policy.history_depth = Number.parseInt(historyDepth, 10)
      const hasPolicy = Object.keys(policy).length > 0
      await updateAdminTenant(csrfToken, tenant.realm, {
        display_name: displayName !== tenant.display_name ? displayName : undefined,
        password_policy_override: hasPolicy ? policy : undefined,
      })

      const num = (v: string) => (v.trim() ? Number.parseInt(v, 10) : undefined)
      await updateAdminTenantQuota(csrfToken, tenant.id, {
        users: num(qUsers),
        groups: num(qGroups),
        agents: num(qAgents),
        applications: num(qApps),
        oauth2_clients: num(qClients),
        active_sessions: num(qSessions),
        consents: num(qConsents),
        active_jobs: num(qJobs),
        audit_events_retained: num(qAudits),
        export_artifacts_bytes: num(qArtifacts),
      })

      onSaved(tenant.id)
    } catch (cause) {
      setEditError(
        cause instanceof AuthenticationAPIError ? cause.message : t.tenantUpdateFailedError,
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSave} className="mt-5 grid gap-3 border-t border-slate-100 pt-5">
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
        {t.editHeading}
      </p>
      {editError ? <Alert variant="destructive">{editError}</Alert> : null}
      <div className="grid gap-1.5">
        <Label htmlFor={`name-${tenant.id}`}>{t.displayNameLabel}</Label>
        <Input
          id={`name-${tenant.id}`}
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
        />
      </div>
      <p className="mt-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
        {t.passwordPolicyOverrideHeading}
      </p>
      <p className="text-xs text-slate-500">{t.passwordPolicyOverrideHelp}</p>
      <div className="grid grid-cols-3 gap-2">
        <div className="grid gap-1.5">
          <Label htmlFor={`min-${tenant.id}`}>{t.minLabel}</Label>
          <Input
            id={`min-${tenant.id}`}
            type="number"
            min={1}
            value={minLength}
            onChange={(e) => setMinLength(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`max-${tenant.id}`}>{t.maxLabel}</Label>
          <Input
            id={`max-${tenant.id}`}
            type="number"
            min={1}
            value={maxLength}
            onChange={(e) => setMaxLength(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`hist-${tenant.id}`}>{t.historyLabel}</Label>
          <Input
            id={`hist-${tenant.id}`}
            type="number"
            min={0}
            value={historyDepth}
            onChange={(e) => setHistoryDepth(e.target.value)}
          />
        </div>
      </div>
      <p className="mt-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
        {t.quotaHeading}
      </p>
      <p className="text-xs text-slate-500">{t.quotaHelp}</p>
      <div className="grid grid-cols-4 gap-2">
        <div className="grid gap-1.5">
          <Label htmlFor={`qu-${tenant.id}`}>{t.quotaUsers}</Label>
          <Input
            id={`qu-${tenant.id}`}
            type="number"
            min={1}
            value={qUsers}
            onChange={(e) => setQUsers(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qg-${tenant.id}`}>{t.quotaGroups}</Label>
          <Input
            id={`qg-${tenant.id}`}
            type="number"
            min={1}
            value={qGroups}
            onChange={(e) => setQGroups(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qa-${tenant.id}`}>{t.quotaAgents}</Label>
          <Input
            id={`qa-${tenant.id}`}
            type="number"
            min={1}
            value={qAgents}
            onChange={(e) => setQAgents(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qap-${tenant.id}`}>{t.quotaApplications}</Label>
          <Input
            id={`qap-${tenant.id}`}
            type="number"
            min={1}
            value={qApps}
            onChange={(e) => setQApps(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qc-${tenant.id}`}>{t.quotaOAuth2Clients}</Label>
          <Input
            id={`qc-${tenant.id}`}
            type="number"
            min={1}
            value={qClients}
            onChange={(e) => setQClients(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qs-${tenant.id}`}>{t.quotaActiveSessions}</Label>
          <Input
            id={`qs-${tenant.id}`}
            type="number"
            min={1}
            value={qSessions}
            onChange={(e) => setQSessions(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qco-${tenant.id}`}>{t.quotaConsents}</Label>
          <Input
            id={`qco-${tenant.id}`}
            type="number"
            min={1}
            value={qConsents}
            onChange={(e) => setQConsents(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qj-${tenant.id}`}>{t.quotaActiveJobs}</Label>
          <Input
            id={`qj-${tenant.id}`}
            type="number"
            min={1}
            value={qJobs}
            onChange={(e) => setQJobs(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qau-${tenant.id}`}>{t.quotaAuditEventsRetained}</Label>
          <Input
            id={`qau-${tenant.id}`}
            type="number"
            min={1}
            value={qAudits}
            onChange={(e) => setQAudits(e.target.value)}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`qar-${tenant.id}`}>{t.quotaExportArtifactsBytes}</Label>
          <Input
            id={`qar-${tenant.id}`}
            type="number"
            min={1}
            value={qArtifacts}
            onChange={(e) => setQArtifacts(e.target.value)}
          />
        </div>
      </div>
      <Button type="submit" disabled={busy || saving} className="mt-2 justify-self-start">
        {saving ? t.saving : t.save}
      </Button>
    </form>
  )
}

export function StatusBadge({ status }: { status: AdminTenant['status'] }) {
  return status === 'active' ? (
    <span className="rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700">
      active
    </span>
  ) : (
    <span className="rounded-md bg-rose-50 px-2 py-0.5 text-xs font-semibold text-rose-700">
      disabled
    </span>
  )
}

function formatDate(value: string | undefined, locale: 'ja' | 'en'): string {
  if (!value) return '—'
  try {
    return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
  } catch {
    return value
  }
}

import { IconBan, IconCheck, IconPlus, IconRefresh } from '@tabler/icons-react'
import { type FormEvent, useMemo, useState } from 'react'
import {
  AuthenticationAPIError,
  createAdminTenant,
  listAdminTenants,
  setAdminTenantDisabled,
  updateAdminTenant,
  updateAdminTenantQuota,
} from '../../api'
import { FieldError } from '../../components/FieldError'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import {
  formSchema,
  optionalNonNegativeInteger,
  requiredText,
  rule,
  useForm,
  useFormValidation,
} from '../../lib/form'
import { useDictionary, useLocale } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'
import { formatDate, StatusBadge } from './SystemTenantsShared'
import { TenantEndpointStyleEditor } from './TenantEndpointStyleEditor'

export { StatusBadge } from './SystemTenantsShared'

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

  async function handleCreate(input: { realm: string; display_name: string }) {
    await run(async () => {
      const created = await createAdminTenant(csrfToken, input)
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
            <CreateTenantForm
              busy={busy}
              onCancel={() => setShowCreate(false)}
              onCreate={handleCreate}
            />
          </Card>
        </div>
      ) : null}
    </SystemShell>
  )
}

function CreateTenantForm({
  busy,
  onCancel,
  onCreate,
}: {
  busy: boolean
  onCancel: () => void
  onCreate: (input: { realm: string; display_name: string }) => Promise<void>
}) {
  const t = useDictionary(systemTenantsDictionary)
  const schema = useMemo(() => createTenantSchema(t.realmInvalidError), [t.realmInvalidError])
  const form = useForm(schema, { realm: '', displayName: '' })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!form.data) {
      form.revealAll()
      return
    }
    void onCreate({ realm: form.data.realm, display_name: form.data.displayName })
  }

  return (
    <form onSubmit={submit} className="mt-4 grid gap-4">
      <div className="grid gap-1.5">
        <Label htmlFor="tenant-realm" required>
          {t.realmLabel}
        </Label>
        <Input id="tenant-realm" required placeholder="acme" {...form.input('realm')} />
        <FieldError form={form} name="realm" />
        <p className="text-xs text-slate-500">{t.realmHelp}</p>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="tenant-display" required>
          {t.displayNameLabel}
        </Label>
        <Input
          id="tenant-display"
          required
          placeholder="Acme Inc."
          {...form.input('displayName')}
        />
        <FieldError form={form} name="displayName" />
      </div>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel} disabled={busy}>
          {t.cancel}
        </Button>
        <Button type="submit" disabled={busy || !form.data}>
          {t.create}
        </Button>
      </div>
    </form>
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
        <dt className="text-slate-500">{t.endpointStyleLabel}</dt>
        <dd>{tenant.endpoint_style ?? 'path'}</dd>
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
      <TenantEndpointStyleEditor
        tenant={tenant}
        csrfToken={csrfToken}
        busy={busy}
        onSaved={onSaved}
      />
    </Card>
  )
}

const QUOTA_KEYS = [
  'users',
  'groups',
  'agents',
  'applications',
  'oauth2_clients',
  'active_sessions',
  'consents',
  'active_jobs',
  'ssf_streams',
  'audit_events_retained',
  'export_artifacts_bytes',
] as const

type QuotaKey = (typeof QUOTA_KEYS)[number]

// 入力欄の id の接頭辞は、欄を一覧のデータから描く前と同じ値を保つ。E2E が id で欄を探すためである。
const QUOTA_FIELDS: Record<
  QuotaKey,
  { label: keyof (typeof systemTenantsDictionary)['en']; idPrefix: string }
> = {
  users: { label: 'quotaUsers', idPrefix: 'qu' },
  groups: { label: 'quotaGroups', idPrefix: 'qg' },
  agents: { label: 'quotaAgents', idPrefix: 'qa' },
  applications: { label: 'quotaApplications', idPrefix: 'qap' },
  oauth2_clients: { label: 'quotaOAuth2Clients', idPrefix: 'qc' },
  active_sessions: { label: 'quotaActiveSessions', idPrefix: 'qs' },
  consents: { label: 'quotaConsents', idPrefix: 'qco' },
  active_jobs: { label: 'quotaActiveJobs', idPrefix: 'qj' },
  ssf_streams: { label: 'quotaSsfStreams', idPrefix: 'qss' },
  audit_events_retained: { label: 'quotaAuditEventsRetained', idPrefix: 'qau' },
  export_artifacts_bytes: { label: 'quotaExportArtifactsBytes', idPrefix: 'qar' },
}

function quotaForm(tenant: AdminTenant): Record<QuotaKey, string> {
  return Object.fromEntries(
    QUOTA_KEYS.map((key) => [key, tenant.quota?.[key]?.toString() ?? '']),
  ) as Record<QuotaKey, string>
}

// 表示名は Tenant の上限に、クォータは TenantQuotaUpdateRequest の @minValue(0) に従う。
// パスワードポリシーの下限と上限は全体の既定値で決まり、この画面は値を持たないのでサーバーが判定する。
const tenantEditorSchema = formSchema({
  displayName: requiredText(LENGTH.displayName),
  minLength: optionalNonNegativeInteger(),
  maxLength: optionalNonNegativeInteger(),
  historyDepth: optionalNonNegativeInteger(),
  ...(Object.fromEntries(QUOTA_KEYS.map((key) => [key, optionalNonNegativeInteger()])) as Record<
    QuotaKey,
    ReturnType<typeof optionalNonNegativeInteger>
  >),
})

// realm は Tenant の @pattern と同じく、英小文字か数字で始まり終わる 63 文字以内の DNS ラベルである。
const REALM = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/

function createTenantSchema(realmInvalid: string) {
  return formSchema({
    realm: rule(requiredText(), (value) =>
      REALM.test(value) ? undefined : { kind: 'message', text: realmInvalid },
    ),
    displayName: requiredText(LENGTH.displayName),
  })
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
  const [quota, setQuota] = useState(() => quotaForm(tenant))
  const [saving, setSaving] = useState(false)
  const [editError, setEditError] = useState('')
  const t = useDictionary(systemTenantsDictionary)
  // テナントの更新とクォータの更新は別の要求なので、片方だけが保存されないよう送る前にすべて検査する。
  const validation = useFormValidation(tenantEditorSchema, {
    displayName,
    minLength,
    maxLength,
    historyDepth,
    ...quota,
  })

  async function handleSave(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    setSaving(true)
    setEditError('')
    try {
      const policy: AdminTenant['password_policy_override'] = {}
      if (data.minLength !== undefined) policy.min_length = data.minLength
      if (data.maxLength !== undefined) policy.max_length = data.maxLength
      if (data.historyDepth !== undefined) policy.history_depth = data.historyDepth
      const hasPolicy = Object.keys(policy).length > 0
      await updateAdminTenant(csrfToken, tenant.realm, {
        display_name: data.displayName !== tenant.display_name ? data.displayName : undefined,
        password_policy_override: hasPolicy ? policy : undefined,
      })
      await updateAdminTenantQuota(
        csrfToken,
        tenant.realm,
        Object.fromEntries(QUOTA_KEYS.map((key) => [key, data[key]])),
      )

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
          {...validation.fieldProps('displayName')}
        />
        <FieldError form={validation} name="displayName" />
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
            {...validation.fieldProps('minLength')}
          />
          <FieldError form={validation} name="minLength" />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`max-${tenant.id}`}>{t.maxLabel}</Label>
          <Input
            id={`max-${tenant.id}`}
            type="number"
            min={1}
            value={maxLength}
            onChange={(e) => setMaxLength(e.target.value)}
            {...validation.fieldProps('maxLength')}
          />
          <FieldError form={validation} name="maxLength" />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`hist-${tenant.id}`}>{t.historyLabel}</Label>
          <Input
            id={`hist-${tenant.id}`}
            type="number"
            min={0}
            value={historyDepth}
            onChange={(e) => setHistoryDepth(e.target.value)}
            {...validation.fieldProps('historyDepth')}
          />
          <FieldError form={validation} name="historyDepth" />
        </div>
      </div>
      <p className="mt-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
        {t.quotaHeading}
      </p>
      <p className="text-xs text-slate-500">{t.quotaHelp}</p>
      <div className="grid grid-cols-4 gap-2">
        {QUOTA_KEYS.map((key) => (
          <div key={key} className="grid gap-1.5">
            <Label htmlFor={`${QUOTA_FIELDS[key].idPrefix}-${tenant.id}`}>
              {t[QUOTA_FIELDS[key].label]}
            </Label>
            <Input
              id={`${QUOTA_FIELDS[key].idPrefix}-${tenant.id}`}
              type="number"
              min={0}
              value={quota[key]}
              onChange={(e) => setQuota((current) => ({ ...current, [key]: e.target.value }))}
              {...validation.fieldProps(key)}
            />
            <FieldError form={validation} name={key} />
          </div>
        ))}
      </div>
      <Button
        type="submit"
        disabled={busy || saving || !validation.data}
        className="mt-2 justify-self-start"
      >
        {saving ? t.saving : t.save}
      </Button>
    </form>
  )
}

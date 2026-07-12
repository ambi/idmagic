import { IconMail, IconPalette, IconShieldLock, IconTag, IconUsers } from '@tabler/icons-react'
import { type FormEvent, useState, useEffect } from 'react'
import {
  AuthenticationAPIError,
  updateAdminSettings,
  listScimTokens,
  createScimToken,
  revokeScimToken,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary, useLocale } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import type { AdminSettings, ScimToken } from '../../types'
import { adminSettingsDictionary, type AdminSettingsDictionary } from './AdminSettingsPage.i18n'
import { BrandingTab } from './BrandingTab'

const DEFAULT_REALM = 'default'

export function displayNameError(value: string, t: AdminSettingsDictionary): string | null {
  return value.trim() ? null : t.displayNameRequiredError
}

export function passwordPolicyOverride(
  minLength: string,
  maxLength: string,
  historyDepth: string,
): NonNullable<AdminSettings['password_policy_override']> {
  const policy: NonNullable<AdminSettings['password_policy_override']> = {}
  if (minLength.trim()) policy.min_length = Number.parseInt(minLength, 10)
  if (maxLength.trim()) policy.max_length = Number.parseInt(maxLength, 10)
  if (historyDepth.trim()) policy.history_depth = Number.parseInt(historyDepth, 10)
  return policy
}

type TabKey = 'general' | 'password-policy' | 'branding' | 'email' | 'scim'

type Tab = {
  key: TabKey
  label: string
  description: string
  icon: typeof IconTag
  disabled?: boolean
}

function tabs(t: AdminSettingsDictionary): Tab[] {
  return [
    {
      key: 'general',
      label: t.tabGeneralLabel,
      description: t.tabGeneralDescription,
      icon: IconTag,
    },
    {
      key: 'password-policy',
      label: t.tabPasswordPolicyLabel,
      description: t.tabPasswordPolicyDescription,
      icon: IconShieldLock,
    },
    {
      key: 'branding',
      label: t.tabBrandingLabel,
      description: t.tabBrandingDescription,
      icon: IconPalette,
    },
    {
      key: 'scim',
      label: t.tabScimLabel,
      description: t.tabScimDescription,
      icon: IconUsers,
    },
    {
      key: 'email',
      label: t.tabEmailLabel,
      description: t.tabEmailDescription,
      icon: IconMail,
      disabled: true,
    },
  ]
}

export function AdminSettingsPage({
  csrfToken,
  actorUsername,
  actorRoles,
  actorRealm,
  settings: initial,
}: {
  csrfToken: string
  actorUsername?: string
  actorRoles: string[]
  actorRealm: string
  settings: AdminSettings
}) {
  const [settings, setSettings] = useState(initial)
  const [active, setActive] = useState<TabKey>('general')
  const isSystemAdminOnDefault = actorRoles.includes('system_admin') && actorRealm === DEFAULT_REALM
  const t = useDictionary(adminSettingsDictionary)
  const tabList = tabs(t)

  return (
    <AdminShell
      active="settings"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      {isSystemAdminOnDefault ? (
        <Alert>
          <p className="text-sm text-slate-700">
            {t.crossTenantNoticePrefix}
            <a href="/system/tenants" className="ml-1 font-medium text-blue-700 hover:underline">
              {t.crossTenantNoticeLinkText}
            </a>
            {t.crossTenantNoticeSuffix}
          </p>
        </Alert>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[220px_minmax(0,1fr)]">
        <nav className="flex flex-col gap-1" aria-label={t.tabsAriaLabel}>
          {tabList.map((tab) => (
            <button
              key={tab.key}
              type="button"
              onClick={() => !tab.disabled && setActive(tab.key)}
              disabled={tab.disabled}
              aria-current={active === tab.key ? 'page' : undefined}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-left text-sm font-medium',
                tab.disabled
                  ? 'cursor-not-allowed text-slate-400'
                  : active === tab.key
                    ? 'bg-slate-950 text-white shadow-sm'
                    : 'text-slate-600 hover:bg-white hover:text-slate-950 hover:shadow-xs',
              )}
            >
              <tab.icon size={18} stroke={1.8} aria-hidden="true" />
              <span className="flex-1">{tab.label}</span>
              {tab.disabled ? (
                <span className="rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-500">
                  {t.comingSoonBadge}
                </span>
              ) : null}
            </button>
          ))}
        </nav>

        <div className="min-w-0">
          {active === 'general' ? (
            <GeneralTab
              csrfToken={csrfToken}
              settings={settings}
              onSaved={(next) => setSettings(next)}
            />
          ) : null}
          {active === 'password-policy' ? (
            <PasswordPolicyTab
              csrfToken={csrfToken}
              settings={settings}
              onSaved={(next) => setSettings(next)}
            />
          ) : null}
          {active === 'branding' ? <BrandingTab csrfToken={csrfToken} /> : null}
          {active === 'scim' ? <ScimTab csrfToken={csrfToken} tenantID={settings.realm} /> : null}
          {active === 'email' ? (
            <Card className="p-6">
              <h2 className="text-base font-semibold text-slate-900">{t.emailTabHeading}</h2>
              <p className="mt-2 text-sm text-slate-600">{t.emailTabDescription}</p>
            </Card>
          ) : null}
        </div>
      </div>
    </AdminShell>
  )
}

function GeneralTab({
  csrfToken,
  settings,
  onSaved,
}: {
  csrfToken: string
  settings: AdminSettings
  onSaved: (next: AdminSettings) => void
}) {
  const [displayName, setDisplayName] = useState(settings.display_name)
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminSettingsDictionary)

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const trimmed = displayName.trim()
      const validationError = displayNameError(displayName, t)
      if (validationError) {
        setError(validationError)
        return
      }
      if (trimmed === settings.display_name) {
        setNotice(t.noChangesNotice)
        return
      }
      const next = await updateAdminSettings(csrfToken, { display_name: trimmed })
      onSaved(next)
      setDisplayName(next.display_name)
      setEditing(false)
      setNotice(t.displayNameUpdatedNotice)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : t.settingsUpdateFailedError,
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="p-6">
      <header>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900">{t.generalHeading}</h2>
            <p className="mt-1 text-sm text-slate-600">{t.generalSubheading}</p>
          </div>
          {!editing ? (
            <Button type="button" variant="outline" onClick={() => setEditing(true)}>
              {t.edit}
            </Button>
          ) : null}
        </div>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-2">
            <ReadSetting label={t.tenantIdLabel} value={settings.tenant_id} mono />
            <ReadSetting label={t.displayNameLabel} value={settings.display_name} />
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="tenant-id">{t.tenantIdLabel}</Label>
              <Input
                id="tenant-id"
                value={settings.tenant_id}
                readOnly
                aria-readonly="true"
                className="bg-slate-50 font-mono"
                tabIndex={-1}
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="display-name">{t.displayNameLabel}</Label>
              <Input
                id="display-name"
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                maxLength={200}
              />
              <p className="text-xs text-slate-500">{t.displayNameHelp}</p>
            </div>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving}>
                {saving ? t.saving : t.save}
              </Button>
              <Button
                type="button"
                variant="ghost"
                disabled={saving}
                onClick={() => {
                  setDisplayName(settings.display_name)
                  setEditing(false)
                }}
              >
                {t.cancel}
              </Button>
            </div>
          </form>
        )}
      </div>
    </Card>
  )
}

function PasswordPolicyTab({
  csrfToken,
  settings,
  onSaved,
}: {
  csrfToken: string
  settings: AdminSettings
  onSaved: (next: AdminSettings) => void
}) {
  const override = settings.password_policy_override
  const defaults = settings.password_policy_defaults
  const [minLength, setMinLength] = useState(override?.min_length?.toString() ?? '')
  const [maxLength, setMaxLength] = useState(override?.max_length?.toString() ?? '')
  const [historyDepth, setHistoryDepth] = useState(override?.history_depth?.toString() ?? '')
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminSettingsDictionary)

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const policy = passwordPolicyOverride(minLength, maxLength, historyDepth)
      const next = await updateAdminSettings(csrfToken, {
        password_policy_override: policy,
      })
      onSaved(next)
      setMinLength(next.password_policy_override?.min_length?.toString() ?? '')
      setMaxLength(next.password_policy_override?.max_length?.toString() ?? '')
      setHistoryDepth(next.password_policy_override?.history_depth?.toString() ?? '')
      setEditing(false)
      setNotice(t.passwordPolicyUpdatedNotice)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : t.passwordPolicyUpdateFailedError,
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="p-6">
      <header>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900">{t.passwordPolicyHeading}</h2>
            <p className="mt-1 text-sm text-slate-600">{t.passwordPolicySubheading}</p>
          </div>
          {!editing ? (
            <Button type="button" variant="outline" onClick={() => setEditing(true)}>
              {t.edit}
            </Button>
          ) : null}
        </div>
        <dl className="mt-3 grid grid-cols-3 gap-3 rounded-md border border-slate-200 bg-slate-50 px-4 py-3 text-xs">
          <div>
            <dt className="text-slate-500">{t.standardMinLengthLabel}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {`${defaults.min_length}${t.charsSuffix}`}
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">{t.standardMaxLengthLabel}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {`${defaults.max_length}${t.charsSuffix}`}
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">{t.standardHistoryDepthLabel}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {`${defaults.history_depth}${t.countSuffix}`}
            </dd>
          </div>
        </dl>
        <p className="mt-2 text-xs text-slate-500">{t.weakerPolicyWarning}</p>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-3">
            <ReadSetting
              label={t.minLengthLabel}
              value={`${override?.min_length ?? defaults.min_length}${t.charsSuffix}`}
            />
            <ReadSetting
              label={t.maxLengthLabel}
              value={`${override?.max_length ?? defaults.max_length}${t.charsSuffix}`}
            />
            <ReadSetting
              label={t.historyDepthLabel}
              value={`${override?.history_depth ?? defaults.history_depth}${t.countSuffix}`}
            />
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <PolicyField
                id="min-length"
                label={t.minLengthFieldLabel}
                value={minLength}
                onChange={setMinLength}
                min={defaults.min_length}
                max={defaults.max_length}
                placeholder={defaults.min_length.toString()}
                hint={t.atLeastHint.replace('{n}', defaults.min_length.toString())}
              />
              <PolicyField
                id="max-length"
                label={t.maxLengthFieldLabel}
                value={maxLength}
                onChange={setMaxLength}
                min={defaults.min_length}
                max={defaults.max_length}
                placeholder={defaults.max_length.toString()}
                hint={t.atMostHint.replace('{n}', defaults.max_length.toString())}
              />
              <PolicyField
                id="history-depth"
                label={t.historyDepthFieldLabel}
                value={historyDepth}
                onChange={setHistoryDepth}
                min={defaults.history_depth}
                max={50}
                placeholder={defaults.history_depth.toString()}
                hint={t.atLeastHint.replace('{n}', defaults.history_depth.toString())}
              />
            </div>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving}>
                {saving ? t.saving : t.save}
              </Button>
              <Button
                type="button"
                variant="ghost"
                disabled={saving}
                onClick={() => {
                  setMinLength(settings.password_policy_override?.min_length?.toString() ?? '')
                  setMaxLength(settings.password_policy_override?.max_length?.toString() ?? '')
                  setHistoryDepth(
                    settings.password_policy_override?.history_depth?.toString() ?? '',
                  )
                  setEditing(false)
                }}
              >
                {t.cancel}
              </Button>
            </div>
          </form>
        )}
      </div>
    </Card>
  )
}

function ReadSetting({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="rounded-lg border border-slate-200/80 bg-white/70 px-3 py-2.5">
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className={cn('mt-0.5 text-sm font-medium text-slate-900', mono && 'font-mono')}>
        {value}
      </dd>
    </div>
  )
}

function PolicyField({
  id,
  label,
  value,
  onChange,
  min,
  max,
  placeholder,
  hint,
}: {
  id: string
  label: string
  value: string
  onChange: (next: string) => void
  min: number
  max: number
  placeholder: string
  hint: string
}) {
  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="number"
        min={min}
        max={max}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
      <p className="text-xs text-slate-500">{hint}</p>
    </div>
  )
}

function ScimTab({ csrfToken, tenantID }: { csrfToken: string; tenantID: string }) {
  const [tokens, setTokens] = useState<ScimToken[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [tokenDesc, setTokenDesc] = useState('')
  const [tokenExpiry, setTokenExpiry] = useState('7')
  const [generatedToken, setGeneratedToken] = useState('')
  const [creating, setCreating] = useState(false)
  const t = useDictionary(adminSettingsDictionary)
  const { locale } = useLocale()

  // biome-ignore lint/correctness/useExhaustiveDependencies: 初回マウント時のみ取得する
  useEffect(() => {
    async function loadData() {
      try {
        const tList = await listScimTokens()
        setTokens(tList)
      } catch {
        setError(t.scimTokensFetchFailedError)
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [])

  async function handleCreateToken(e: FormEvent) {
    e.preventDefault()
    setError('')
    setNotice('')
    setGeneratedToken('')
    if (!tokenDesc.trim()) {
      setError(t.tokenDescriptionRequiredError)
      return
    }
    try {
      const res = await createScimToken(csrfToken, {
        description: tokenDesc.trim(),
        expiry_days: Number.parseInt(tokenExpiry, 10),
      })
      setGeneratedToken(res.token)
      setTokenDesc('')
      setCreating(false)
      const tList = await listScimTokens()
      setTokens(tList)
      setNotice(t.scimTokenIssuedNotice)
    } catch {
      setError(t.tokenIssueFailedError)
    }
  }

  async function handleRevokeToken(id: string) {
    setError('')
    setNotice('')
    try {
      await revokeScimToken(csrfToken, id)
      setTokens(tokens.filter((token) => token.id !== id))
      setNotice(t.tokenRevokedNotice)
    } catch {
      setError(t.tokenRevokeFailedError)
    }
  }

  if (loading) {
    return <div className="text-sm text-slate-500">{t.loadingNotice}</div>
  }

  const endpointUrl = `${window.location.origin}/realms/${tenantID}/scim/v2`

  return (
    <Card className="p-6">
      <header>
        <h2 className="text-base font-semibold text-slate-900">{t.scimHeading}</h2>
        <p className="mt-1 text-sm text-slate-600">{t.scimDescription}</p>
      </header>

      <div className="mt-6 grid gap-6">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />

        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
          <h3 className="text-sm font-semibold text-slate-900">{t.connectionInfoHeading}</h3>
          <div className="mt-3 grid gap-3">
            <div>
              <span className="text-xs text-slate-500">{t.scimBaseUrlLabel}</span>
              <div className="mt-1 flex items-center gap-2">
                <input
                  readOnly
                  value={endpointUrl}
                  className="flex-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 font-mono text-sm"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    navigator.clipboard.writeText(endpointUrl)
                    setNotice(t.urlCopiedNotice)
                  }}
                >
                  {t.copy}
                </Button>
              </div>
              <p className="mt-1 text-xs text-slate-500">{t.scimConnectorHelp}</p>
            </div>
          </div>
        </div>

        <div className="grid gap-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="text-sm font-semibold text-slate-900">{t.scimTokenHeading}</h3>
            {!creating ? (
              <Button type="button" variant="outline" onClick={() => setCreating(true)}>
                {t.issueToken}
              </Button>
            ) : null}
          </div>

          {generatedToken ? (
            <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-4">
              <h4 className="text-sm font-bold text-emerald-800">{t.issuedTokenHeading}</h4>
              <p className="mt-1 text-xs text-emerald-700">{t.issuedTokenWarning}</p>
              <div className="mt-3 flex items-center gap-2">
                <input
                  readOnly
                  value={generatedToken}
                  className="flex-1 rounded-md border border-emerald-300 bg-white px-3 py-1.5 font-mono text-sm text-emerald-900"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    navigator.clipboard.writeText(generatedToken)
                    setNotice(t.tokenCopiedNotice)
                  }}
                >
                  {t.copy}
                </Button>
              </div>
            </div>
          ) : null}

          {tokens.length === 0 ? (
            <p className="text-sm text-slate-500">{t.noTokensNotice}</p>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-slate-200">
              <table className="min-w-full divide-y divide-slate-200 text-left text-sm text-slate-700">
                <thead className="bg-slate-50 font-semibold text-slate-900">
                  <tr>
                    <th className="px-4 py-2">{t.tableHeaderDescription}</th>
                    <th className="px-4 py-2">{t.tableHeaderCreatedAt}</th>
                    <th className="px-4 py-2">{t.tableHeaderExpiresAt}</th>
                    <th className="px-4 py-2">{t.tableHeaderAction}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {tokens.map((tok) => (
                    <tr key={tok.id}>
                      <td className="px-4 py-3">{tok.description}</td>
                      <td className="px-4 py-3">
                        {new Date(tok.created_at).toLocaleString(
                          locale === 'ja' ? 'ja-JP' : 'en-US',
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {tok.expires_at
                          ? new Date(tok.expires_at).toLocaleString(
                              locale === 'ja' ? 'ja-JP' : 'en-US',
                            )
                          : t.noneLabel}
                      </td>
                      <td className="px-4 py-3">
                        <Button
                          type="button"
                          variant="ghost"
                          className="text-red-600 hover:text-red-700 hover:bg-red-50"
                          onClick={() => handleRevokeToken(tok.id)}
                        >
                          {t.revoke}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {creating ? (
            <form
              onSubmit={handleCreateToken}
              className="mt-4 rounded-lg border border-slate-200 p-4"
            >
              <h4 className="text-sm font-semibold text-slate-900">{t.newTokenHeading}</h4>
              <div className="mt-3 grid gap-4 sm:grid-cols-2">
                <div className="grid gap-1.5">
                  <Label htmlFor="token-desc">{t.tokenDescriptionLabel}</Label>
                  <Input
                    id="token-desc"
                    placeholder={t.tokenDescriptionPlaceholder}
                    value={tokenDesc}
                    onChange={(e) => setTokenDesc(e.target.value)}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="token-expiry">{t.tokenExpiryLabel}</Label>
                  <Input
                    id="token-expiry"
                    type="number"
                    min={1}
                    max={365}
                    value={tokenExpiry}
                    onChange={(e) => setTokenExpiry(e.target.value)}
                  />
                </div>
              </div>
              <div className="mt-4 flex items-center gap-2">
                <Button type="submit">{t.issueToken}</Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setTokenDesc('')
                    setError('')
                    setCreating(false)
                  }}
                >
                  {t.cancel}
                </Button>
              </div>
            </form>
          ) : null}
        </div>
      </div>
    </Card>
  )
}

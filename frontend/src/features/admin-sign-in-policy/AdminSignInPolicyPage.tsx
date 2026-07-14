import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, tenantURL, updateTenantDefaultSignInPolicy } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select, type SelectOption } from '../../components/ui/select'
import { useDictionary } from '../../lib/i18n'
import { validateReauthMaxAge, parseNetworkCIDRs } from '../../lib/validation'
import type {
  AdminApplication,
  AppSignInPolicyView,
  RequiredAuthnStrength,
  SignInRule,
  TenantDefaultSignInPolicy,
} from '../../types'
import {
  adminSignInPolicyDictionary,
  type AdminSignInPolicyDictionary,
} from './AdminSignInPolicyPage.i18n'

function strengthOptions(t: AdminSignInPolicyDictionary): SelectOption[] {
  return [
    { value: 'Password', label: t.strengthPasswordLabel },
    { value: 'Mfa', label: t.strengthMfaLabel },
  ]
}

// アプリ単位のサインインポリシー一覧行。上書きの有無・警告・実効ポリシーを総合表示する。
export type SignInPolicyAppRow = {
  application: AdminApplication
  view: AppSignInPolicyView
}

// 内部ルール名。UI には表示しないが、保存契約上ルールには名前が要る (ADR-081)。
const DEFAULT_RULE_NAME = 'tenant-default'

function strengthLabel(strength: RequiredAuthnStrength, t: AdminSignInPolicyDictionary): string {
  return strength === 'Mfa' ? t.strengthMfaLabel : t.strengthPasswordLabel
}

// summarizeRules は実効ルール列を利用者向けの短い文へ要約する。内部ルール名は見せない (ADR-081)。
function summarizeRules(rules: SignInRule[], t: AdminSignInPolicyDictionary): string {
  const enabled = rules.filter((rule) => rule.enabled)
  if (enabled.length === 0) {
    return t.noAdditionalRequirementsNotice
  }
  return enabled
    .map((rule) => {
      const parts: string[] = [strengthLabel(rule.required_authn.strength, t)]
      if (rule.condition.reauth_max_age_seconds) {
        parts.push(
          t.reauthSuffix.replace('{seconds}', String(rule.condition.reauth_max_age_seconds)),
        )
      }
      const cidrs = rule.condition.network_allow_cidrs ?? []
      if (cidrs.length > 0) {
        parts.push(t.allowedNetworkPrefix.replace('{cidrs}', cidrs.join(', ')))
      }
      return parts.join(' / ')
    })
    .join('、')
}

// AdminSignInPolicyPage はテナントのサインインポリシーを総合的に扱う画面 (wi-115, ADR-081)。
// 上段で全アプリ共通のデフォルトポリシーを詳細表示/編集し、下段で各アプリの上書き・実効ポリシーを一覧する。
export function AdminSignInPolicyPage({
  csrfToken,
  actorUsername,
  policy,
  apps,
  unenrolledUserCount,
}: {
  csrfToken: string
  actorUsername?: string
  policy: TenantDefaultSignInPolicy
  apps: SignInPolicyAppRow[]
  unenrolledUserCount: number
}) {
  const t = useDictionary(adminSignInPolicyDictionary)
  return (
    <AdminShell
      active="sign-in-policy"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      <div className="grid gap-6">
        <DefaultPolicyCard
          csrfToken={csrfToken}
          policy={policy}
          unenrolledUserCount={unenrolledUserCount}
        />
        <ApplicationPolicyList apps={apps} />
      </div>
    </AdminShell>
  )
}

function DefaultPolicyCard({
  csrfToken,
  policy: initial,
  unenrolledUserCount,
}: {
  csrfToken: string
  policy: TenantDefaultSignInPolicy
  unenrolledUserCount: number
}) {
  const [rule, setRule] = useState<SignInRule | undefined>(initial.rules?.[0])
  const [editing, setEditing] = useState(false)
  const t = useDictionary(adminSignInPolicyDictionary)

  const strength = rule?.required_authn.strength ?? 'Password'
  const reauth = rule?.condition.reauth_max_age_seconds
  const cidrs = rule?.condition.network_allow_cidrs ?? []

  return (
    <Card className="p-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-slate-900">{t.defaultPolicyHeading}</h2>
          <p className="mt-1 text-sm text-slate-600">{t.defaultPolicyDescription}</p>
        </div>
        {!editing ? (
          <Button type="button" variant="outline" onClick={() => setEditing(true)}>
            {t.edit}
          </Button>
        ) : null}
      </header>
      <div className="mt-5">
        {unenrolledUserCount > 0 ? (
          <Alert className="mb-4">
            {t.unenrolledWarning.replace('{count}', String(unenrolledUserCount))}
          </Alert>
        ) : null}
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-3">
            <ReadField
              label={t.requiredAuthnStrengthFieldLabel}
              value={strengthLabel(strength, t)}
            />
            <ReadField
              label={t.reauthTimeFieldLabel}
              value={
                reauth
                  ? t.reauthSecondsValue.replace('{seconds}', String(reauth))
                  : t.unlimitedNoReauthNotice
              }
            />
            <ReadField
              label={t.allowedNetworksFieldLabel}
              value={cidrs.length > 0 ? cidrs.join(', ') : t.noRestrictionNotice}
            />
          </dl>
        ) : (
          <DefaultPolicyForm
            csrfToken={csrfToken}
            rule={rule}
            onCancel={() => setEditing(false)}
            onSaved={(next) => {
              setRule(next.rules?.[0])
              setEditing(false)
            }}
          />
        )}
      </div>
    </Card>
  )
}

export function DefaultPolicyFormPresentation({
  rule,
  onCancel,
  onSubmit,
  saving,
  error: externalError,
}: {
  rule?: SignInRule
  onCancel: () => void
  onSubmit: (rules: SignInRule[]) => Promise<void>
  saving: boolean
  error?: string
}) {
  const [strength, setStrength] = useState<RequiredAuthnStrength>(
    rule?.required_authn.strength ?? 'Password',
  )
  const [reauthMaxAge, setReauthMaxAge] = useState(
    rule?.condition.reauth_max_age_seconds?.toString() ?? '',
  )
  const [networkCIDRs, setNetworkCIDRs] = useState(
    (rule?.condition.network_allow_cidrs ?? []).join('\n'),
  )
  const [enforcementStart, setEnforcementStart] = useState(
    rule?.mfa_enrollment?.enforcement_start_at?.slice(0, 16) ?? '',
  )
  const [gracePeriod, setGracePeriod] = useState(
    rule?.mfa_enrollment?.grace_period_seconds?.toString() ?? '900',
  )
  const [allowAdminBypass, setAllowAdminBypass] = useState(
    rule?.mfa_enrollment?.allow_admin_bypass ?? true,
  )
  const [validationError, setValidationError] = useState('')
  const t = useDictionary(adminSignInPolicyDictionary)

  const error = validationError || externalError

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setValidationError('')

    const validationResult = validateReauthMaxAge(reauthMaxAge)
    if (!validationResult.isValid) {
      setValidationError(t.reauthMaxAgeInvalidError)
      return
    }

    const cidrs = parseNetworkCIDRs(networkCIDRs)
    const graceSeconds = Number(gracePeriod)
    if (
      strength === 'Mfa' &&
      (!enforcementStart || !Number.isInteger(graceSeconds) || graceSeconds <= 0)
    ) {
      setValidationError(t.mfaEnrollmentInvalidError)
      return
    }

    // デフォルトは常に適用される baseline なので、認証強度が最低でも 1 ルールとして保存する。
    const rules: SignInRule[] = [
      {
        rule_id: rule?.rule_id ?? '',
        name: DEFAULT_RULE_NAME,
        enabled: true,
        required_authn: { strength },
        condition: {
          reauth_max_age_seconds: validationResult.parsed,
          network_allow_cidrs: cidrs.length > 0 ? cidrs : undefined,
        },
        mfa_enrollment:
          strength === 'Mfa'
            ? {
                enforcement_start_at: new Date(enforcementStart).toISOString(),
                grace_period_seconds: graceSeconds,
                allow_admin_bypass: allowAdminBypass,
              }
            : undefined,
      },
    ]

    await onSubmit(rules)
  }

  return (
    <form onSubmit={handleSubmit} className="grid gap-4">
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="grid gap-1.5">
        <Label>{t.requiredAuthnStrengthFieldLabel}</Label>
        <Select
          value={strength}
          onValueChange={(value) => setStrength(value as RequiredAuthnStrength)}
          options={strengthOptions(t)}
          className="w-full sm:w-72"
        />
        <p className="text-xs text-slate-500">{t.mfaStepUpHelp}</p>
      </div>
      {strength === 'Mfa' ? (
        <div className="grid gap-4 rounded-lg border border-amber-200 bg-amber-50/50 p-4">
          <div className="grid gap-1.5">
            <Label htmlFor="mfa-enforcement-start">{t.enforcementStartLabel}</Label>
            <Input
              id="mfa-enforcement-start"
              type="datetime-local"
              required
              value={enforcementStart}
              onChange={(event) => setEnforcementStart(event.target.value)}
              className="sm:w-72"
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="mfa-grace-period">{t.gracePeriodLabel}</Label>
            <Input
              id="mfa-grace-period"
              type="number"
              min="1"
              required
              value={gracePeriod}
              onChange={(event) => setGracePeriod(event.target.value)}
              className="sm:w-72"
            />
            <p className="text-xs text-slate-500">{t.gracePeriodHelp}</p>
          </div>
          <label className="flex items-start gap-2 text-sm text-slate-700">
            <input
              type="checkbox"
              checked={allowAdminBypass}
              onChange={(event) => setAllowAdminBypass(event.target.checked)}
            />
            <span>{t.allowAdminBypassLabel}</span>
          </label>
        </div>
      ) : null}
      <div className="grid gap-1.5">
        <Label htmlFor="default-reauth">{t.reauthSecondsFieldLabel}</Label>
        <Input
          id="default-reauth"
          type="number"
          min="1"
          value={reauthMaxAge}
          onChange={(e) => setReauthMaxAge(e.target.value)}
          placeholder={t.reauthSecondsPlaceholder}
          className="sm:w-72"
        />
        <p className="text-xs text-slate-500">{t.reauthSecondsHelp}</p>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="default-cidrs">{t.allowedNetworksCidrFieldLabel}</Label>
        <textarea
          id="default-cidrs"
          value={networkCIDRs}
          onChange={(e) => setNetworkCIDRs(e.target.value)}
          rows={3}
          spellCheck={false}
          className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
          placeholder={'10.0.0.0/8\n192.168.1.0/24'}
        />
        <p className="text-xs text-slate-500">{t.allowedNetworksHelp}</p>
      </div>
      <div className="flex items-center gap-2 border-t border-slate-200 pt-5">
        <Button type="submit" disabled={saving}>
          {saving ? t.saving : t.save}
        </Button>
        <Button type="button" variant="ghost" disabled={saving} onClick={onCancel}>
          {t.cancel}
        </Button>
      </div>
    </form>
  )
}

function DefaultPolicyForm({
  csrfToken,
  rule,
  onCancel,
  onSaved,
}: {
  csrfToken: string
  rule?: SignInRule
  onCancel: () => void
  onSaved: (next: TenantDefaultSignInPolicy) => void
}) {
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminSignInPolicyDictionary)

  async function handleSubmit(rules: SignInRule[]) {
    setSaving(true)
    setError('')
    try {
      const next = await updateTenantDefaultSignInPolicy(csrfToken, rules)
      onSaved(next)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : t.defaultPolicyUpdateFailedError,
      )
      setSaving(false)
    }
  }

  return (
    <DefaultPolicyFormPresentation
      rule={rule}
      onCancel={onCancel}
      onSubmit={handleSubmit}
      saving={saving}
      error={error}
    />
  )
}

function ReadField({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-slate-200/80 bg-white/70 px-3 py-2.5">
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className="mt-0.5 text-sm font-medium text-slate-900">{value}</dd>
    </div>
  )
}

function ApplicationPolicyList({ apps }: { apps: SignInPolicyAppRow[] }) {
  const t = useDictionary(adminSignInPolicyDictionary)
  return (
    <Card className="p-6">
      <header>
        <h2 className="text-base font-semibold text-slate-900">{t.applicationPolicyHeading}</h2>
        <p className="mt-1 text-sm text-slate-600">{t.applicationPolicyDescription}</p>
      </header>
      <div className="mt-5">
        {apps.length === 0 ? (
          <p className="text-sm text-slate-500">{t.noAppsNotice}</p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-slate-200">
            <table className="min-w-full divide-y divide-slate-200 text-left text-sm text-slate-700">
              <thead className="bg-slate-50 font-semibold text-slate-900">
                <tr>
                  <th className="px-4 py-2">{t.tableHeaderApplication}</th>
                  <th className="px-4 py-2">{t.tableHeaderOverride}</th>
                  <th className="px-4 py-2">{t.tableHeaderEffectivePolicy}</th>
                  <th className="px-4 py-2" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200">
                {apps.map(({ application, view }) => {
                  const hasOverride = (view.policy.rules ?? []).some((rule) => rule.enabled)
                  return (
                    <tr key={application.application_id}>
                      <td className="px-4 py-3 font-medium text-slate-900">{application.name}</td>
                      <td className="px-4 py-3">
                        {hasOverride ? (
                          <span className="inline-flex items-center gap-1.5">
                            <span className="rounded-md bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700">
                              {t.overrideBadge}
                            </span>
                            {view.weaker_than_default ? (
                              <span className="rounded-md bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700">
                                {t.weakerThanDefaultBadge}
                              </span>
                            ) : null}
                          </span>
                        ) : (
                          <span className="text-xs text-slate-400">{t.defaultAppliedBadge}</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-xs text-slate-600">
                        {summarizeRules(view.effective_rules ?? [], t)}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <a
                          href={tenantURL(
                            `/admin/applications/${encodeURIComponent(application.application_id)}/edit`,
                          )}
                          className="text-xs font-medium text-blue-700 hover:underline"
                        >
                          {t.edit}
                        </a>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Card>
  )
}

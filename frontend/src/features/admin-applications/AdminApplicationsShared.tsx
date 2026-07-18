import {
  IconCheck,
  IconCopy,
  IconKey,
  IconLink,
  IconServer,
  IconWorldShare,
} from '@tabler/icons-react'
import { type ReactNode, useState } from 'react'
import { AuthenticationAPIError, tenantURL } from '../../api'
import { Button } from '../../components/ui/button'
import { Label } from '../../components/ui/label'
import type { SelectOption } from '../../components/ui/select'
import { safeApplicationIconURL } from '../../lib/applicationIcon'
import { useDictionary } from '../../lib/i18n'
import {
  adminApplicationsDictionary,
  type AdminApplicationsDictionary,
} from './AdminApplicationsPage.i18n'
import type {
  AdminApplication,
  RequiredAuthnStrength,
  SignInRule,
  WsFedTokenType,
} from '../../types'

export type AppType = 'oidc' | 'wsfed' | 'saml' | 'weblink' | 'service'

export const TOKEN_TYPE_SAML11: WsFedTokenType = 'urn:oasis:names:tc:SAML:1.0:assertion'
export const TOKEN_TYPE_SAML20: WsFedTokenType = 'urn:oasis:names:tc:SAML:2.0:assertion'

export const AUTH_METHODS: SelectOption[] = [
  { value: 'client_secret_basic', label: 'client_secret_basic' },
  { value: 'client_secret_post', label: 'client_secret_post' },
  { value: 'private_key_jwt', label: 'private_key_jwt' },
  { value: 'tls_client_auth', label: 'tls_client_auth' },
  { value: 'none', label: 'none (public)' },
]

export const DEFAULT_NAMEID_FORMAT = 'urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified'
// SAML 2.0 の既定 NameID 形式は persistent (Okta / Entra の既定運用に合わせる)。
export const SAML_DEFAULT_NAMEID_FORMAT = 'urn:oasis:names:tc:SAML:2.0:nameid-format:persistent'
export const DEFAULT_NAMEID_SOURCE = 'sub'

export function wsfedTokenTypeOptions(t: AdminApplicationsDictionary): SelectOption[] {
  return [
    { value: TOKEN_TYPE_SAML11, label: t.wsfedTokenTypeSaml11 },
    { value: TOKEN_TYPE_SAML20, label: t.wsfedTokenTypeSaml20 },
  ]
}

export function appTypeOptions(
  t: AdminApplicationsDictionary,
): { type: AppType; label: string; description: string; icon: typeof IconKey }[] {
  return [
    { type: 'oidc', label: t.oidcTypeLabel, description: t.oidcTypeDescription, icon: IconKey },
    {
      type: 'wsfed',
      label: t.wsfedTypeLabel,
      description: t.wsfedTypeDescription,
      icon: IconWorldShare,
    },
    {
      type: 'saml',
      label: t.samlTypeLabel,
      description: t.samlTypeDescription,
      icon: IconWorldShare,
    },
    {
      type: 'weblink',
      label: t.weblinkTypeLabel,
      description: t.weblinkTypeDescription,
      icon: IconLink,
    },
    {
      type: 'service',
      label: t.serviceTypeLabel,
      description: t.serviceTypeDescription,
      icon: IconServer,
    },
  ]
}

export function nameIdFormatOptions(t: AdminApplicationsDictionary): SelectOption[] {
  return [
    {
      value: 'urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified',
      label: t.nameIdFormatUnspecified,
    },
    {
      value: 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
      label: t.nameIdFormatEmail,
    },
    {
      value: 'urn:oasis:names:tc:SAML:2.0:nameid-format:persistent',
      label: t.nameIdFormatPersistent,
    },
  ]
}

export function statusOptions(t: AdminApplicationsDictionary): SelectOption[] {
  return [
    { value: 'active', label: t.statusActive },
    { value: 'disabled', label: t.statusDisabled },
  ]
}

export function signInStrengthOptions(t: AdminApplicationsDictionary): SelectOption[] {
  return [
    { value: 'Password', label: t.strengthPasswordLabel },
    { value: 'Mfa', label: t.strengthMfaLabel },
  ]
}

// summarizeSignInRule は 1 件のサインインルールを利用者向けの 1 行へ要約する。
// 内部ルール名は表示せず、テナントデフォルト・実効ポリシーの読み取り専用表示に用いる (wi-115, ADR-081)。
export function summarizeSignInRule(rule: SignInRule, t: AdminApplicationsDictionary): string {
  const parts: string[] = [
    rule.required_authn.strength === 'Mfa' ? t.strengthMfaLabel : t.strengthPasswordLabel,
  ]
  if (rule.condition.reauth_max_age_seconds) {
    parts.push(t.reauthSuffix.replace('{seconds}', String(rule.condition.reauth_max_age_seconds)))
  }
  const cidrs = rule.condition.network_allow_cidrs ?? []
  if (cidrs.length > 0) {
    parts.push(t.allowedNetworkPrefix.replace('{cidrs}', cidrs.join(', ')))
  }
  return parts.join(' / ')
}

// 編集中のアプリ個別ポリシー入力から表示用の SignInRule を組み立てる (ADR-081, 上書きモデル)。
export function appRuleFromInputs(
  strength: RequiredAuthnStrength,
  reauthText: string,
  cidrsText: string,
): SignInRule {
  const reauth = reauthText.trim() === '' ? undefined : Number.parseInt(reauthText.trim(), 10)
  const cidrs = cidrsText
    .split('\n')
    .map((entry) => entry.trim())
    .filter((entry) => entry !== '')
  return {
    rule_id: 'app-override',
    name: 'app-override',
    enabled: true,
    required_authn: { strength },
    condition: {
      reauth_max_age_seconds: reauth && Number.isFinite(reauth) && reauth > 0 ? reauth : undefined,
      network_allow_cidrs: cidrs.length > 0 ? cidrs : undefined,
    },
  }
}

// signInRuleWeakerThanDefault はアプリ個別ルールがデフォルトより弱いかの UI 用ヒント判定。
// 認証強度・再認証を求めるまでの時間・許可ネットワークの 3 項目で見る (サーバの判定と対応, ADR-081)。
export function signInRuleWeakerThanDefault(
  appRule: SignInRule,
  defaultRules: SignInRule[],
): boolean {
  const def = defaultRules.find((rule) => rule.enabled)
  if (!def) return false
  if (def.required_authn.strength === 'Mfa' && appRule.required_authn.strength !== 'Mfa') {
    return true
  }
  const defReauth = def.condition.reauth_max_age_seconds
  const appReauth = appRule.condition.reauth_max_age_seconds
  if (defReauth != null && (appReauth == null || appReauth > defReauth)) {
    return true
  }
  const defCIDRs = def.condition.network_allow_cidrs ?? []
  const appCIDRs = appRule.condition.network_allow_cidrs ?? []
  if (defCIDRs.length > 0) {
    if (appCIDRs.length === 0) return true
    if (appCIDRs.some((entry) => !defCIDRs.includes(entry))) return true
  }
  return false
}

export function listURL(): string {
  return tenantURL('/admin/applications')
}
export function detailURL(id: string): string {
  return tenantURL(`/admin/applications/${encodeURIComponent(id)}`)
}
export function editURL(id: string): string {
  return tenantURL(`/admin/applications/${encodeURIComponent(id)}/edit`)
}

export function messageOf(cause: unknown, fallback: string): string {
  return cause instanceof AuthenticationAPIError ? cause.message : fallback
}

// parseList は空白・カンマ・改行区切りの入力を一意な URL 配列へ正規化する。
export function parseList(value: string): string[] {
  return [
    ...new Set(
      value
        .split(/[\s,]+/)
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  ]
}

export function initials(name: string): string {
  return name.trim().slice(0, 2).toUpperCase() || '??'
}

export function AppIcon({ app, size = 'md' }: { app: AdminApplication; size?: 'sm' | 'md' }) {
  const dim = size === 'sm' ? 'size-9 text-xs' : 'size-11 text-sm'
  const iconURL = safeApplicationIconURL(app.icon_url)
  if (iconURL) {
    return <img src={iconURL} alt="" className={`${dim} rounded-lg object-cover`} />
  }
  return (
    <span
      className={`flex ${dim} items-center justify-center rounded-lg border border-blue-100 bg-blue-50 font-bold text-blue-700`}
    >
      {initials(app.name)}
    </span>
  )
}

export function StatusBadge({ status }: { status: AdminApplication['status'] }) {
  const t = useDictionary(adminApplicationsDictionary)
  const active = status === 'active'
  return (
    <span
      className={`rounded-md px-2 py-0.5 text-xs font-medium ${
        active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'
      }`}
    >
      {active ? t.statusActive : t.statusDisabled}
    </span>
  )
}

export function kindLabel(app: AdminApplication, t: AdminApplicationsDictionary): string {
  if (app.kind === 'weblink') return t.weblinkTypeLabel
  if (app.kind === 'service') return t.serviceTypeLabel
  const binding = app.bindings[0]?.type
  if (binding === 'wsfed') return t.wsfedTypeLabel
  if (binding === 'saml') return t.samlKindLabel
  if (binding === 'oidc') return t.oidcKindLabel
  return t.federationKindLabel
}

export function KindBadge({ app }: { app: AdminApplication }) {
  const t = useDictionary(adminApplicationsDictionary)
  return (
    <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
      {kindLabel(app, t)}
    </span>
  )
}

export function SectionTitle({ children }: { children: ReactNode }) {
  return <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">{children}</h3>
}

export function ReadOnlyField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-bold uppercase tracking-normal text-slate-400">{label}</dt>
      <dd className="mt-1 text-sm text-slate-700">{children}</dd>
    </div>
  )
}

// ReadonlyMeta は更新契約上の不変項目 (認証方式・クライアント種別・FAPI プロファイル) を
// 編集欄ではなく小さなラベル付きテキストで示し、「ここでは変えられない」ことを伝える。
export function ReadonlyMeta({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="font-semibold text-slate-500">{label}</p>
      <p className="mt-0.5 break-all font-mono text-slate-800">{value || '—'}</p>
    </div>
  )
}

export function UriList({ values }: { values: string[] }) {
  if (values.length === 0) return <span className="text-slate-400">—</span>
  return (
    <ul className="grid gap-1">
      {values.map((v) => (
        <li key={v} className="break-all font-mono text-xs text-slate-700">
          {v}
        </li>
      ))}
    </ul>
  )
}

// CopyableValue は変更できない値 (client_id / secret 等) を入力欄ではなくテキストとして
// 表示し、コピーボタンだけを添える。フォームに見せないことで「編集不可」を明示する。
export function CopyableValue({ value }: { value: string }) {
  const [copied, setCopied] = useState(false)
  const t = useDictionary(adminApplicationsDictionary)
  return (
    <div className="flex items-center gap-2">
      <code className="min-w-0 flex-1 break-all rounded-md bg-slate-50 px-3 py-2 font-mono text-xs text-slate-800">
        {value}
      </code>
      <Button
        type="button"
        variant="outline"
        className="size-9 shrink-0 px-0"
        aria-label={t.copyAria}
        onClick={() => {
          void navigator.clipboard?.writeText(value)
          setCopied(true)
          setTimeout(() => setCopied(false), 1500)
        }}
      >
        {copied ? (
          <IconCheck size={16} className="text-emerald-600" aria-hidden="true" />
        ) : (
          <IconCopy size={16} aria-hidden="true" />
        )}
      </Button>
    </div>
  )
}

export function CopyableField({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      <CopyableValue value={value} />
    </div>
  )
}

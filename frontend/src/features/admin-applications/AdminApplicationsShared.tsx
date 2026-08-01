import {
  IconCheck,
  IconCopy,
  IconKey,
  IconLink,
  IconPlus,
  IconServer,
  IconTrash,
  IconWorldShare,
} from '@tabler/icons-react'
import { type ReactNode, useEffect, useState } from 'react'
import { AuthenticationAPIError, getTenantUserAttributeSchema, tenantURL } from '../../api'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select, type SelectOption } from '../../components/ui/select'
import { safeApplicationIconURL } from '../../lib/applicationIcon'
import { useDictionary } from '../../lib/i18n'
import {
  adminApplicationsDictionary,
  type AdminApplicationsDictionary,
} from './AdminApplicationsPage.i18n'
import type {
  AdminApplication,
  AttrVisibility,
  RequiredAuthnStrength,
  SignInRule,
  UserAttributeDef,
  WsFedClaimMappingRule,
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
export function newURL(): string {
  return tenantURL('/admin/applications/new')
}
export function detailURL(id: string): string {
  return tenantURL(`/admin/applications/${encodeURIComponent(id)}`)
}
export function editURL(id: string): string {
  return tenantURL(`/admin/applications/${encodeURIComponent(id)}/edit`)
}
export function provisioningURL(id: string): string {
  return tenantURL(`/admin/applications/${encodeURIComponent(id)}/provisioning`)
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
  const protocol = app.protocol?.type
  if (protocol === 'wsfed') return t.wsfedTypeLabel
  if (protocol === 'saml') return t.samlKindLabel
  if (protocol === 'oidc') return t.oidcKindLabel
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

// coreAttributeKeys は User の型付きフィールドで、UserAttributeDef には現れないが常に
// releasable な source (backend/claimmapping/usecases/floor.go の coreAttributeKeys と同じ集合)。
const CORE_ATTRIBUTE_KEYS = [
  'sub',
  'preferred_username',
  'email',
  'email_verified',
  'name',
  'given_name',
  'family_name',
  'roles',
] as const

function coreAttributeLabel(key: string, t: AdminApplicationsDictionary): string {
  switch (key) {
    case 'sub':
      return t.coreAttributeSubLabel
    case 'preferred_username':
      return t.coreAttributePreferredUsernameLabel
    case 'email':
      return t.coreAttributeEmailLabel
    case 'email_verified':
      return t.coreAttributeEmailVerifiedLabel
    case 'name':
      return t.coreAttributeNameLabel
    case 'given_name':
      return t.coreAttributeGivenNameLabel
    case 'family_name':
      return t.coreAttributeFamilyNameLabel
    case 'roles':
      return t.coreAttributeRolesLabel
    default:
      return key
  }
}

export type ReleasableAttributeOption = SelectOption & { visibility?: AttrVisibility }

// useReleasableAttributes はテナントの属性定義 (組み込み + custom) を一度だけ取得し、
// (1) claim release preview 用の全定義 (Private を含む、非PII: key/type/visibility のみ)、
// (2) source 選択肢用の releasable 一覧 (User の core field + visibility != private) を返す
// (wi-73, ADR-151)。実際の利用者の値は取得・表示しない。
export function useReleasableAttributes(): {
  allDefs: UserAttributeDef[] | null
  options: ReleasableAttributeOption[]
  error: boolean
} {
  const t = useDictionary(adminApplicationsDictionary)
  const [defs, setDefs] = useState<UserAttributeDef[] | null>(null)
  const [error, setError] = useState(false)
  useEffect(() => {
    let cancelled = false
    getTenantUserAttributeSchema()
      .then((schema) => {
        if (cancelled) return
        setDefs([...schema.builtin, ...schema.attributes])
      })
      .catch(() => {
        if (!cancelled) setError(true)
      })
    return () => {
      cancelled = true
    }
  }, [])
  // 表示名を先頭にし、内部属性キー (sub 等、OIDC 由来の名前が SAML/WS-Fed 画面にも出る) は
  // 括弧書きの補足に留める。値そのものは変えない (プロトコル間で共有する内部表現)。
  const core: ReleasableAttributeOption[] = CORE_ATTRIBUTE_KEYS.map((key) => ({
    value: key,
    label: `${coreAttributeLabel(key, t)} (${key})`,
  }))
  const releasable: ReleasableAttributeOption[] = (defs ?? [])
    .filter((def) => def.visibility !== 'private')
    .map((def) => ({
      value: def.key,
      label: def.label ? `${def.label} (${def.key})` : def.key,
      visibility: def.visibility,
    }))
  return { allDefs: defs, options: [...core, ...releasable], error }
}

// ClaimReleaseAttributesPreview は claim release rule / NameID・sub source の候補を示す
// 非PII preview (wi-73, ADR-151)。実際の利用者の値は取得・表示せず、テナントの属性
// 定義 (key / type / visibility) だけを一覧する。visibility=private は fail-closed floor で
// 常に拒否されることを明示する。
export function ClaimReleaseAttributesPreview() {
  const t = useDictionary(adminApplicationsDictionary)
  const { allDefs: defs, error } = useReleasableAttributes()
  if (error) return null
  return (
    <details className="rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs">
      <summary className="cursor-pointer font-semibold text-slate-600">
        {t.claimReleasePreviewHeading}
      </summary>
      <p className="mt-2 text-slate-500">{t.claimReleasePreviewHelp}</p>
      {defs ? (
        <table className="mt-2 w-full text-left">
          <thead>
            <tr className="text-slate-400">
              <th className="py-1 pr-2 font-semibold">{t.attributeKeyColumnLabel}</th>
              <th className="py-1 pr-2 font-semibold">{t.attributeTypeColumnLabel}</th>
              <th className="py-1 font-semibold">{t.attributeVisibilityColumnLabel}</th>
            </tr>
          </thead>
          <tbody>
            {defs.map((def) => (
              <tr key={def.key} className="border-t border-slate-200">
                <td className="py-1 pr-2 font-mono">{def.key}</td>
                <td className="py-1 pr-2 text-slate-500">{def.type}</td>
                <td
                  className={
                    def.visibility === 'private'
                      ? 'py-1 font-semibold text-red-600'
                      : 'py-1 text-slate-600'
                  }
                >
                  {visibilityLabel(def.visibility, t)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </details>
  )
}

function visibilityLabel(visibility: AttrVisibility, t: AdminApplicationsDictionary): string {
  switch (visibility) {
    case 'private':
      return t.visibilityPrivate
    case 'self_readable':
      return t.visibilitySelfReadable
    case 'admin_readable':
      return t.visibilityAdminReadable
    case 'claim_exposed':
      return t.visibilityClaimExposed
    default:
      return visibility
  }
}

// SourceAttributeSelect は NameID / sub の source 属性を選ぶ。releasable な属性 (User の
// core field + visibility != private) だけを選択肢にする (wi-73, ADR-151)。既存値が一覧に
// 無い場合 (未取得中・tenant 定義から外れた値) も選択肢に含め、値を失わないようにする。
export function SourceAttributeSelect({
  value,
  onChange,
  id,
}: {
  value: string
  onChange: (value: string) => void
  id?: string
}) {
  const { options } = useReleasableAttributes()
  const selectOptions =
    value && !options.some((option) => option.value === value)
      ? [...options, { value, label: value }]
      : options
  return (
    <Select
      id={id}
      value={value}
      onValueChange={onChange}
      options={selectOptions}
      className="w-full"
    />
  )
}

function claimRuleSourceOptions(t: AdminApplicationsDictionary): SelectOption[] {
  return [
    { value: 'user_attribute', label: t.claimRuleSourceUserAttributeLabel },
    { value: 'fixed', label: t.claimRuleSourceFixedLabel },
    { value: 'nameid', label: t.claimRuleSourceNameIdLabel },
  ]
}

// ClaimRulesEditor は claim release 上書き rule の構造化エディタ (wi-73, ADR-151)。
// 生 JSON を書かせる代わりに、claim 名の入力・source の選択・(user_attribute のときの)
// releasable な属性選択・(fixed のときの) 固定値入力・required チェックボックスを行単位で
// 提供する。source_key は floor (visibility != private) を満たす属性だけから選べるため、
// 保存前に無効な値を作りにくい。
export function ClaimRulesEditor({
  rules,
  onChange,
  t,
}: {
  rules: WsFedClaimMappingRule[]
  onChange: (rules: WsFedClaimMappingRule[]) => void
  t: AdminApplicationsDictionary
}) {
  function updateRule(index: number, patch: Partial<WsFedClaimMappingRule>) {
    onChange(rules.map((rule, i) => (i === index ? { ...rule, ...patch } : rule)))
  }
  function removeRule(index: number) {
    onChange(rules.filter((_, i) => i !== index))
  }
  function addRule() {
    onChange([...rules, { claim_type: '', source: 'user_attribute', source_key: '' }])
  }
  return (
    <div className="grid gap-2">
      {rules.length === 0 ? (
        <p className="text-xs text-slate-400">{t.claimRulesEmptyNotice}</p>
      ) : (
        <div className="grid gap-2">
          {rules.map((rule, index) => (
            <div
              // biome-ignore lint/suspicious/noArrayIndexKey: rows have no stable identity until saved
              key={index}
              className="grid gap-2 rounded-lg border border-slate-200 p-2 sm:grid-cols-[1fr_auto] sm:items-end"
            >
              <div className="grid gap-2 sm:grid-cols-4">
                <div className="grid gap-1">
                  <Label className="text-[11px] font-semibold text-slate-500">
                    {t.claimTypeFieldLabel}
                  </Label>
                  <Input
                    value={rule.claim_type}
                    onChange={(e) => updateRule(index, { claim_type: e.target.value })}
                    className="font-mono text-xs"
                    placeholder="department"
                  />
                </div>
                <div className="grid gap-1">
                  <Label className="text-[11px] font-semibold text-slate-500">
                    {t.claimRuleSourceFieldLabel}
                  </Label>
                  <Select
                    value={rule.source}
                    onValueChange={(v) =>
                      updateRule(index, { source: v as WsFedClaimMappingRule['source'] })
                    }
                    options={claimRuleSourceOptions(t)}
                    className="w-full"
                  />
                </div>
                {rule.source === 'user_attribute' ? (
                  <div className="grid gap-1">
                    <Label className="text-[11px] font-semibold text-slate-500">
                      {t.claimRuleSourceKeyFieldLabel}
                    </Label>
                    <SourceAttributeSelect
                      value={rule.source_key ?? ''}
                      onChange={(v) => updateRule(index, { source_key: v })}
                    />
                  </div>
                ) : null}
                {rule.source === 'fixed' ? (
                  <div className="grid gap-1">
                    <Label className="text-[11px] font-semibold text-slate-500">
                      {t.claimRuleFixedValueFieldLabel}
                    </Label>
                    <Input
                      value={rule.fixed_value ?? ''}
                      onChange={(e) => updateRule(index, { fixed_value: e.target.value })}
                      className="font-mono text-xs"
                    />
                  </div>
                ) : null}
                <label className="flex items-center gap-2 pb-1.5 text-xs font-medium text-slate-700">
                  <input
                    type="checkbox"
                    checked={rule.required ?? false}
                    onChange={(e) => updateRule(index, { required: e.target.checked })}
                    className="size-4"
                  />
                  {t.claimRuleRequiredLabel}
                </label>
              </div>
              <Button
                type="button"
                variant="outline"
                onClick={() => removeRule(index)}
                aria-label={t.claimRuleRemoveAria}
              >
                <IconTrash size={16} aria-hidden="true" />
              </Button>
            </div>
          ))}
        </div>
      )}
      <Button type="button" variant="outline" onClick={addRule}>
        <IconPlus size={16} aria-hidden="true" />
        {t.claimRuleAddButton}
      </Button>
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

import {
  IconApps,
  IconArrowLeft,
  IconCheck,
  IconCopy,
  IconExternalLink,
  IconKey,
  IconLink,
  IconPencil,
  IconPlus,
  IconRefresh,
  IconServer,
  IconTrash,
  IconUserPlus,
  IconWorldShare,
  IconX,
} from '@tabler/icons-react'
import { type FormEvent, type ReactNode, useEffect, useMemo, useRef, useState } from 'react'
import {
  assignApplication,
  AuthenticationAPIError,
  createAdminApplication,
  createApplicationCategory,
  deleteAdminApplication,
  deleteApplicationIcon,
  deleteApplicationCategory,
  listAdminApplications,
  listAdminGroups,
  listAdminUsers,
  listApplicationAssignments,
  listApplicationCategories,
  setApplicationCategories,
  tenantURL,
  unassignApplication,
  updateAdminApplication,
  updateApplicationOidcConfig,
  updateAppSignInPolicy,
  updateApplicationSamlConfig,
  updateApplicationWsFedConfig,
  uploadApplicationIcon,
} from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select, type SelectOption } from '../../components/ui/select'
import {
  MAX_APPLICATION_ICON_BYTES,
  safeApplicationIconURL,
  validateApplicationIconFile,
} from '../../lib/applicationIcon'
import { useDictionary, useLocale } from '../../lib/i18n'
import {
  adminApplicationsDictionary,
  type AdminApplicationsDictionary,
} from './AdminApplicationsPage.i18n'
import type {
  AdminApplication,
  AdminApplicationDetail,
  AdminGroup,
  AdminUser,
  ApplicationAssignment,
  ApplicationCategory,
  ApplicationStatus,
  RequiredAuthnStrength,
  SignInRule,
  WsFedClaimMappingRule,
  WsFedTokenType,
} from '../../types'

type AppType = 'oidc' | 'wsfed' | 'saml' | 'weblink' | 'service'

const TOKEN_TYPE_SAML11: WsFedTokenType = 'urn:oasis:names:tc:SAML:1.0:assertion'
const TOKEN_TYPE_SAML20: WsFedTokenType = 'urn:oasis:names:tc:SAML:2.0:assertion'

function wsfedTokenTypeOptions(t: AdminApplicationsDictionary): SelectOption[] {
  return [
    { value: TOKEN_TYPE_SAML11, label: t.wsfedTokenTypeSaml11 },
    { value: TOKEN_TYPE_SAML20, label: t.wsfedTokenTypeSaml20 },
  ]
}

function appTypeOptions(
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

function nameIdFormatOptions(t: AdminApplicationsDictionary): SelectOption[] {
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

function statusOptions(t: AdminApplicationsDictionary): SelectOption[] {
  return [
    { value: 'active', label: t.statusActive },
    { value: 'disabled', label: t.statusDisabled },
  ]
}

function signInStrengthOptions(t: AdminApplicationsDictionary): SelectOption[] {
  return [
    { value: 'Password', label: t.strengthPasswordLabel },
    { value: 'Mfa', label: t.strengthMfaLabel },
  ]
}

// summarizeSignInRule は 1 件のサインインルールを利用者向けの 1 行へ要約する。
// 内部ルール名は表示せず、テナントデフォルト・実効ポリシーの読み取り専用表示に用いる (wi-115, ADR-081)。
function summarizeSignInRule(rule: SignInRule, t: AdminApplicationsDictionary): string {
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
function appRuleFromInputs(
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
function signInRuleWeakerThanDefault(appRule: SignInRule, defaultRules: SignInRule[]): boolean {
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

// OIDC client の token endpoint 認証方式。作成時に確定し以後不変。
const AUTH_METHODS: SelectOption[] = [
  { value: 'client_secret_basic', label: 'client_secret_basic' },
  { value: 'client_secret_post', label: 'client_secret_post' },
  { value: 'private_key_jwt', label: 'private_key_jwt' },
  { value: 'tls_client_auth', label: 'tls_client_auth' },
  { value: 'none', label: 'none (public)' },
]

const DEFAULT_NAMEID_FORMAT = 'urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified'
// SAML 2.0 の既定 NameID 形式は persistent (Okta / Entra の既定運用に合わせる)。
const SAML_DEFAULT_NAMEID_FORMAT = 'urn:oasis:names:tc:SAML:2.0:nameid-format:persistent'
const DEFAULT_NAMEID_SOURCE = 'sub'

function listURL(): string {
  return tenantURL('/admin/applications')
}
function detailURL(id: string): string {
  return tenantURL(`/admin/applications/${encodeURIComponent(id)}`)
}
function editURL(id: string): string {
  return tenantURL(`/admin/applications/${encodeURIComponent(id)}/edit`)
}

function messageOf(cause: unknown, fallback: string): string {
  return cause instanceof AuthenticationAPIError ? cause.message : fallback
}

// parseList は空白・カンマ・改行区切りの入力を一意な URL 配列へ正規化する。
function parseList(value: string): string[] {
  return [
    ...new Set(
      value
        .split(/[\s,]+/)
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  ]
}

function initials(name: string): string {
  return name.trim().slice(0, 2).toUpperCase() || '??'
}

function AppIcon({ app, size = 'md' }: { app: AdminApplication; size?: 'sm' | 'md' }) {
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

function StatusBadge({ status }: { status: AdminApplication['status'] }) {
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

function kindLabel(app: AdminApplication, t: AdminApplicationsDictionary): string {
  if (app.kind === 'weblink') return t.weblinkTypeLabel
  if (app.kind === 'service') return t.serviceTypeLabel
  const binding = app.bindings[0]?.type
  if (binding === 'wsfed') return t.wsfedTypeLabel
  if (binding === 'saml') return t.samlKindLabel
  if (binding === 'oidc') return t.oidcKindLabel
  return t.federationKindLabel
}

function KindBadge({ app }: { app: AdminApplication }) {
  const t = useDictionary(adminApplicationsDictionary)
  return (
    <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
      {kindLabel(app, t)}
    </span>
  )
}

function SectionTitle({ children }: { children: ReactNode }) {
  return <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">{children}</h3>
}

function ReadOnlyField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-bold uppercase tracking-normal text-slate-400">{label}</dt>
      <dd className="mt-1 text-sm text-slate-700">{children}</dd>
    </div>
  )
}

// ReadonlyMeta は更新契約上の不変項目 (認証方式・クライアント種別・FAPI プロファイル) を
// 編集欄ではなく小さなラベル付きテキストで示し、「ここでは変えられない」ことを伝える。
function ReadonlyMeta({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="font-semibold text-slate-500">{label}</p>
      <p className="mt-0.5 break-all font-mono text-slate-800">{value || '—'}</p>
    </div>
  )
}

function UriList({ values }: { values: string[] }) {
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
function CopyableValue({ value }: { value: string }) {
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

function CopyableField({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      <CopyableValue value={value} />
    </div>
  )
}

// ===========================================================================
// 一覧画面
// ===========================================================================

export function AdminApplicationsPage({
  csrfToken,
  actorUsername,
  applications: initial,
}: {
  csrfToken: string
  actorUsername?: string
  applications: AdminApplication[]
}) {
  const [applications, setApplications] = useState(initial)
  const [selectedID, setSelectedID] = useState<string>(() => initial[0]?.application_id ?? '')
  const [showCreate, setShowCreate] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminApplicationsDictionary)

  const selected = applications.find((a) => a.application_id === selectedID) ?? null

  async function refresh(preferredID = selectedID) {
    const next = await listAdminApplications()
    setApplications(next)
    setSelectedID(
      next.find((a) => a.application_id === preferredID)?.application_id ??
        next[0]?.application_id ??
        '',
    )
  }

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(success)
    } catch (cause) {
      setError(messageOf(cause, t.genericOpError))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="applications"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <>
          <Button
            variant="outline"
            className="size-9 px-0"
            aria-label={t.reloadAriaLabel}
            onClick={() => run(() => refresh(), t.listRefreshedNotice)}
            disabled={busy}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
          <Button onClick={() => setShowCreate(true)} disabled={busy}>
            <IconPlus size={16} aria-hidden="true" />
            {t.addApplication}
          </Button>
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,420px)]">
        <Card className="overflow-hidden">
          {applications.length === 0 ? (
            <div className="flex min-h-48 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
              <IconApps size={28} className="text-slate-300" aria-hidden="true" />
              <p className="mt-3">{t.emptyApplicationsNotice}</p>
            </div>
          ) : (
            <ul>
              {applications.map((app) => (
                <li key={app.application_id}>
                  <button
                    type="button"
                    onClick={() => setSelectedID(app.application_id)}
                    className={`flex w-full items-center gap-3 border-t border-slate-100 px-4 py-3 text-left first:border-t-0 hover:bg-slate-50 ${
                      selectedID === app.application_id ? 'bg-blue-50/60' : ''
                    }`}
                  >
                    <AppIcon app={app} size="sm" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate font-semibold text-slate-900">{app.name}</span>
                        <StatusBadge status={app.status} />
                      </div>
                      <div className="mt-0.5">
                        <KindBadge app={app} />
                      </div>
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <ApplicationSummaryCard
          key={selectedID || 'none'}
          app={selected}
          busy={busy}
          onDelete={(id) =>
            run(async () => {
              await deleteAdminApplication(csrfToken, id)
              await refresh()
            }, t.applicationDeletedNotice)
          }
        />
      </div>

      {showCreate ? (
        <CreateApplicationDialog
          csrfToken={csrfToken}
          onClose={() => setShowCreate(false)}
          onCreated={(id) => {
            window.location.assign(detailURL(id))
          }}
        />
      ) : null}
    </AdminShell>
  )
}

function ApplicationSummaryCard({
  app,
  busy,
  onDelete,
}: {
  app: AdminApplication | null
  busy: boolean
  onDelete: (id: string) => void
}) {
  const [confirmDelete, setConfirmDelete] = useState(false)
  const t = useDictionary(adminApplicationsDictionary)
  const { locale } = useLocale()

  if (!app) {
    return (
      <Card className="flex min-h-48 items-center justify-center p-6 text-sm text-slate-500">
        {t.selectApplicationPrompt}
      </Card>
    )
  }

  return (
    <Card className="overflow-hidden">
      <div className="border-b border-slate-200 p-5">
        <div className="flex items-start gap-3">
          <AppIcon app={app} />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-lg font-semibold text-slate-950">{app.name}</h2>
              <StatusBadge status={app.status} />
            </div>
            <div className="mt-1">
              <KindBadge app={app} />
            </div>
          </div>
        </div>
        <div className="mt-4">
          <AdminPaneActions
            detailHref={detailURL(app.application_id)}
            busy={busy}
            actions={[
              {
                label: t.deleteApplication,
                icon: IconTrash,
                onClick: () => setConfirmDelete(true),
                tone: 'danger',
              },
            ]}
          />
        </div>
      </div>
      {confirmDelete ? (
        <Alert
          variant="destructive"
          className="m-5 flex flex-wrap items-center justify-between gap-2"
        >
          <span>{t.confirmDeleteAppPrompt}</span>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setConfirmDelete(false)} disabled={busy}>
              {t.dismissConfirm}
            </Button>
            <Button
              variant="destructive"
              disabled={busy}
              onClick={() => onDelete(app.application_id)}
            >
              <IconTrash size={14} aria-hidden="true" />
              {t.confirmDelete}
            </Button>
          </div>
        </Alert>
      ) : null}
      <dl className="grid gap-4 p-5">
        <ReadOnlyField label={t.kindFieldLabel}>{kindLabel(app, t)}</ReadOnlyField>

        <ReadOnlyField label={t.statusFieldLabel}>
          <StatusBadge status={app.status} />
        </ReadOnlyField>

        <ReadOnlyField label={t.categoryFieldLabel}>
          {app.category_names && app.category_names.length > 0 ? (
            <div className="flex flex-wrap gap-1">
              {app.category_names.map((name) => (
                <span key={name} className="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700">
                  {name}
                </span>
              ))}
            </div>
          ) : (
            <span className="text-slate-400">{t.noCategoryNotice}</span>
          )}
        </ReadOnlyField>

        <ReadOnlyField label={t.bindingFieldLabel}>
          {app.binding_summaries && app.binding_summaries.length > 0 ? (
            <div className="flex flex-col gap-1 font-mono text-xs text-slate-700">
              {app.binding_summaries.map((summary, idx) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: static list
                <span key={idx}>{summary}</span>
              ))}
            </div>
          ) : (
            <span className="text-slate-400">{t.notSetLabel}</span>
          )}
        </ReadOnlyField>

        <ReadOnlyField label={t.assignmentStatusFieldLabel}>
          {app.assigned_subject_count > 0 ? (
            <span className="text-slate-700">
              {t.assignedCount.replace('{count}', String(app.assigned_subject_count))}
            </span>
          ) : (
            <span className="text-slate-400">{t.noAssignmentNotice}</span>
          )}
        </ReadOnlyField>

        <ReadOnlyField label={t.signInPolicyFieldLabel}>
          {app.sign_in_policy_summary ? (
            <span className="text-slate-700">{app.sign_in_policy_summary}</span>
          ) : (
            <span className="text-slate-400">{t.notSetLabel}</span>
          )}
        </ReadOnlyField>

        {app.kind === 'service' ? (
          <ReadOnlyField label={t.serviceDescriptionFieldLabel}>
            <p className="text-xs text-slate-500">{t.serviceM2mDescription}</p>
          </ReadOnlyField>
        ) : (
          <ReadOnlyField label={t.launchUrlFieldLabel}>
            {app.launch_url ? (
              <a
                href={app.launch_url}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 break-all font-mono text-xs text-blue-700 hover:underline"
              >
                {app.launch_url}
                <IconExternalLink size={13} aria-hidden="true" />
              </a>
            ) : (
              <span className="text-slate-400">{t.notSetLabel}</span>
            )}
          </ReadOnlyField>
        )}

        <ReadOnlyField label={t.registeredUpdatedFieldLabel}>
          <div className="text-xs text-slate-500">
            <div>
              {t.registeredLabel.replace(
                '{date}',
                app.created_at
                  ? new Date(app.created_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
                  : t.unknownDate,
              )}
            </div>
            <div className="mt-0.5">
              {t.updatedLabel.replace(
                '{date}',
                app.updated_at
                  ? new Date(app.updated_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
                  : t.unknownDate,
              )}
            </div>
          </div>
        </ReadOnlyField>
      </dl>
    </Card>
  )
}

// ===========================================================================
// 詳細画面 (read-only)
// ===========================================================================

export function AdminApplicationDetailPage({
  csrfToken,
  actorUsername,
  detail,
}: {
  csrfToken: string
  actorUsername?: string
  detail: AdminApplicationDetail
}) {
  const app = detail.application
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminApplicationsDictionary)
  const { locale } = useLocale()

  async function handleDelete() {
    setBusy(true)
    setError('')
    try {
      await deleteAdminApplication(csrfToken, app.application_id)
      window.location.assign(listURL())
    } catch (cause) {
      setError(messageOf(cause, t.applicationDeleteFailedError))
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="applications"
      actorUsername={actorUsername}
      title={app.name}
      description={kindLabel(app, t)}
      actions={
        <div className="flex items-center gap-2">
          <a
            href={listURL()}
            className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
          >
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.backToList}
          </a>
          <Button asChild>
            <a href={editURL(app.application_id)}>
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </a>
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={busy}
            onClick={() => setConfirmDelete(true)}
          >
            <IconTrash size={16} aria-hidden="true" />
            {t.delete}
          </Button>
        </div>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {confirmDelete ? (
        <Alert variant="destructive" className="flex flex-wrap items-center justify-between gap-2">
          <span>{t.confirmDeleteAppPrompt}</span>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setConfirmDelete(false)} disabled={busy}>
              {t.dismissConfirm}
            </Button>
            <Button variant="destructive" disabled={busy} onClick={() => void handleDelete()}>
              <IconTrash size={14} aria-hidden="true" />
              {t.confirmDelete}
            </Button>
          </div>
        </Alert>
      ) : null}

      <div className="grid max-w-3xl gap-6">
        <Card className="overflow-hidden">
          <div className="flex items-start gap-3 border-b border-slate-200 p-5">
            <AppIcon app={app} />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h2 className="truncate text-lg font-semibold text-slate-950">{app.name}</h2>
                <StatusBadge status={app.status} />
              </div>
              <div className="mt-1">
                <KindBadge app={app} />
              </div>
            </div>
          </div>

          <div className="grid gap-6 p-5">
            {/* 基本情報セクション */}
            <section className="grid gap-4 sm:grid-cols-2">
              <ReadOnlyField label={t.kindFieldLabel}>
                <span>{kindLabel(app, t)}</span>
              </ReadOnlyField>
              <ReadOnlyField label={t.statusFieldLabel}>
                <StatusBadge status={app.status} />
              </ReadOnlyField>
              <ReadOnlyField label={t.categoryFieldLabel}>
                {app.category_names && app.category_names.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {app.category_names.map((name) => (
                      <span
                        key={name}
                        className="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700"
                      >
                        {name}
                      </span>
                    ))}
                  </div>
                ) : (
                  <span className="text-slate-400">{t.noCategoryNotice}</span>
                )}
              </ReadOnlyField>
              {app.kind !== 'service' ? (
                <ReadOnlyField label={t.launchUrlFieldLabel}>
                  {app.launch_url ? (
                    <a
                      href={app.launch_url}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 break-all font-mono text-xs text-blue-700 hover:underline"
                    >
                      {app.launch_url}
                      <IconExternalLink size={13} aria-hidden="true" />
                    </a>
                  ) : (
                    <span className="text-slate-400">{t.notSetLabel}</span>
                  )}
                </ReadOnlyField>
              ) : null}
            </section>

            <section className="grid gap-4 border-t border-slate-100 pt-5 sm:grid-cols-2 text-xs text-slate-500">
              <ReadOnlyField label={t.registeredAtFieldLabel}>
                <span>
                  {app.created_at
                    ? new Date(app.created_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
                    : t.unknownDate}
                </span>
              </ReadOnlyField>
              <ReadOnlyField label={t.lastUpdatedAtFieldLabel}>
                <span>
                  {app.updated_at
                    ? new Date(app.updated_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
                    : t.unknownDate}
                </span>
              </ReadOnlyField>
            </section>

            {detail.oidc ? (
              <section className="grid gap-3 border-t border-slate-100 pt-5 first:border-t-0 first:pt-0">
                <div className="flex items-center gap-2">
                  <IconKey size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>
                    {app.kind === 'service' ? t.serviceKindSectionHeading : t.oidcSectionHeading}
                  </SectionTitle>
                </div>
                <CopyableField label={t.clientIdFieldLabel} value={detail.oidc.client_id} />
                {app.kind !== 'service' ? (
                  <ReadOnlyField label={t.redirectUriFieldLabel}>
                    <UriList values={detail.oidc.redirect_uris} />
                  </ReadOnlyField>
                ) : null}
                <ReadOnlyField label={t.scopeFieldLabel}>
                  <span className="font-mono text-xs">{detail.oidc.scope || '—'}</span>
                </ReadOnlyField>
                <ReadOnlyField label={t.grantTypesFieldLabel}>
                  <span className="font-mono text-xs">
                    {detail.oidc.grant_types.join(', ') || '—'}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.responseTypesFieldLabel}>
                  <span className="font-mono text-xs">
                    {detail.oidc.response_types.join(', ') || '—'}
                  </span>
                </ReadOnlyField>
                <div className="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs sm:grid-cols-3">
                  <ReadonlyMeta label={t.clientTypeMetaLabel} value={detail.oidc.client_type} />
                  <ReadonlyMeta
                    label={t.authMethodMetaLabel}
                    value={detail.oidc.token_endpoint_auth_method}
                  />
                  <ReadonlyMeta label={t.fapiProfileMetaLabel} value={detail.oidc.fapi_profile} />
                </div>
                <ReadOnlyField label={t.securityFieldLabel}>
                  <span className="text-xs text-slate-700">
                    {[
                      detail.oidc.require_pushed_authorization_requests ? t.parRequired : '',
                      detail.oidc.dpop_bound_access_tokens ? t.dpopBound : '',
                    ]
                      .filter(Boolean)
                      .join(', ') || t.standardSecurity}
                  </span>
                </ReadOnlyField>
                {app.kind === 'service' ? (
                  <p className="text-xs text-slate-500">{t.m2mNoLoginNotice}</p>
                ) : null}
              </section>
            ) : null}

            {detail.wsfed ? (
              <section className="grid gap-3 border-t border-slate-100 pt-5">
                <div className="flex items-center gap-2">
                  <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.wsFedSectionHeading}</SectionTitle>
                </div>
                <CopyableField label={t.wtrealmFieldLabel} value={detail.wsfed.wtrealm} />
                <ReadOnlyField label={t.replyUrlFieldLabel}>
                  <UriList values={detail.wsfed.reply_urls} />
                </ReadOnlyField>
                <ReadOnlyField label={t.nameIdFormatFieldLabel}>
                  <span className="break-all font-mono text-xs">
                    {nameIdFormatOptions(t).find((f) => f.value === detail.wsfed?.name_id_format)
                      ?.label ?? detail.wsfed.name_id_format}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.nameIdSourceFieldLabel}>
                  <span className="font-mono text-xs">{detail.wsfed.name_id_source}</span>
                </ReadOnlyField>
                <ReadOnlyField label={t.audienceFieldLabel}>
                  <span className="font-mono text-xs">
                    {detail.wsfed.audience ||
                      t.audienceDefaultSuffix.replace('{value}', detail.wsfed.wtrealm)}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.tokenTypeFieldLabel}>
                  <span className="text-xs">
                    {wsfedTokenTypeOptions(t).find((opt) => opt.value === detail.wsfed?.token_type)
                      ?.label ?? detail.wsfed.token_type}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.claimMappingRulesFieldLabel}>
                  {detail.wsfed.rules.length === 0 ? (
                    <span className="text-xs text-slate-400">{t.nameIdOnlyNotice}</span>
                  ) : (
                    <ul className="flex flex-wrap gap-1.5">
                      {detail.wsfed.rules.map((rule) => (
                        <li
                          key={rule.claim_type}
                          className="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs text-slate-700"
                        >
                          {rule.claim_type.split('/').pop()}
                          {rule.required ? '*' : ''}
                        </li>
                      ))}
                    </ul>
                  )}
                </ReadOnlyField>
              </section>
            ) : null}

            {detail.saml ? (
              <section className="grid gap-3 border-t border-slate-100 pt-5">
                <div className="flex items-center gap-2">
                  <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.samlSectionHeading}</SectionTitle>
                </div>
                <CopyableField label={t.entityIdFieldLabel} value={detail.saml.entity_id} />
                <ReadOnlyField label={t.acsUrlFieldLabel}>
                  <UriList values={detail.saml.acs_urls} />
                </ReadOnlyField>
                <ReadOnlyField label={t.sloUrlFieldLabel}>
                  <span className="break-all font-mono text-xs">{detail.saml.slo_url || '—'}</span>
                </ReadOnlyField>
                <ReadOnlyField label={t.nameIdFormatFieldLabel}>
                  <span className="break-all font-mono text-xs">
                    {nameIdFormatOptions(t).find((f) => f.value === detail.saml?.name_id_format)
                      ?.label ?? detail.saml.name_id_format}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.nameIdSourceFieldLabel}>
                  <span className="font-mono text-xs">{detail.saml.name_id_source}</span>
                </ReadOnlyField>
                <ReadOnlyField label={t.audienceFieldLabel}>
                  <span className="font-mono text-xs">
                    {detail.saml.audience ||
                      t.audienceDefaultSuffix.replace('{value}', detail.saml.entity_id)}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.signatureFieldLabel}>
                  <span className="text-xs">
                    {[
                      detail.saml.sign_assertion ? t.assertionSigned : '',
                      detail.saml.sign_response ? t.responseSigned : '',
                    ]
                      .filter(Boolean)
                      .join(' / ') || t.noSignature}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.requestSignatureVerificationFieldLabel}>
                  <span className="text-xs">
                    {detail.saml.want_authn_requests_signed
                      ? t.authnRequestSignatureRequired
                      : t.optional}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.claimMappingRulesFieldLabel}>
                  {detail.saml.rules.length === 0 ? (
                    <span className="text-xs text-slate-400">{t.nameIdOnlyNotice}</span>
                  ) : (
                    <ul className="flex flex-wrap gap-1.5">
                      {detail.saml.rules.map((rule) => (
                        <li
                          key={rule.claim_type}
                          className="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs text-slate-700"
                        >
                          {rule.claim_type.split('/').pop()}
                          {rule.required ? '*' : ''}
                        </li>
                      ))}
                    </ul>
                  )}
                </ReadOnlyField>
              </section>
            ) : null}

            {/* ログインポリシーセクション */}
            <section className="grid gap-3 border-t border-slate-100 pt-5">
              <div className="flex items-center gap-2">
                <IconKey size={16} className="text-slate-400" aria-hidden="true" />
                <SectionTitle>{t.signInPolicySectionHeading}</SectionTitle>
              </div>
              <ReadOnlyField label={t.applicationStatusFieldLabel}>
                <span className="text-slate-700 font-semibold">{app.sign_in_policy_summary}</span>
              </ReadOnlyField>
              {detail.sign_in_policy && detail.sign_in_policy.effective_rules.length > 0 ? (
                <ReadOnlyField label={t.appliedRulesFieldLabel}>
                  <ul className="mt-1 flex flex-col gap-1.5">
                    {detail.sign_in_policy.effective_rules.map((rule) => (
                      <li
                        key={rule.rule_id}
                        className="rounded-lg border border-slate-200 bg-slate-50 p-3 font-mono text-xs text-slate-700"
                      >
                        <div className="font-sans font-semibold mb-1">
                          {rule.name || t.ruleNameFallback.replace('{id}', rule.rule_id)}
                        </div>
                        {summarizeSignInRule(rule, t)}
                      </li>
                    ))}
                  </ul>
                </ReadOnlyField>
              ) : (
                <ReadOnlyField label={t.appliedRulesFieldLabel}>
                  <span className="text-xs text-slate-400">{t.noAppliedRulesNotice}</span>
                </ReadOnlyField>
              )}
            </section>

            {app.kind !== 'service' ? (
              <section className="grid gap-3 border-t border-slate-100 pt-5">
                <SectionTitle>{t.assignmentsHeading}</SectionTitle>
                <AssignmentList appID={app.application_id} onError={setError} />
              </section>
            ) : null}
          </div>
        </Card>
      </div>
    </AdminShell>
  )
}

// ===========================================================================
// 編集画面 (基本情報・プロトコル設定・割り当て)
// ===========================================================================

export function AdminApplicationEditPage({
  csrfToken,
  actorUsername,
  detail,
}: {
  csrfToken: string
  actorUsername?: string
  detail: AdminApplicationDetail
}) {
  const app = detail.application
  const [name, setName] = useState(app.name)
  const [iconFile, setIconFile] = useState<File | null>(null)
  const [iconPreviewURL, setIconPreviewURL] = useState('')
  const [removeIcon, setRemoveIcon] = useState(false)
  const iconSelectionToken = useRef(0)
  const [launchURL, setLaunchURL] = useState(app.launch_url ?? '')
  const [status, setStatus] = useState<ApplicationStatus>(app.status)
  const [redirects, setRedirects] = useState((detail.oidc?.redirect_uris ?? []).join('\n'))
  const [scope, setScope] = useState(detail.oidc?.scope ?? '')
  const [grantTypes, setGrantTypes] = useState((detail.oidc?.grant_types ?? []).join(', '))
  const [responseTypes, setResponseTypes] = useState((detail.oidc?.response_types ?? []).join(', '))
  const [requirePAR, setRequirePAR] = useState(
    detail.oidc?.require_pushed_authorization_requests ?? false,
  )
  const [dpopBound, setDpopBound] = useState(detail.oidc?.dpop_bound_access_tokens ?? false)
  const [replies, setReplies] = useState((detail.wsfed?.reply_urls ?? []).join('\n'))
  const [audience, setAudience] = useState(detail.wsfed?.audience ?? '')
  const [tokenType, setTokenType] = useState<WsFedTokenType>(
    detail.wsfed?.token_type || TOKEN_TYPE_SAML11,
  )
  const [nameIDFormat, setNameIDFormat] = useState(
    detail.wsfed?.name_id_format || DEFAULT_NAMEID_FORMAT,
  )
  const [nameIDSource, setNameIDSource] = useState(
    detail.wsfed?.name_id_source || DEFAULT_NAMEID_SOURCE,
  )
  const [rulesJSON, setRulesJSON] = useState(JSON.stringify(detail.wsfed?.rules ?? [], null, 2))
  const [samlACS, setSamlACS] = useState((detail.saml?.acs_urls ?? []).join('\n'))
  const [samlSLO, setSamlSLO] = useState(detail.saml?.slo_url ?? '')
  const [samlAudience, setSamlAudience] = useState(detail.saml?.audience ?? '')
  const [samlNameIDFormat, setSamlNameIDFormat] = useState(
    detail.saml?.name_id_format || SAML_DEFAULT_NAMEID_FORMAT,
  )
  const [samlNameIDSource, setSamlNameIDSource] = useState(
    detail.saml?.name_id_source || DEFAULT_NAMEID_SOURCE,
  )
  const [samlSignAssertion, setSamlSignAssertion] = useState(detail.saml?.sign_assertion ?? true)
  const [samlSignResponse, setSamlSignResponse] = useState(detail.saml?.sign_response ?? false)
  const [samlWantSignedRequests, setSamlWantSignedRequests] = useState(
    detail.saml?.want_authn_requests_signed ?? false,
  )
  const [samlSigningCert, setSamlSigningCert] = useState(
    detail.saml?.authn_request_signing_certificate_pem ?? '',
  )
  const [samlRulesJSON, setSamlRulesJSON] = useState(
    JSON.stringify(detail.saml?.rules ?? [], null, 2),
  )
  const signInView = detail.sign_in_policy
  const initialSignInRule = signInView?.policy?.rules?.[0]
  const [signInEnabled, setSignInEnabled] = useState(initialSignInRule?.enabled ?? false)
  const [signInStrength, setSignInStrength] = useState<RequiredAuthnStrength>(
    initialSignInRule?.required_authn.strength ?? 'Password',
  )
  const [signInReauthMaxAge, setSignInReauthMaxAge] = useState(
    initialSignInRule?.condition.reauth_max_age_seconds?.toString() ?? '',
  )
  const [signInNetworkCIDRs, setSignInNetworkCIDRs] = useState(
    (initialSignInRule?.condition.network_allow_cidrs ?? []).join('\n'),
  )
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminApplicationsDictionary)

  const nameInvalid = name.trim() === ''

  async function selectIconFile(file: File | null) {
    const token = ++iconSelectionToken.current
    if (!file) {
      setError('')
      setIconFile(null)
      setRemoveIcon(false)
      return
    }
    const validationError = await validateApplicationIconFile(file)
    if (token !== iconSelectionToken.current) return
    if (validationError) {
      setIconFile(null)
      setRemoveIcon(false)
      setError(validationError === 'too-large' ? t.iconTooLargeError : t.iconInvalidTypeError)
      return
    }
    setError('')
    setIconFile(file)
    setRemoveIcon(false)
  }

  useEffect(() => {
    if (!iconFile) {
      setIconPreviewURL('')
      return
    }
    const url = URL.createObjectURL(iconFile)
    setIconPreviewURL(url)
    return () => URL.revokeObjectURL(url)
  }, [iconFile])

  const iconPreview = iconPreviewURL || (removeIcon ? '' : safeApplicationIconURL(app.icon_url))

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (nameInvalid) return
    setSaving(true)
    setError('')
    try {
      const metaPatch: Record<string, unknown> = {}
      if (name.trim() !== app.name) metaPatch.name = name.trim()
      if (app.kind !== 'service' && launchURL.trim() !== (app.launch_url ?? '')) {
        metaPatch.launch_url = launchURL.trim()
      }
      if (status !== app.status) metaPatch.status = status
      if (Object.keys(metaPatch).length > 0) {
        await updateAdminApplication(csrfToken, app.application_id, metaPatch)
      }
      if (removeIcon && app.icon_object_key) {
        await deleteApplicationIcon(csrfToken, app.application_id)
      }
      if (iconFile) {
        if (iconFile.size > MAX_APPLICATION_ICON_BYTES) {
          setError(t.iconTooLargeError)
          setSaving(false)
          return
        }
        await uploadApplicationIcon(csrfToken, app.application_id, iconFile)
      }
      if (detail.oidc) {
        const nextRedirects = parseList(redirects)
        const nextGrants = parseList(grantTypes)
        const nextResponses = parseList(responseTypes)
        const redirectsChanged =
          app.kind !== 'service' && nextRedirects.join(',') !== detail.oidc.redirect_uris.join(',')
        const scopeChanged = scope.trim() !== detail.oidc.scope
        const grantsChanged = nextGrants.join(',') !== detail.oidc.grant_types.join(',')
        const responsesChanged = nextResponses.join(',') !== detail.oidc.response_types.join(',')
        const parChanged = requirePAR !== detail.oidc.require_pushed_authorization_requests
        const dpopChanged = dpopBound !== detail.oidc.dpop_bound_access_tokens
        if (
          redirectsChanged ||
          scopeChanged ||
          grantsChanged ||
          responsesChanged ||
          parChanged ||
          dpopChanged
        ) {
          await updateApplicationOidcConfig(csrfToken, app.application_id, {
            redirect_uris: redirectsChanged ? nextRedirects : undefined,
            scope: scopeChanged ? scope.trim() : undefined,
            grant_types: grantsChanged ? nextGrants : undefined,
            response_types: responsesChanged ? nextResponses : undefined,
            require_pushed_authorization_requests: parChanged ? requirePAR : undefined,
            dpop_bound_access_tokens: dpopChanged ? dpopBound : undefined,
          })
        }
      }
      if (detail.wsfed) {
        let nextRules: WsFedClaimMappingRule[]
        try {
          const parsed = JSON.parse(rulesJSON || '[]')
          if (!Array.isArray(parsed)) throw new Error('not an array')
          nextRules = parsed
        } catch {
          setError(t.invalidClaimRulesJsonError)
          setSaving(false)
          return
        }
        const nextReplies = parseList(replies)
        const changed =
          nextReplies.join(',') !== detail.wsfed.reply_urls.join(',') ||
          audience.trim() !== detail.wsfed.audience ||
          tokenType !== detail.wsfed.token_type ||
          nameIDFormat !== detail.wsfed.name_id_format ||
          nameIDSource.trim() !== detail.wsfed.name_id_source ||
          JSON.stringify(nextRules) !== JSON.stringify(detail.wsfed.rules ?? [])
        if (changed) {
          await updateApplicationWsFedConfig(csrfToken, app.application_id, {
            reply_urls: nextReplies,
            audience: audience.trim(),
            token_type: tokenType,
            name_id_format: nameIDFormat,
            name_id_source: nameIDSource.trim(),
            rules: nextRules,
          })
        }
      }
      if (detail.saml) {
        let nextRules: WsFedClaimMappingRule[]
        try {
          const parsed = JSON.parse(samlRulesJSON || '[]')
          if (!Array.isArray(parsed)) throw new Error('not an array')
          nextRules = parsed
        } catch {
          setError(t.invalidClaimRulesJsonError)
          setSaving(false)
          return
        }
        const nextACS = parseList(samlACS)
        if (samlWantSignedRequests && samlSigningCert.trim() === '') {
          setError(t.signingCertRequiredError)
          setSaving(false)
          return
        }
        const changed =
          nextACS.join(',') !== detail.saml.acs_urls.join(',') ||
          samlSLO.trim() !== detail.saml.slo_url ||
          samlAudience.trim() !== detail.saml.audience ||
          samlNameIDFormat !== detail.saml.name_id_format ||
          samlNameIDSource.trim() !== detail.saml.name_id_source ||
          samlSignAssertion !== detail.saml.sign_assertion ||
          samlSignResponse !== detail.saml.sign_response ||
          samlWantSignedRequests !== (detail.saml.want_authn_requests_signed ?? false) ||
          samlSigningCert.trim() !== (detail.saml.authn_request_signing_certificate_pem ?? '') ||
          JSON.stringify(nextRules) !== JSON.stringify(detail.saml.rules ?? [])
        if (changed) {
          if (nextACS.length === 0) {
            setError(t.acsUrlRequiredError)
            setSaving(false)
            return
          }
          await updateApplicationSamlConfig(csrfToken, app.application_id, {
            acs_urls: nextACS,
            slo_url: samlSLO.trim(),
            audience: samlAudience.trim(),
            name_id_format: samlNameIDFormat,
            name_id_source: samlNameIDSource.trim(),
            sign_assertion: samlSignAssertion,
            sign_response: samlSignResponse,
            want_authn_requests_signed: samlWantSignedRequests,
            authn_request_signing_certificate_pem: samlSigningCert.trim(),
            rules: nextRules,
          })
        }
      }
      const reauthText = signInReauthMaxAge.trim()
      const reauthMaxAge = reauthText === '' ? undefined : Number.parseInt(reauthText, 10)
      if (
        reauthText !== '' &&
        (reauthMaxAge === undefined || !Number.isFinite(reauthMaxAge) || reauthMaxAge <= 0)
      ) {
        setError(t.reauthPositiveIntegerError)
        setSaving(false)
        return
      }
      const networkCIDRs = signInNetworkCIDRs
        .split('\n')
        .map((entry) => entry.trim())
        .filter((entry) => entry !== '')
      const nextSignInRules: SignInRule[] = signInEnabled
        ? [
            {
              rule_id: initialSignInRule?.rule_id ?? '',
              name: 'app-override',
              enabled: true,
              required_authn: {
                strength: signInStrength,
              },
              condition: {
                reauth_max_age_seconds: reauthMaxAge,
                network_allow_cidrs: networkCIDRs.length > 0 ? networkCIDRs : undefined,
              },
            },
          ]
        : []
      const prevSignInRules = signInView?.policy?.rules ?? []
      if (JSON.stringify(nextSignInRules) !== JSON.stringify(prevSignInRules)) {
        await updateAppSignInPolicy(csrfToken, app.application_id, nextSignInRules)
      }
      window.location.assign(detailURL(app.application_id))
    } catch (cause) {
      setError(messageOf(cause, t.applicationUpdateFailedError))
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="applications"
      actorUsername={actorUsername}
      title={t.editTitle.replace('{name}', app.name)}
      description={kindLabel(app, t)}
      actions={
        <a
          href={detailURL(app.application_id)}
          className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToDetail}
        </a>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      <div className="grid max-w-3xl gap-6">
        <Card className="p-6">
          <form onSubmit={submit} className="grid gap-6">
            <section className="grid gap-4">
              <SectionTitle>{t.basicInfoHeading}</SectionTitle>
              <div className="grid gap-1.5">
                <Label htmlFor="edit-name">{t.nameFieldLabel}</Label>
                <Input
                  id="edit-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  aria-invalid={nameInvalid}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-icon-file">{t.iconImageFieldLabel}</Label>
                <fieldset
                  className="flex items-center gap-3 rounded-lg border border-dashed border-slate-300 p-3"
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={(event) => {
                    event.preventDefault()
                    void selectIconFile(event.dataTransfer.files?.[0] ?? null)
                  }}
                >
                  {iconPreview ? (
                    <img
                      src={iconPreview}
                      alt=""
                      className="size-14 rounded-lg border border-slate-200 object-cover"
                    />
                  ) : (
                    <span className="flex size-14 items-center justify-center rounded-lg border border-blue-100 bg-blue-50 text-sm font-bold text-blue-700">
                      {initials(name)}
                    </span>
                  )}
                  <div className="grid flex-1 gap-2 sm:flex sm:items-center">
                    <Input
                      id="edit-icon-file"
                      type="file"
                      accept="image/png,image/jpeg,image/webp,image/gif"
                      onChange={(e) => {
                        void selectIconFile(e.target.files?.[0] ?? null)
                      }}
                    />
                    {app.icon_object_key || iconFile ? (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => {
                          setIconFile(null)
                          setRemoveIcon(true)
                        }}
                      >
                        <IconTrash size={16} aria-hidden="true" />
                        {t.delete}
                      </Button>
                    ) : null}
                  </div>
                </fieldset>
                <p className="text-xs text-slate-500">{t.iconHelpText}</p>
              </div>
              {app.kind !== 'service' ? (
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-launch">{t.launchUrlFieldLabel}</Label>
                  <Input
                    id="edit-launch"
                    value={launchURL}
                    onChange={(e) => setLaunchURL(e.target.value)}
                    placeholder="https://app.example.com/launch"
                  />
                </div>
              ) : null}
              <div className="grid gap-1.5">
                <Label>{t.statusFieldLabel}</Label>
                <Select
                  value={status}
                  onValueChange={(v) => setStatus(v as ApplicationStatus)}
                  options={statusOptions(t)}
                  className="w-40"
                />
              </div>
            </section>

            {detail.oidc ? (
              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <div className="flex items-center gap-2">
                  <IconKey size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>
                    {app.kind === 'service' ? t.serviceKindSectionHeading : t.oidcSectionHeading}
                  </SectionTitle>
                </div>
                <CopyableField label={t.clientIdFieldLabel} value={detail.oidc.client_id} />
                {app.kind !== 'service' ? (
                  <div className="grid gap-1.5">
                    <Label htmlFor="edit-redirects">{t.redirectUriFieldLabel}</Label>
                    <textarea
                      id="edit-redirects"
                      value={redirects}
                      onChange={(e) => setRedirects(e.target.value)}
                      rows={3}
                      className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                      placeholder="https://app.example.com/callback"
                    />
                    <p className="text-xs text-slate-500">{t.redirectUriHelp}</p>
                  </div>
                ) : null}
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-scope">{t.scopeFieldLabel}</Label>
                  <Input
                    id="edit-scope"
                    value={scope}
                    onChange={(e) => setScope(e.target.value)}
                    className="font-mono text-xs"
                    placeholder="openid profile email"
                  />
                </div>
                {app.kind !== 'service' ? (
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="grid gap-1.5">
                      <Label htmlFor="edit-grant-types">{t.grantTypesFieldLabel}</Label>
                      <Input
                        id="edit-grant-types"
                        value={grantTypes}
                        onChange={(e) => setGrantTypes(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="authorization_code, refresh_token"
                      />
                      <p className="text-xs text-slate-500">{t.grantTypesHelp}</p>
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="edit-response-types">{t.responseTypesFieldLabel}</Label>
                      <Input
                        id="edit-response-types"
                        value={responseTypes}
                        onChange={(e) => setResponseTypes(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="code"
                      />
                      <p className="text-xs text-slate-500">{t.responseTypesHelp}</p>
                    </div>
                  </div>
                ) : null}
                <div className="grid gap-2.5">
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={requirePAR}
                      onChange={(e) => setRequirePAR(e.target.checked)}
                      className="size-4"
                    />
                    {t.requirePARLabel}
                  </label>
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={dpopBound}
                      onChange={(e) => setDpopBound(e.target.checked)}
                      className="size-4"
                    />
                    {t.requireDpopLabel}
                  </label>
                </div>
                <div className="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs sm:grid-cols-3">
                  <ReadonlyMeta label={t.clientTypeMetaLabel} value={detail.oidc.client_type} />
                  <ReadonlyMeta
                    label={t.authMethodMetaLabel}
                    value={detail.oidc.token_endpoint_auth_method}
                  />
                  <ReadonlyMeta label={t.fapiProfileMetaLabel} value={detail.oidc.fapi_profile} />
                </div>
              </section>
            ) : null}

            {detail.wsfed ? (
              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <div className="flex items-center gap-2">
                  <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.wsFedSectionHeading}</SectionTitle>
                </div>
                <CopyableField label={t.wtrealmFieldLabel} value={detail.wsfed.wtrealm} />
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-replies">{t.replyUrlFieldLabel}</Label>
                  <textarea
                    id="edit-replies"
                    value={replies}
                    onChange={(e) => setReplies(e.target.value)}
                    rows={2}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                    placeholder="https://app.example.com/wsfed"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label>{t.nameIdFormatFieldLabel}</Label>
                  <Select
                    value={nameIDFormat}
                    onValueChange={setNameIDFormat}
                    options={nameIdFormatOptions(t)}
                    className="w-full"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-nameid-source">{t.nameIdSourceFieldLabel}</Label>
                  <Input
                    id="edit-nameid-source"
                    value={nameIDSource}
                    onChange={(e) => setNameIDSource(e.target.value)}
                    placeholder="sub"
                  />
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="grid gap-1.5">
                    <Label htmlFor="edit-audience">{t.audienceOptionalFieldLabel}</Label>
                    <Input
                      id="edit-audience"
                      value={audience}
                      onChange={(e) => setAudience(e.target.value)}
                      className="font-mono text-xs"
                      placeholder={t.audiencePlaceholderDefault}
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label>{t.tokenTypeSamlVersionFieldLabel}</Label>
                    <Select
                      value={tokenType}
                      onValueChange={(v) => setTokenType(v as WsFedTokenType)}
                      options={wsfedTokenTypeOptions(t)}
                      className="w-full"
                    />
                  </div>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-wsfed-rules">{t.claimMappingRulesJsonFieldLabel}</Label>
                  <textarea
                    id="edit-wsfed-rules"
                    value={rulesJSON}
                    onChange={(e) => setRulesJSON(e.target.value)}
                    rows={8}
                    spellCheck={false}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                    placeholder='[{"claim_type":"http://schemas.xmlsoap.org/claims/UPN","source":"user_attribute","source_key":"preferred_username","required":true}]'
                  />
                  <p className="text-xs text-slate-500">{t.claimMappingRulesHelp}</p>
                </div>
              </section>
            ) : null}

            {detail.saml ? (
              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <div className="flex items-center gap-2">
                  <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.samlSectionHeading}</SectionTitle>
                </div>
                <CopyableField label={t.entityIdFieldLabel} value={detail.saml.entity_id} />
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-saml-acs">{t.acsUrlFieldLabel}</Label>
                  <textarea
                    id="edit-saml-acs"
                    value={samlACS}
                    onChange={(e) => setSamlACS(e.target.value)}
                    rows={2}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                    placeholder="https://app.example.com/saml/acs"
                  />
                  <p className="text-xs text-slate-500">{t.acsUrlHelp}</p>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-saml-slo">{t.sloUrlOptionalFieldLabel}</Label>
                  <Input
                    id="edit-saml-slo"
                    value={samlSLO}
                    onChange={(e) => setSamlSLO(e.target.value)}
                    className="font-mono text-xs"
                    placeholder="https://app.example.com/saml/slo"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label>{t.nameIdFormatFieldLabel}</Label>
                  <Select
                    value={samlNameIDFormat}
                    onValueChange={setSamlNameIDFormat}
                    options={nameIdFormatOptions(t)}
                    className="w-full"
                  />
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="grid gap-1.5">
                    <Label htmlFor="edit-saml-nameid-source">{t.nameIdSourceFieldLabel}</Label>
                    <Input
                      id="edit-saml-nameid-source"
                      value={samlNameIDSource}
                      onChange={(e) => setSamlNameIDSource(e.target.value)}
                      placeholder="sub"
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="edit-saml-audience">{t.audienceOptionalFieldLabel}</Label>
                    <Input
                      id="edit-saml-audience"
                      value={samlAudience}
                      onChange={(e) => setSamlAudience(e.target.value)}
                      className="font-mono text-xs"
                      placeholder={t.audienceEntityDefault}
                    />
                  </div>
                </div>
                <div className="grid gap-2.5">
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={samlSignAssertion}
                      onChange={(e) => setSamlSignAssertion(e.target.checked)}
                      className="size-4"
                    />
                    {t.signAssertionLabel}
                  </label>
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={samlSignResponse}
                      onChange={(e) => setSamlSignResponse(e.target.checked)}
                      className="size-4"
                    />
                    {t.signResponseLabel}
                  </label>
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={samlWantSignedRequests}
                      onChange={(e) => setSamlWantSignedRequests(e.target.checked)}
                      className="size-4"
                    />
                    {t.wantSignedRequestsLabel}
                  </label>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-saml-request-signing-cert">
                    {t.requestSigningCertFieldLabel}
                  </Label>
                  <textarea
                    id="edit-saml-request-signing-cert"
                    value={samlSigningCert}
                    onChange={(e) => setSamlSigningCert(e.target.value)}
                    rows={7}
                    spellCheck={false}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                    placeholder="-----BEGIN CERTIFICATE-----"
                  />
                  <p className="text-xs text-slate-500">{t.requestSigningCertHelp}</p>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-saml-rules">{t.claimMappingRulesJsonFieldLabel}</Label>
                  <textarea
                    id="edit-saml-rules"
                    value={samlRulesJSON}
                    onChange={(e) => setSamlRulesJSON(e.target.value)}
                    rows={8}
                    spellCheck={false}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                    placeholder='[{"claim_type":"email","source":"user_attribute","source_key":"email","required":true}]'
                  />
                  <p className="text-xs text-slate-500">{t.claimMappingRulesHelp}</p>
                </div>
              </section>
            ) : null}

            {app.kind !== 'service' ? (
              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <div className="flex items-center gap-2">
                  <IconKey size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.signInPolicySectionHeading}</SectionTitle>
                </div>
                <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                  <input
                    type="checkbox"
                    checked={signInEnabled}
                    onChange={(e) => setSignInEnabled(e.target.checked)}
                    className="size-4"
                  />
                  {t.overrideTenantDefaultLabel}
                </label>
                {signInEnabled ? (
                  <div className="grid gap-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
                    <div className="grid gap-1.5">
                      <Label>{t.requiredAuthnStrengthFieldLabel}</Label>
                      <Select
                        value={signInStrength}
                        onValueChange={(value) => setSignInStrength(value as RequiredAuthnStrength)}
                        options={signInStrengthOptions(t)}
                        className="w-full"
                      />
                      <p className="text-xs text-slate-500">{t.mfaStepUpHelp}</p>
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="edit-sign-in-reauth">{t.reauthSecondsFieldLabel}</Label>
                      <Input
                        id="edit-sign-in-reauth"
                        type="number"
                        min="1"
                        value={signInReauthMaxAge}
                        onChange={(e) => setSignInReauthMaxAge(e.target.value)}
                        placeholder={t.reauthSecondsPlaceholder}
                      />
                      <p className="text-xs text-slate-500">{t.reauthSecondsHelp}</p>
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="edit-sign-in-cidrs">{t.allowedNetworksFieldLabel}</Label>
                      <textarea
                        id="edit-sign-in-cidrs"
                        value={signInNetworkCIDRs}
                        onChange={(e) => setSignInNetworkCIDRs(e.target.value)}
                        rows={3}
                        spellCheck={false}
                        className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                        placeholder={'10.0.0.0/8\n192.168.1.0/24'}
                      />
                      <p className="text-xs text-slate-500">{t.allowedNetworksHelp}</p>
                    </div>
                  </div>
                ) : null}
                {(() => {
                  const defaultRules = signInView?.tenant_default?.rules ?? []
                  const appRule = appRuleFromInputs(
                    signInStrength,
                    signInReauthMaxAge,
                    signInNetworkCIDRs,
                  )
                  const effectiveSummary = signInEnabled
                    ? summarizeSignInRule(appRule, t)
                    : defaultRules
                        .filter((r) => r.enabled)
                        .map((rule) => summarizeSignInRule(rule, t))
                        .join('、') || t.noAdditionalRequirementsNotice
                  const weaker = signInEnabled && signInRuleWeakerThanDefault(appRule, defaultRules)
                  return (
                    <div className="grid gap-3 rounded-lg border border-slate-200 bg-white p-4">
                      <div className="grid gap-1">
                        <p className="text-xs font-semibold text-slate-500">
                          {t.tenantDefaultLabel}
                        </p>
                        {defaultRules.filter((r) => r.enabled).length > 0 ? (
                          <ul className="grid gap-1 text-xs text-slate-600">
                            {defaultRules
                              .filter((r) => r.enabled)
                              .map((rule) => (
                                <li key={rule.rule_id}>{summarizeSignInRule(rule, t)}</li>
                              ))}
                          </ul>
                        ) : (
                          <p className="text-xs text-slate-500">
                            {t.noAdditionalRequirementsNotice}
                          </p>
                        )}
                      </div>
                      <div className="grid gap-1">
                        <p className="text-xs font-semibold text-slate-500">
                          {t.effectivePolicyLabel}
                        </p>
                        <p className="text-xs text-slate-600">{effectiveSummary}</p>
                        <p className="text-xs text-slate-400">
                          {signInEnabled ? t.overrideAppliedNotice : t.tenantDefaultAppliedNotice}
                          {t.savedAfterConfirmSuffix}
                        </p>
                      </div>
                      {weaker ? (
                        <Alert variant="destructive">{t.weakerThanDefaultWarning}</Alert>
                      ) : null}
                    </div>
                  )
                })()}
              </section>
            ) : null}

            <div className="flex justify-end gap-2 border-t border-slate-200 pt-5">
              <Button asChild variant="outline">
                <a href={detailURL(app.application_id)}>{t.cancel}</a>
              </Button>
              <Button type="submit" disabled={saving || nameInvalid}>
                {saving ? t.saving : t.save}
              </Button>
            </div>
          </form>
        </Card>

        {app.kind !== 'service' ? (
          <Card className="p-6">
            <AssignmentManager
              appID={app.application_id}
              csrfToken={csrfToken}
              onError={setError}
            />
          </Card>
        ) : null}

        {app.kind !== 'service' ? (
          <Card className="p-6">
            <CategoryManager app={app} csrfToken={csrfToken} onError={setError} />
          </Card>
        ) : null}
      </div>
    </AdminShell>
  )
}

// ===========================================================================
// カテゴリ (定義の管理 + アプリへの付与) — wi-70 / ADR-069
// ===========================================================================

function CategoryManager({
  app,
  csrfToken,
  onError,
}: {
  app: AdminApplication
  csrfToken: string
  onError: (msg: string) => void
}) {
  const [categories, setCategories] = useState<ApplicationCategory[]>([])
  const [assigned, setAssigned] = useState<Set<string>>(new Set(app.category_ids))
  const [newName, setNewName] = useState('')
  const [busy, setBusy] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const t = useDictionary(adminApplicationsDictionary)

  useEffect(() => {
    let cancelled = false
    void listApplicationCategories()
      .then((list) => {
        if (cancelled) return
        setCategories(list)
        setLoaded(true)
      })
      .catch((cause) => onError(messageOf(cause, t.categoryFetchFailedError)))
    return () => {
      cancelled = true
    }
  }, [onError, t.categoryFetchFailedError])

  async function run(action: () => Promise<void>) {
    setBusy(true)
    try {
      await action()
    } catch (cause) {
      onError(messageOf(cause, t.categoryUpdateFailedError))
    } finally {
      setBusy(false)
    }
  }

  function toggle(categoryID: string) {
    const next = new Set(assigned)
    if (next.has(categoryID)) next.delete(categoryID)
    else next.add(categoryID)
    setAssigned(next)
    void run(async () => {
      const updated = await setApplicationCategories(csrfToken, app.application_id, [...next])
      setAssigned(new Set(updated.category_ids))
    })
  }

  function addCategory() {
    const name = newName.trim()
    if (name === '') return
    void run(async () => {
      const created = await createApplicationCategory(csrfToken, { name })
      setCategories((current) => [...current, created])
      setNewName('')
    })
  }

  function removeCategory(categoryID: string) {
    void run(async () => {
      await deleteApplicationCategory(csrfToken, categoryID)
      setCategories((current) => current.filter((c) => c.category_id !== categoryID))
      setAssigned((current) => {
        const next = new Set(current)
        next.delete(categoryID)
        return next
      })
    })
  }

  return (
    <div className="flex flex-col gap-4">
      <SectionTitle>{t.categoriesHeading}</SectionTitle>
      <p className="text-xs text-slate-500">{t.categoriesHelp}</p>
      {loaded && categories.length === 0 ? (
        <p className="text-sm text-slate-500">{t.noCategoriesNotice}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {categories.map((category) => (
            <li key={category.category_id} className="flex items-center gap-3">
              <label className="flex flex-1 items-center gap-2 text-sm text-slate-800">
                <input
                  type="checkbox"
                  checked={assigned.has(category.category_id)}
                  onChange={() => toggle(category.category_id)}
                  disabled={busy}
                  className="size-4 rounded border-slate-300"
                />
                {category.name}
              </label>
              <Button
                type="button"
                variant="ghost"
                size="default"
                className="size-9 px-0 text-slate-400 hover:text-red-600"
                disabled={busy}
                onClick={() => removeCategory(category.category_id)}
                aria-label={t.deleteCategoryAria.replace('{name}', category.name)}
              >
                <IconTrash size={16} aria-hidden="true" />
              </Button>
            </li>
          ))}
        </ul>
      )}
      <div className="flex items-center gap-2">
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder={t.newCategoryPlaceholder}
          disabled={busy}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              addCategory()
            }
          }}
        />
        <Button type="button" variant="secondary" onClick={addCategory} disabled={busy}>
          {t.add}
        </Button>
      </div>
    </div>
  )
}

// ===========================================================================
// 割り当て (read-only リスト / 管理)
// ===========================================================================

function useAssignmentData(
  appID: string,
  onError: (msg: string) => void,
  t: AdminApplicationsDictionary,
) {
  const [assignments, setAssignments] = useState<ApplicationAssignment[]>([])
  const [users, setUsers] = useState<AdminUser[]>([])
  const [groups, setGroups] = useState<AdminGroup[]>([])
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    let cancelled = false
    void Promise.all([listApplicationAssignments(appID), listAdminUsers(), listAdminGroups()])
      .then(([a, u, g]) => {
        if (cancelled) return
        setAssignments(a)
        setUsers(u)
        setGroups(g)
        setLoaded(true)
      })
      .catch((cause) => onError(messageOf(cause, t.assignmentFetchFailedError)))
    return () => {
      cancelled = true
    }
  }, [appID, onError, t.assignmentFetchFailedError])

  return { assignments, setAssignments, users, groups, loaded }
}

function useDisplayName(users: AdminUser[], groups: AdminGroup[]) {
  const userName = useMemo(() => new Map(users.map((u) => [u.id, u.preferred_username])), [users])
  const groupName = useMemo(() => new Map(groups.map((g) => [g.id, g.name])), [groups])
  return (a: ApplicationAssignment): string => {
    if (a.subject_type === 'user') return userName.get(a.subject_id) ?? a.subject_id
    return groupName.get(a.subject_id) ?? a.subject_id
  }
}

function AssignmentChip({ a, displayName }: { a: ApplicationAssignment; displayName: string }) {
  const t = useDictionary(adminApplicationsDictionary)
  return (
    <span className="flex items-center gap-2">
      <span
        className={`rounded px-1.5 py-0.5 text-xs ${
          a.subject_type === 'user' ? 'bg-blue-50 text-blue-700' : 'bg-violet-50 text-violet-700'
        }`}
      >
        {a.subject_type === 'user' ? t.userTypeLabel : t.groupTypeLabel}
      </span>
      <span className="font-medium text-slate-800">{displayName}</span>
    </span>
  )
}

function AssignmentList({ appID, onError }: { appID: string; onError: (msg: string) => void }) {
  const t = useDictionary(adminApplicationsDictionary)
  const { assignments, users, groups, loaded } = useAssignmentData(appID, onError, t)
  const displayName = useDisplayName(users, groups)

  if (!loaded) return <p className="text-xs text-slate-400">{t.loadingNotice}</p>
  if (assignments.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-slate-200 px-3 py-4 text-center text-xs text-slate-400">
        {t.noAssignmentsNotice}
      </p>
    )
  }
  return (
    <ul className="grid gap-2">
      {assignments.map((a) => (
        <li
          key={`${a.subject_type}:${a.subject_id}`}
          className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
        >
          <AssignmentChip a={a} displayName={displayName(a)} />
        </li>
      ))}
    </ul>
  )
}

function AssignmentManager({
  appID,
  csrfToken,
  onError,
}: {
  appID: string
  csrfToken: string
  onError: (msg: string) => void
}) {
  const t = useDictionary(adminApplicationsDictionary)
  const { assignments, setAssignments, users, groups, loaded } = useAssignmentData(
    appID,
    onError,
    t,
  )
  const displayName = useDisplayName(users, groups)
  const [subjectType, setSubjectType] = useState<'user' | 'group'>('user')
  const [subjectID, setSubjectID] = useState('')
  const [busy, setBusy] = useState(false)

  const assignedKeys = useMemo(
    () => new Set(assignments.map((a) => `${a.subject_type}:${a.subject_id}`)),
    [assignments],
  )

  const options: SelectOption[] = useMemo(() => {
    const source =
      subjectType === 'user'
        ? users.map((u) => ({ value: u.id, label: u.preferred_username }))
        : groups.map((g) => ({ value: g.id, label: g.name }))
    return source.filter((o) => !assignedKeys.has(`${subjectType}:${o.value}`))
  }, [subjectType, users, groups, assignedKeys])

  async function add(event: FormEvent) {
    event.preventDefault()
    if (!subjectID) return
    setBusy(true)
    try {
      const created = await assignApplication(csrfToken, appID, {
        subject_type: subjectType,
        subject_id: subjectID,
      })
      setAssignments((current) => [...current, created])
      setSubjectID('')
    } catch (cause) {
      onError(messageOf(cause, t.assignmentAddFailedError))
    } finally {
      setBusy(false)
    }
  }

  async function remove(a: ApplicationAssignment) {
    try {
      await unassignApplication(csrfToken, appID, a.subject_type, a.subject_id)
      setAssignments((current) =>
        current.filter(
          (x) => !(x.subject_type === a.subject_type && x.subject_id === a.subject_id),
        ),
      )
    } catch (cause) {
      onError(messageOf(cause, t.assignmentRemoveFailedError))
    }
  }

  return (
    <section className="grid gap-3">
      <SectionTitle>{t.assignmentsHeading}</SectionTitle>
      <p className="text-xs text-slate-500">{t.assignmentsManagerHelp}</p>
      {!loaded ? (
        <p className="text-xs text-slate-400">{t.loadingNotice}</p>
      ) : assignments.length === 0 ? (
        <p className="rounded-lg border border-dashed border-slate-200 px-3 py-4 text-center text-xs text-slate-400">
          {t.noAssignmentsShortNotice}
        </p>
      ) : (
        <ul className="grid gap-2">
          {assignments.map((a) => (
            <li
              key={`${a.subject_type}:${a.subject_id}`}
              className="flex items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
            >
              <AssignmentChip a={a} displayName={displayName(a)} />
              <Button
                variant="ghost"
                className="text-rose-700 hover:bg-rose-50"
                onClick={() => remove(a)}
              >
                <IconX size={14} aria-hidden="true" />
                {t.unassign}
              </Button>
            </li>
          ))}
        </ul>
      )}

      <form className="flex flex-wrap items-end gap-2" onSubmit={add}>
        <div className="grid gap-1.5">
          <Label>{t.targetFieldLabel}</Label>
          <Select
            value={subjectType}
            onValueChange={(v) => {
              setSubjectType(v as 'user' | 'group')
              setSubjectID('')
            }}
            options={[
              { value: 'user', label: t.userTypeLabel },
              { value: 'group', label: t.groupTypeLabel },
            ]}
            className="w-32"
          />
        </div>
        <div className="grid flex-1 gap-1.5">
          <Label>{subjectType === 'user' ? t.selectUserFieldLabel : t.selectGroupFieldLabel}</Label>
          <Select
            value={subjectID}
            onValueChange={setSubjectID}
            options={options}
            placeholder={options.length === 0 ? t.noTargetsNotice : t.selectPlaceholder}
            className="w-full"
            disabled={options.length === 0}
          />
        </div>
        <Button type="submit" disabled={busy || !subjectID}>
          <IconUserPlus size={16} aria-hidden="true" />
          {t.assign}
        </Button>
      </form>
    </section>
  )
}

// ===========================================================================
// 作成ダイアログ
// ===========================================================================

function CreateApplicationDialog({
  csrfToken,
  onClose,
  onCreated,
}: {
  csrfToken: string
  onClose: () => void
  onCreated: (id: string) => void
}) {
  const [type, setType] = useState<AppType>('oidc')
  const [name, setName] = useState('')
  const [launchURL, setLaunchURL] = useState('')
  const [redirectURIs, setRedirectURIs] = useState('')
  const [scope, setScope] = useState('')
  const [clientType, setClientType] = useState<'confidential' | 'public'>('confidential')
  const [authMethod, setAuthMethod] = useState('client_secret_basic')
  const [jwksURI, setJwksURI] = useState('')
  const [tlsSubjectDN, setTlsSubjectDN] = useState('')
  const [wtrealm, setWtrealm] = useState('')
  const [replyURLs, setReplyURLs] = useState('')
  const [nameIDFormat, setNameIDFormat] = useState(DEFAULT_NAMEID_FORMAT)
  const [nameIDSource, setNameIDSource] = useState(DEFAULT_NAMEID_SOURCE)
  const [samlEntityID, setSamlEntityID] = useState('')
  const [samlACSURLs, setSamlACSURLs] = useState('')
  const [samlSLOURL, setSamlSLOURL] = useState('')
  const [samlNameIDFormat, setSamlNameIDFormat] = useState(SAML_DEFAULT_NAMEID_FORMAT)
  const [samlNameIDSource, setSamlNameIDSource] = useState(DEFAULT_NAMEID_SOURCE)
  const [samlSignResponse, setSamlSignResponse] = useState(false)
  const [samlWantSignedRequests, setSamlWantSignedRequests] = useState(false)
  const [samlSigningCert, setSamlSigningCert] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [secret, setSecret] = useState<{ clientID: string; clientSecret: string } | null>(null)
  const [createdID, setCreatedID] = useState('')
  const t = useDictionary(adminApplicationsDictionary)

  const nameInvalid = name.trim() === ''

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (nameInvalid) return
    setSaving(true)
    setError('')
    if (type === 'saml' && samlWantSignedRequests && samlSigningCert.trim() === '') {
      setError(t.signingCertRequiredError)
      setSaving(false)
      return
    }
    try {
      const result = await createAdminApplication(csrfToken, {
        name: name.trim(),
        type,
        launch_url: launchURL.trim() || undefined,
        redirect_uris: type === 'oidc' ? parseList(redirectURIs) : undefined,
        scope: type === 'service' || type === 'oidc' ? scope.trim() || undefined : undefined,
        client_type: type === 'oidc' ? clientType : undefined,
        token_endpoint_auth_method: type === 'oidc' ? authMethod : undefined,
        jwks_uri: type === 'oidc' && authMethod === 'private_key_jwt' ? jwksURI.trim() : undefined,
        tls_client_auth_subject_dn:
          type === 'oidc' && authMethod === 'tls_client_auth' ? tlsSubjectDN.trim() : undefined,
        wtrealm: type === 'wsfed' ? wtrealm.trim() : undefined,
        reply_urls: type === 'wsfed' ? parseList(replyURLs) : undefined,
        name_id_format:
          type === 'wsfed' ? nameIDFormat : type === 'saml' ? samlNameIDFormat : undefined,
        name_id_source:
          type === 'wsfed'
            ? nameIDSource.trim()
            : type === 'saml'
              ? samlNameIDSource.trim()
              : undefined,
        entity_id: type === 'saml' ? samlEntityID.trim() : undefined,
        acs_urls: type === 'saml' ? parseList(samlACSURLs) : undefined,
        slo_url: type === 'saml' ? samlSLOURL.trim() || undefined : undefined,
        sign_response: type === 'saml' ? samlSignResponse : undefined,
        want_authn_requests_signed: type === 'saml' ? samlWantSignedRequests : undefined,
        authn_request_signing_certificate_pem:
          type === 'saml' ? samlSigningCert.trim() || undefined : undefined,
      })
      const id = result.application.application_id
      if (result.client_secret && result.client_id) {
        // OIDC / サービスは client_secret を一度だけ提示し、確認後に詳細へ遷移する。
        setCreatedID(id)
        setSecret({ clientID: result.client_id, clientSecret: result.client_secret })
        return
      }
      onCreated(id)
    } catch (cause) {
      setError(messageOf(cause, t.applicationCreateFailedError))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/30 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="app-create-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative flex max-h-[88vh] w-full max-w-xl flex-col overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <div>
            <p className="text-xs font-bold uppercase tracking-normal text-blue-700">
              {t.applicationEyebrow}
            </p>
            <h2 id="app-create-title" className="mt-1 text-xl font-semibold">
              {secret ? t.createdHeading : t.addApplicationHeading}
            </h2>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>

        {secret ? (
          <div className="grid gap-4 p-6">
            <Alert variant="success">
              {t.clientCreatedNotice}
              <strong>{t.clientSecretVisibleOnceStrong}</strong>
              {t.clientSecretVisibleOnceSuffix}
            </Alert>
            <CopyableField label={t.clientIdFieldLabel} value={secret.clientID} />
            <CopyableField label={t.clientSecretFieldLabel} value={secret.clientSecret} />
            <div className="flex justify-end">
              <Button type="button" onClick={() => onCreated(createdID)}>
                <IconCheck size={16} aria-hidden="true" />
                {t.storedConfirm}
              </Button>
            </div>
          </div>
        ) : (
          <form onSubmit={submit} className="flex min-h-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 overflow-y-auto">
              <div className="grid gap-6 p-6">
                <section className="grid gap-2">
                  <SectionTitle>{t.typeSectionHeading}</SectionTitle>
                  <div className="grid gap-2 sm:grid-cols-2">
                    {appTypeOptions(t).map((option) => {
                      const Icon = option.icon
                      const active = type === option.type
                      return (
                        <button
                          key={option.type}
                          type="button"
                          onClick={() => setType(option.type)}
                          className={`grid gap-1.5 rounded-xl border p-3 text-left transition ${
                            active
                              ? 'border-blue-500 bg-blue-50/60 ring-2 ring-blue-500/20'
                              : 'border-slate-200 hover:border-slate-300'
                          }`}
                        >
                          <Icon
                            size={20}
                            className={active ? 'text-blue-600' : 'text-slate-400'}
                            aria-hidden="true"
                          />
                          <span className="text-sm font-semibold text-slate-900">
                            {option.label}
                          </span>
                          <span className="text-xs leading-snug text-slate-500">
                            {option.description}
                          </span>
                        </button>
                      )
                    })}
                  </div>
                </section>

                <section className="grid gap-4 border-t border-slate-200 pt-5">
                  <SectionTitle>{t.basicInfoHeading}</SectionTitle>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-name">{t.nameFieldLabel}</Label>
                    <Input
                      id="app-name"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      required
                      placeholder="Payroll"
                    />
                  </div>
                  {type !== 'service' ? (
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-launch">
                        {type === 'weblink' ? t.linkUrlFieldLabel : t.launchUrlOptionalFieldLabel}
                      </Label>
                      <Input
                        id="app-launch"
                        value={launchURL}
                        onChange={(e) => setLaunchURL(e.target.value)}
                        placeholder="https://app.example.com"
                        required={type === 'weblink'}
                      />
                      {type !== 'weblink' ? (
                        <p className="text-xs text-slate-500">{t.launchUrlHelp}</p>
                      ) : null}
                    </div>
                  ) : null}
                </section>

                {type === 'service' ? (
                  <section className="grid gap-4 border-t border-slate-200 pt-5">
                    <SectionTitle>{t.serviceKindSectionHeading}</SectionTitle>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-scope">{t.scopeOptionalFieldLabel}</Label>
                      <Input
                        id="app-scope"
                        value={scope}
                        onChange={(e) => setScope(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="catalog:read invoice:read"
                      />
                      <p className="text-xs text-slate-500">{t.serviceScopeHelp}</p>
                    </div>
                  </section>
                ) : null}

                {type === 'oidc' ? (
                  <section className="grid gap-4 border-t border-slate-200 pt-5">
                    <SectionTitle>{t.oidcSectionHeading}</SectionTitle>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-redirects">{t.redirectUriFieldLabel}</Label>
                      <textarea
                        id="app-redirects"
                        value={redirectURIs}
                        onChange={(e) => setRedirectURIs(e.target.value)}
                        rows={3}
                        required
                        className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                        placeholder="https://app.example.com/callback"
                      />
                      <p className="text-xs text-slate-500">{t.oidcRedirectHelp}</p>
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-oidc-scope">{t.scopeOptionalFieldLabel}</Label>
                      <Input
                        id="app-oidc-scope"
                        value={scope}
                        onChange={(e) => setScope(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="openid profile email"
                      />
                    </div>
                    <div className="grid gap-4 sm:grid-cols-2">
                      <div className="grid gap-1.5">
                        <Label>{t.clientTypeFieldLabel}</Label>
                        <Select
                          value={clientType}
                          onValueChange={(v) => {
                            const next = v as 'confidential' | 'public'
                            setClientType(next)
                            setAuthMethod(next === 'public' ? 'none' : 'client_secret_basic')
                          }}
                          options={[
                            { value: 'confidential', label: 'confidential' },
                            { value: 'public', label: 'public' },
                          ]}
                          className="w-full"
                        />
                      </div>
                      <div className="grid gap-1.5">
                        <Label>{t.authMethodFieldLabel}</Label>
                        <Select
                          value={authMethod}
                          onValueChange={setAuthMethod}
                          options={AUTH_METHODS}
                          className="w-full"
                        />
                      </div>
                    </div>
                    {authMethod === 'private_key_jwt' ? (
                      <div className="grid gap-1.5">
                        <Label htmlFor="app-jwks-uri">{t.jwksUriFieldLabel}</Label>
                        <Input
                          id="app-jwks-uri"
                          type="url"
                          value={jwksURI}
                          onChange={(e) => setJwksURI(e.target.value)}
                          className="font-mono text-xs"
                          placeholder="https://app.example.com/jwks.json"
                          required
                        />
                      </div>
                    ) : null}
                    {authMethod === 'tls_client_auth' ? (
                      <div className="grid gap-1.5">
                        <Label htmlFor="app-tls-dn">{t.tlsClientCertSubjectDnFieldLabel}</Label>
                        <Input
                          id="app-tls-dn"
                          value={tlsSubjectDN}
                          onChange={(e) => setTlsSubjectDN(e.target.value)}
                          className="font-mono text-xs"
                          placeholder="CN=app,OU=clients,O=example"
                          required
                        />
                      </div>
                    ) : null}
                    <p className="text-xs text-slate-500">{t.authMethodFixedNotice}</p>
                  </section>
                ) : null}

                {type === 'wsfed' ? (
                  <section className="grid gap-4 border-t border-slate-200 pt-5">
                    <SectionTitle>{t.wsFedSectionHeading}</SectionTitle>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-wtrealm">{t.wtrealmFieldLabel}</Label>
                      <Input
                        id="app-wtrealm"
                        value={wtrealm}
                        onChange={(e) => setWtrealm(e.target.value)}
                        required
                        className="font-mono text-xs"
                        placeholder="urn:app:example"
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-replies">{t.replyUrlFieldLabel}</Label>
                      <textarea
                        id="app-replies"
                        value={replyURLs}
                        onChange={(e) => setReplyURLs(e.target.value)}
                        rows={2}
                        required
                        className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                        placeholder="https://app.example.com/wsfed"
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label>{t.nameIdFormatFieldLabel}</Label>
                      <Select
                        value={nameIDFormat}
                        onValueChange={setNameIDFormat}
                        options={nameIdFormatOptions(t)}
                        className="w-full"
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-nameid-source">{t.nameIdSourceFieldLabel}</Label>
                      <Input
                        id="app-nameid-source"
                        value={nameIDSource}
                        onChange={(e) => setNameIDSource(e.target.value)}
                        placeholder="sub"
                      />
                    </div>
                  </section>
                ) : null}

                {type === 'saml' ? (
                  <section className="grid gap-4 border-t border-slate-200 pt-5">
                    <SectionTitle>{t.samlSectionHeading}</SectionTitle>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-saml-entity">{t.entityIdFieldLabel}</Label>
                      <Input
                        id="app-saml-entity"
                        value={samlEntityID}
                        onChange={(e) => setSamlEntityID(e.target.value)}
                        required
                        className="font-mono text-xs"
                        placeholder="https://app.example.com/saml/metadata"
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-saml-acs">{t.acsUrlFieldLabel}</Label>
                      <textarea
                        id="app-saml-acs"
                        value={samlACSURLs}
                        onChange={(e) => setSamlACSURLs(e.target.value)}
                        rows={2}
                        required
                        className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                        placeholder="https://app.example.com/saml/acs"
                      />
                      <p className="text-xs text-slate-500">{t.redirectUriHelp}</p>
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-saml-slo">{t.sloUrlOptionalFieldLabel}</Label>
                      <Input
                        id="app-saml-slo"
                        value={samlSLOURL}
                        onChange={(e) => setSamlSLOURL(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="https://app.example.com/saml/slo"
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label>{t.nameIdFormatFieldLabel}</Label>
                      <Select
                        value={samlNameIDFormat}
                        onValueChange={setSamlNameIDFormat}
                        options={nameIdFormatOptions(t)}
                        className="w-full"
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-saml-nameid-source">{t.nameIdSourceFieldLabel}</Label>
                      <Input
                        id="app-saml-nameid-source"
                        value={samlNameIDSource}
                        onChange={(e) => setSamlNameIDSource(e.target.value)}
                        placeholder="sub"
                      />
                    </div>
                    <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                      <input
                        type="checkbox"
                        checked={samlSignResponse}
                        onChange={(e) => setSamlSignResponse(e.target.checked)}
                        className="size-4"
                      />
                      {t.signResponseDefaultLabel}
                    </label>
                    <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                      <input
                        type="checkbox"
                        checked={samlWantSignedRequests}
                        onChange={(e) => setSamlWantSignedRequests(e.target.checked)}
                        className="size-4"
                      />
                      {t.wantSignedRequestsLabel}
                    </label>
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-saml-request-signing-cert">
                        {t.requestSigningCertFieldLabel}
                      </Label>
                      <textarea
                        id="app-saml-request-signing-cert"
                        value={samlSigningCert}
                        onChange={(e) => setSamlSigningCert(e.target.value)}
                        rows={6}
                        spellCheck={false}
                        className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                        placeholder="-----BEGIN CERTIFICATE-----"
                      />
                    </div>
                  </section>
                ) : null}

                {error ? <Alert variant="destructive">{error}</Alert> : null}
              </div>
            </div>
            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <Button type="button" variant="outline" onClick={onClose} disabled={saving}>
                {t.cancel}
              </Button>
              <Button type="submit" disabled={saving || nameInvalid}>
                {saving ? t.creating : t.create}
              </Button>
            </div>
          </form>
        )}
      </Card>
    </div>
  )
}

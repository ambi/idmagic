import { tenantURL } from '../../api'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import type { WorkloadTrustBundle } from '../../types'
import type { Dictionary } from './presentation'

export function listURL(): string {
  return tenantURL('/admin/workload-identity')
}

export function newURL(): string {
  return tenantURL('/admin/workload-identity/new')
}

export function detailURL(trustBundleID: string): string {
  return tenantURL(`/admin/workload-identity/${encodeURIComponent(trustBundleID)}`)
}

export function editURL(trustBundleID: string): string {
  return tenantURL(`/admin/workload-identity/${encodeURIComponent(trustBundleID)}/edit`)
}

export type BundleFormState = {
  name: string
  trustDomain: string
  issuer: string
  jwksUri: string
  jwks: string
  acceptedAudiences: string
  maxTtlSeconds: string
}

export const emptyBundleForm: BundleFormState = {
  name: '',
  trustDomain: '',
  issuer: '',
  jwksUri: '',
  jwks: '',
  acceptedAudiences: '',
  maxTtlSeconds: '3600',
}

// toBundleForm は既存の信頼バンドルを編集フォームへ写す。inline JWKS の本体は API が
// 返さないため、jwks は常に空から始まる (空のまま送れば現状維持になる)。
export function toBundleForm(bundle: WorkloadTrustBundle): BundleFormState {
  return {
    name: bundle.name,
    trustDomain: bundle.trust_domain,
    issuer: bundle.issuer,
    jwksUri: bundle.jwks_uri ?? '',
    jwks: '',
    acceptedAudiences: bundle.accepted_audiences.join(' '),
    maxTtlSeconds: String(bundle.max_subject_token_ttl_seconds),
  }
}

// EnabledStatusBadge は enabled/disabled の二状態を出す。信頼バンドルとバインディングは
// 同じライフサイクルの形をしているので、片方に名前を寄せず状態そのものを名前にする。
export function EnabledStatusBadge({
  status,
  t,
}: {
  status: 'enabled' | 'disabled'
  t: Dictionary
}) {
  return (
    <span
      className={
        status === 'enabled'
          ? 'rounded-full bg-emerald-50 px-2 py-0.5 text-[0.68rem] font-bold text-emerald-700'
          : 'rounded-full bg-slate-100 px-2 py-0.5 text-[0.68rem] font-bold text-slate-500'
      }
    >
      {status === 'enabled' ? t.statusEnabled : t.statusDisabled}
    </span>
  )
}

// TrustBundleFormFields は登録/編集で共有するフォーム本体。issuer と trust_domain は
// 更新リクエストが受け付けないため、編集時は locked にして入力できないことを示す。
export function TrustBundleFormFields({
  form,
  onChange,
  locked,
  t,
}: {
  form: BundleFormState
  onChange: (next: BundleFormState) => void
  locked: boolean
  t: Dictionary
}) {
  return (
    <>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="name">{t.nameLabel}</Label>
        <Input
          id="name"
          value={form.name}
          onChange={(event) => onChange({ ...form, name: event.target.value })}
          required
        />
        <p className="text-xs leading-5 text-slate-500">{t.nameHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="trust_domain">{t.trustDomainLabel}</Label>
        <Input
          id="trust_domain"
          value={form.trustDomain}
          disabled={locked}
          placeholder={t.trustDomainPlaceholder}
          onChange={(event) => onChange({ ...form, trustDomain: event.target.value })}
          required
        />
        <p className="text-xs leading-5 text-slate-500">{t.trustDomainHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="issuer">{t.issuerLabel}</Label>
        <Input
          id="issuer"
          type="url"
          value={form.issuer}
          disabled={locked}
          placeholder={t.issuerPlaceholder}
          onChange={(event) => onChange({ ...form, issuer: event.target.value })}
          required
        />
        <p className="text-xs leading-5 text-slate-500">{t.issuerHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="jwks_uri">{t.jwksUriLabel}</Label>
        <Input
          id="jwks_uri"
          type="url"
          value={form.jwksUri}
          placeholder={t.jwksUriPlaceholder}
          onChange={(event) => onChange({ ...form, jwksUri: event.target.value })}
        />
        <p className="text-xs leading-5 text-slate-500">{t.jwksUriHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="jwks">{t.jwksLabel}</Label>
        <textarea
          id="jwks"
          value={form.jwks}
          rows={4}
          placeholder={t.jwksPlaceholder}
          onChange={(event) => onChange({ ...form, jwks: event.target.value })}
          className="rounded-md border border-slate-300 bg-white p-2 font-mono text-xs text-slate-900 focus:border-blue-500 focus:outline-none"
        />
        <p className="text-xs leading-5 text-slate-500">{t.jwksHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="accepted_audiences">{t.acceptedAudiencesLabel}</Label>
        <Input
          id="accepted_audiences"
          value={form.acceptedAudiences}
          placeholder={t.acceptedAudiencesPlaceholder}
          onChange={(event) => onChange({ ...form, acceptedAudiences: event.target.value })}
          required
        />
        <p className="text-xs leading-5 text-slate-500">{t.acceptedAudiencesHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="max_ttl">{t.maxTtlLabel}</Label>
        <Input
          id="max_ttl"
          type="number"
          min={1}
          value={form.maxTtlSeconds}
          onChange={(event) => onChange({ ...form, maxTtlSeconds: event.target.value })}
          required
        />
        <p className="text-xs leading-5 text-slate-500">{t.maxTtlHelp}</p>
      </div>
    </>
  )
}

// ConfirmDialog は破壊的操作の前に影響範囲を読ませる。信頼バンドルの削除はバインディングを
// カスケードで消すため、message には件数を埋めた文を渡すこと (presentation.bundleConfirmation)。
export function ConfirmDialog({
  title,
  message,
  confirmLabel,
  cancelLabel,
  onCancel,
  onConfirm,
  busy,
}: {
  title: string
  message: string
  confirmLabel: string
  cancelLabel: string
  onCancel: () => void
  onConfirm: () => void
  busy: boolean
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={title}
      className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 px-4"
    >
      <Card className="w-full max-w-md p-6">
        <h2 className="text-base font-semibold text-slate-900">{title}</h2>
        <p className="mt-3 text-sm leading-6 text-slate-600">{message}</p>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" onClick={onCancel} disabled={busy}>
            {cancelLabel}
          </Button>
          <Button onClick={onConfirm} disabled={busy} className="bg-rose-600 hover:bg-rose-700">
            {confirmLabel}
          </Button>
        </div>
      </Card>
    </div>
  )
}

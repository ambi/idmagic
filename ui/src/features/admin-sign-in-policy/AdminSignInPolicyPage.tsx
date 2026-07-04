import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, tenantURL, updateTenantDefaultSignInPolicy } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select, type SelectOption } from '../../components/ui/select'
import type {
  AdminApplication,
  AppSignInPolicyView,
  RequiredAuthnStrength,
  SignInRule,
  TenantDefaultSignInPolicy,
} from '../../types'

const STRENGTH_OPTIONS: SelectOption[] = [
  { value: 'Password', label: 'パスワードのみ' },
  { value: 'Mfa', label: 'MFA 必須' },
]

// アプリ単位のサインインポリシー一覧行。上書きの有無・警告・実効ポリシーを総合表示する。
export type SignInPolicyAppRow = {
  application: AdminApplication
  view: AppSignInPolicyView
}

// 内部ルール名。UI には表示しないが、保存契約上ルールには名前が要る (ADR-081)。
const DEFAULT_RULE_NAME = 'tenant-default'

function strengthLabel(strength: RequiredAuthnStrength): string {
  return strength === 'Mfa' ? 'MFA 必須' : 'パスワードのみ'
}

// summarizeRules は実効ルール列を利用者向けの短い日本語へ要約する。内部ルール名は見せない (ADR-081)。
function summarizeRules(rules: SignInRule[]): string {
  const enabled = rules.filter((rule) => rule.enabled)
  if (enabled.length === 0) {
    return '追加要件なし'
  }
  return enabled
    .map((rule) => {
      const parts: string[] = [strengthLabel(rule.required_authn.strength)]
      if (rule.condition.reauth_max_age_seconds) {
        parts.push(`再認証まで ${rule.condition.reauth_max_age_seconds} 秒`)
      }
      const cidrs = rule.condition.network_allow_cidrs ?? []
      if (cidrs.length > 0) {
        parts.push(`許可ネットワーク ${cidrs.join(', ')}`)
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
}: {
  csrfToken: string
  actorUsername?: string
  policy: TenantDefaultSignInPolicy
  apps: SignInPolicyAppRow[]
}) {
  return (
    <AdminShell
      active="sign-in-policy"
      actorUsername={actorUsername}
      title="サインインポリシー"
      description="全アプリ共通のデフォルトポリシーと、アプリごとの実効ポリシーを一元的に管理します。"
    >
      <div className="grid gap-6">
        <DefaultPolicyCard csrfToken={csrfToken} policy={policy} />
        <ApplicationPolicyList apps={apps} />
      </div>
    </AdminShell>
  )
}

function DefaultPolicyCard({
  csrfToken,
  policy: initial,
}: {
  csrfToken: string
  policy: TenantDefaultSignInPolicy
}) {
  const [rule, setRule] = useState<SignInRule | undefined>(initial.rules?.[0])
  const [editing, setEditing] = useState(false)

  const strength = rule?.required_authn.strength ?? 'Password'
  const reauth = rule?.condition.reauth_max_age_seconds
  const cidrs = rule?.condition.network_allow_cidrs ?? []

  return (
    <Card className="p-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-slate-900">デフォルトサインインポリシー</h2>
          <p className="mt-1 text-sm text-slate-600">
            独自ポリシーを設定していない全アプリケーションに適用される baseline
            です。各アプリはこれを上書きできます。
          </p>
        </div>
        {!editing ? (
          <Button type="button" variant="outline" onClick={() => setEditing(true)}>
            編集
          </Button>
        ) : null}
      </header>
      <div className="mt-5">
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-3">
            <ReadField label="要求する認証強度" value={strengthLabel(strength)} />
            <ReadField
              label="再認証を求めるまでの時間"
              value={reauth ? `${reauth} 秒` : '無期限（再認証を求めない）'}
            />
            <ReadField
              label="許可するネットワーク"
              value={cidrs.length > 0 ? cidrs.join(', ') : '制限なし'}
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
  const [strength, setStrength] = useState<RequiredAuthnStrength>(
    rule?.required_authn.strength ?? 'Password',
  )
  const [reauthMaxAge, setReauthMaxAge] = useState(
    rule?.condition.reauth_max_age_seconds?.toString() ?? '',
  )
  const [networkCIDRs, setNetworkCIDRs] = useState(
    (rule?.condition.network_allow_cidrs ?? []).join('\n'),
  )
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const reauthText = reauthMaxAge.trim()
      const reauth = reauthText === '' ? undefined : Number.parseInt(reauthText, 10)
      if (reauth !== undefined && (Number.isNaN(reauth) || reauth < 1)) {
        setError('再認証を求めるまでの時間には 1 以上の秒数を入力してください。')
        setSaving(false)
        return
      }
      const cidrs = networkCIDRs
        .split('\n')
        .map((entry) => entry.trim())
        .filter((entry) => entry !== '')
      // デフォルトは常に適用される baseline なので、認証強度が最低でも 1 ルールとして保存する。
      const rules: SignInRule[] = [
        {
          rule_id: rule?.rule_id ?? '',
          name: DEFAULT_RULE_NAME,
          enabled: true,
          required_authn: { strength },
          condition: {
            reauth_max_age_seconds: reauth,
            network_allow_cidrs: cidrs.length > 0 ? cidrs : undefined,
          },
        },
      ]
      const next = await updateTenantDefaultSignInPolicy(csrfToken, rules)
      onSaved(next)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : 'デフォルトサインインポリシーを更新できませんでした。',
      )
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSave} className="grid gap-4">
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="grid gap-1.5">
        <Label>要求する認証強度</Label>
        <Select
          value={strength}
          onValueChange={(value) => setStrength(value as RequiredAuthnStrength)}
          options={STRENGTH_OPTIONS}
          className="w-full sm:w-72"
        />
        <p className="text-xs text-slate-500">
          「MFA 必須」の場合、単要素セッションはサインイン時に再認証 (step-up) へ誘導されます。
        </p>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="default-reauth">再認証を求めるまでの時間（秒）</Label>
        <Input
          id="default-reauth"
          type="number"
          min="1"
          value={reauthMaxAge}
          onChange={(e) => setReauthMaxAge(e.target.value)}
          placeholder="例: 3600"
          className="sm:w-72"
        />
        <p className="text-xs text-slate-500">
          この秒数を超えた認証は再認証（再ログイン）を求めます。空欄なら無期限です。
        </p>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="default-cidrs">許可するネットワーク (CIDR)</Label>
        <textarea
          id="default-cidrs"
          value={networkCIDRs}
          onChange={(e) => setNetworkCIDRs(e.target.value)}
          rows={3}
          spellCheck={false}
          className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
          placeholder={'10.0.0.0/8\n192.168.1.0/24'}
        />
        <p className="text-xs text-slate-500">
          1 行に 1 つの CIDR を入力します。指定するとリスト外の IP
          からのサインインは拒否されます。空欄なら制限しません。
        </p>
      </div>
      <div className="flex items-center gap-2 border-t border-slate-200 pt-5">
        <Button type="submit" disabled={saving}>
          {saving ? '保存中…' : '保存'}
        </Button>
        <Button type="button" variant="ghost" disabled={saving} onClick={onCancel}>
          キャンセル
        </Button>
      </div>
    </form>
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
  return (
    <Card className="p-6">
      <header>
        <h2 className="text-base font-semibold text-slate-900">アプリケーション別ポリシー</h2>
        <p className="mt-1 text-sm text-slate-600">
          アプリ独自のポリシーはデフォルトを上書きします。「最終的に適用されるポリシー」で実効値を確認でき、
          個別設定はアプリ編集画面から変更できます。
        </p>
      </header>
      <div className="mt-5">
        {apps.length === 0 ? (
          <p className="text-sm text-slate-500">
            サインインポリシー対象のアプリケーションがありません。
          </p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-slate-200">
            <table className="min-w-full divide-y divide-slate-200 text-left text-sm text-slate-700">
              <thead className="bg-slate-50 font-semibold text-slate-900">
                <tr>
                  <th className="px-4 py-2">アプリケーション</th>
                  <th className="px-4 py-2">個別設定</th>
                  <th className="px-4 py-2">最終的に適用されるポリシー</th>
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
                              上書き
                            </span>
                            {view.weaker_than_default ? (
                              <span className="rounded-md bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700">
                                デフォルトより弱い
                              </span>
                            ) : null}
                          </span>
                        ) : (
                          <span className="text-xs text-slate-400">デフォルトを適用</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-xs text-slate-600">
                        {summarizeRules(view.effective_rules ?? [])}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <a
                          href={tenantURL(
                            `/admin/applications/${encodeURIComponent(application.application_id)}/edit`,
                          )}
                          className="text-xs font-medium text-blue-700 hover:underline"
                        >
                          編集
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

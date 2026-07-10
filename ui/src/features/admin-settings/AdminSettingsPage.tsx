import { IconMail, IconShieldLock, IconTag, IconUsers } from '@tabler/icons-react'
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
import { cn } from '../../lib/utils'
import type { AdminSettings, ScimToken } from '../../types'

const DEFAULT_REALM = 'default'

export function displayNameError(value: string): string | null {
  return value.trim() ? null : '表示名を入力してください。'
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

type TabKey = 'general' | 'password-policy' | 'email' | 'scim'

type Tab = {
  key: TabKey
  label: string
  description: string
  icon: typeof IconTag
  disabled?: boolean
}

const tabs: Tab[] = [
  {
    key: 'general',
    label: '一般',
    description: 'テナント表示名などの基本情報を管理します。',
    icon: IconTag,
  },
  {
    key: 'password-policy',
    label: 'パスワードポリシー',
    description: 'テナント単位の上書き値。空欄のフィールドは IdMagic の標準値が適用されます。',
    icon: IconShieldLock,
  },
  {
    key: 'scim',
    label: 'SCIM 同期',
    description: '外部 IDP からのユーザー・グループ同期 (SCIM 2.0) を設定します。',
    icon: IconUsers,
  },
  {
    key: 'email',
    label: 'メール送信',
    description: '別 WI で扱う予定です。現状は環境変数経由で設定します。',
    icon: IconMail,
    disabled: true,
  },
]

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

  return (
    <AdminShell
      active="settings"
      actorUsername={actorUsername}
      title="設定"
      description="このテナントの管理者が触れる設定を集約します。"
    >
      {isSystemAdminOnDefault ? (
        <Alert>
          <p className="text-sm text-slate-700">
            他テナントの設定を編集するには
            <a href="/system/tenants" className="ml-1 font-medium text-blue-700 hover:underline">
              システムコンソールのテナント
            </a>
            ページを利用してください。
          </p>
        </Alert>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[220px_minmax(0,1fr)]">
        <nav className="flex flex-col gap-1" aria-label="設定タブ">
          {tabs.map((tab) => (
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
                  予定
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
          {active === 'scim' ? <ScimTab csrfToken={csrfToken} tenantID={settings.realm} /> : null}
          {active === 'email' ? (
            <Card className="p-6">
              <h2 className="text-base font-semibold text-slate-900">メール送信</h2>
              <p className="mt-2 text-sm text-slate-600">
                送信先 SMTP の設定は ADR-035 に従い環境変数で管理しています。UI 経由の編集は 別 WI
                で扱います。
              </p>
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

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const trimmed = displayName.trim()
      const validationError = displayNameError(displayName)
      if (validationError) {
        setError(validationError)
        return
      }
      if (trimmed === settings.display_name) {
        setNotice('変更はありません。')
        return
      }
      const next = await updateAdminSettings(csrfToken, { display_name: trimmed })
      onSaved(next)
      setDisplayName(next.display_name)
      setEditing(false)
      setNotice('表示名を更新しました。')
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : '設定を更新できませんでした。',
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
            <h2 className="text-base font-semibold text-slate-900">一般</h2>
            <p className="mt-1 text-sm text-slate-600">テナントの基本情報を確認できます。</p>
          </div>
          {!editing ? (
            <Button type="button" variant="outline" onClick={() => setEditing(true)}>
              編集
            </Button>
          ) : null}
        </div>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-2">
            <ReadSetting label="テナント ID" value={settings.tenant_id} mono />
            <ReadSetting label="表示名" value={settings.display_name} />
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="tenant-id">テナント ID</Label>
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
              <Label htmlFor="display-name">表示名</Label>
              <Input
                id="display-name"
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                maxLength={200}
              />
              <p className="text-xs text-slate-500">管理画面と承諾画面に表示される名前です。</p>
            </div>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving}>
                {saving ? '保存中…' : '保存'}
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
                キャンセル
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
      setNotice('パスワードポリシーを更新しました。')
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : 'パスワードポリシーを更新できませんでした。',
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
            <h2 className="text-base font-semibold text-slate-900">パスワードポリシー</h2>
            <p className="mt-1 text-sm text-slate-600">
              テナントに適用されるパスワード要件を確認できます。
            </p>
          </div>
          {!editing ? (
            <Button type="button" variant="outline" onClick={() => setEditing(true)}>
              編集
            </Button>
          ) : null}
        </div>
        <dl className="mt-3 grid grid-cols-3 gap-3 rounded-md border border-slate-200 bg-slate-50 px-4 py-3 text-xs">
          <div>
            <dt className="text-slate-500">標準 最小長</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {defaults.min_length} 文字
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">標準 最大長</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {defaults.max_length} 文字
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">標準 履歴件数</dt>
            <dd className="mt-0.5 text-sm font-semibold text-slate-900">
              {defaults.history_depth} 件
            </dd>
          </div>
        </dl>
        <p className="mt-2 text-xs text-slate-500">
          標準値より弱い設定 (最小長を下げる / 最大長を上げる / 履歴件数を減らす) は
          サーバ側で拒否されます。
        </p>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-3">
            <ReadSetting
              label="最小長"
              value={`${override?.min_length ?? defaults.min_length} 文字`}
            />
            <ReadSetting
              label="最大長"
              value={`${override?.max_length ?? defaults.max_length} 文字`}
            />
            <ReadSetting
              label="履歴件数"
              value={`${override?.history_depth ?? defaults.history_depth} 件`}
            />
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <PolicyField
                id="min-length"
                label="最小長 (min_length)"
                value={minLength}
                onChange={setMinLength}
                min={defaults.min_length}
                max={defaults.max_length}
                placeholder={defaults.min_length.toString()}
                hint={`${defaults.min_length} 以上`}
              />
              <PolicyField
                id="max-length"
                label="最大長 (max_length)"
                value={maxLength}
                onChange={setMaxLength}
                min={defaults.min_length}
                max={defaults.max_length}
                placeholder={defaults.max_length.toString()}
                hint={`${defaults.max_length} 以下`}
              />
              <PolicyField
                id="history-depth"
                label="履歴件数 (history_depth)"
                value={historyDepth}
                onChange={setHistoryDepth}
                min={defaults.history_depth}
                max={50}
                placeholder={defaults.history_depth.toString()}
                hint={`${defaults.history_depth} 以上`}
              />
            </div>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving}>
                {saving ? '保存中…' : '保存'}
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
                キャンセル
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

  useEffect(() => {
    async function loadData() {
      try {
        const tList = await listScimTokens()
        setTokens(tList)
      } catch {
        setError('SCIM アクセストークンを取得できませんでした。')
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
      setError('トークンの説明を入力してください。')
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
      setNotice('SCIM アクセストークンを発行しました。')
    } catch {
      setError('トークンを発行できませんでした。')
    }
  }

  async function handleRevokeToken(id: string) {
    setError('')
    setNotice('')
    try {
      await revokeScimToken(csrfToken, id)
      setTokens(tokens.filter((t) => t.id !== id))
      setNotice('トークンを失効させました。')
    } catch {
      setError('トークンを失効できませんでした。')
    }
  }

  if (loading) {
    return <div className="text-sm text-slate-500">読み込み中…</div>
  }

  const endpointUrl = `${window.location.origin}/realms/${tenantID}/scim/v2`

  return (
    <Card className="p-6">
      <header>
        <h2 className="text-base font-semibold text-slate-900">SCIM 同期</h2>
        <p className="mt-1 text-sm text-slate-600">
          Okta や Google IAM などからユーザー・グループを自動同期するための設定です。有効な SCIM
          アクセストークンを発行すると、外部 IDP から同期できるようになります。
        </p>
      </header>

      <div className="mt-6 grid gap-6">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />

        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
          <h3 className="text-sm font-semibold text-slate-900">接続情報</h3>
          <div className="mt-3 grid gap-3">
            <div>
              <span className="text-xs text-slate-500">SCIM 2.0 Base URL (エンドポイント)</span>
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
                    setNotice('URLをクリップボードにコピーしました。')
                  }}
                >
                  コピー
                </Button>
              </div>
              <p className="mt-1 text-xs text-slate-500">
                外部 IDP の SCIM コネクタ設定でこの URL を指定します。
              </p>
            </div>
          </div>
        </div>

        <div className="grid gap-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="text-sm font-semibold text-slate-900">SCIM アクセストークン</h3>
            {!creating ? (
              <Button type="button" variant="outline" onClick={() => setCreating(true)}>
                トークンを発行
              </Button>
            ) : null}
          </div>

          {generatedToken ? (
            <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-4">
              <h4 className="text-sm font-bold text-emerald-800">
                発行された SCIM アクセストークン
              </h4>
              <p className="mt-1 text-xs text-emerald-700">
                セキュリティのため、このトークンは一度しか表示されません。必ずコピーして安全な場所に保管してください。
              </p>
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
                    setNotice('トークンをコピーしました。')
                  }}
                >
                  コピー
                </Button>
              </div>
            </div>
          ) : null}

          {tokens.length === 0 ? (
            <p className="text-sm text-slate-500">
              有効なアクセストークンがありません。「トークンを発行」から作成してください。
            </p>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-slate-200">
              <table className="min-w-full divide-y divide-slate-200 text-left text-sm text-slate-700">
                <thead className="bg-slate-50 font-semibold text-slate-900">
                  <tr>
                    <th className="px-4 py-2">説明</th>
                    <th className="px-4 py-2">作成日時</th>
                    <th className="px-4 py-2">有効期限</th>
                    <th className="px-4 py-2">アクション</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {tokens.map((tok) => (
                    <tr key={tok.id}>
                      <td className="px-4 py-3">{tok.description}</td>
                      <td className="px-4 py-3">{new Date(tok.created_at).toLocaleString()}</td>
                      <td className="px-4 py-3">
                        {tok.expires_at ? new Date(tok.expires_at).toLocaleString() : 'なし'}
                      </td>
                      <td className="px-4 py-3">
                        <Button
                          type="button"
                          variant="ghost"
                          className="text-red-600 hover:text-red-700 hover:bg-red-50"
                          onClick={() => handleRevokeToken(tok.id)}
                        >
                          失効
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
              <h4 className="text-sm font-semibold text-slate-900">新規アクセストークンの作成</h4>
              <div className="mt-3 grid gap-4 sm:grid-cols-2">
                <div className="grid gap-1.5">
                  <Label htmlFor="token-desc">トークンの説明</Label>
                  <Input
                    id="token-desc"
                    placeholder="例: Okta-SCIM-Integration"
                    value={tokenDesc}
                    onChange={(e) => setTokenDesc(e.target.value)}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="token-expiry">有効期間 (日数)</Label>
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
                <Button type="submit">トークンを発行</Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setTokenDesc('')
                    setError('')
                    setCreating(false)
                  }}
                >
                  キャンセル
                </Button>
              </div>
            </form>
          ) : null}
        </div>
      </div>
    </Card>
  )
}

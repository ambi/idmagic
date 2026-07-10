import {
  IconActivity,
  IconArrowRight,
  IconCheckupList,
  IconChevronRight,
  IconKey,
  IconShieldCheck,
  IconShieldLock,
  IconUserPlus,
  IconUsers,
  IconLayoutGrid,
} from '@tabler/icons-react'
import { tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Card } from '../../components/ui/card'
import { cn } from '../../lib/utils'
import type { AdminAuditEvent } from '../../types'

export function AdminDashboardPage({
  actorUsername,
  actorRoles,
  userCount,
  activeUserCount,
  disabledUserCount,
  clientCount,
  grantedConsentCount,
  auditEventCount24h,
  recentEvents,
}: {
  actorUsername?: string
  actorRoles: string[]
  userCount: number
  activeUserCount: number
  disabledUserCount: number
  clientCount: number
  grantedConsentCount: number
  auditEventCount24h: number
  recentEvents: AdminAuditEvent[]
}) {
  const activeRate = userCount > 0 ? Math.round((activeUserCount / userCount) * 100) : 0

  // テナントのセキュリティ状態を評価する擬似スコア
  // ユーザー有効率や、クライアント登録、同意付与などを加味して算出
  const securityScore = Math.min(
    100,
    Math.max(
      40,
      Math.round(
        activeRate * 0.8 + (grantedConsentCount > 0 ? 10 : 0) + (clientCount > 0 ? 10 : 0),
      ),
    ),
  )

  return (
    <AdminShell
      active="dashboard"
      actorUsername={actorUsername}
      title="ダッシュボード"
      description="アイデンティティとアクセスの状況をリアルタイムで監視・評価します。"
    >
      {/* テナント全体のセキュリティ・ヘルスステータス (Okta / Entra風の大規模カード) */}
      <section className="mb-6">
        <Card className="relative overflow-hidden bg-gradient-to-br from-slate-900 via-slate-950 to-blue-950 p-6 text-white shadow-xl ring-1 ring-slate-800">
          <div className="absolute -right-10 -top-10 opacity-10">
            <IconShieldCheck size={200} className="text-white" />
          </div>
          <div className="grid gap-6 md:grid-cols-[1fr_220px]">
            <div className="flex flex-col justify-between">
              <div>
                <span className="inline-flex items-center gap-1.5 rounded-full bg-blue-500/10 px-3 py-1 text-xs font-semibold text-blue-400 ring-1 ring-inset ring-blue-500/20">
                  <IconShieldCheck size={14} />
                  テナントセキュリティ評価
                </span>
                <h2 className="mt-3 text-2xl font-bold tracking-tight">IdMagic テナント設定</h2>
                <p className="mt-2 text-sm leading-relaxed text-slate-300">
                  テナントの基本的なアイデンティティ設定は完了しています。MFAの強制化や Entra ID
                  とのドメインフェデレーションを構成することで、セキュリティスコアをさらに向上させることができます。
                </p>
              </div>

              <div className="mt-6 flex flex-wrap gap-4 text-xs">
                <div className="rounded-lg bg-white/5 px-3 py-2 border border-white/10">
                  <span className="block text-slate-400">テナント状態</span>
                  <span className="mt-0.5 block font-semibold text-emerald-400">正常稼働中</span>
                </div>
                <div className="rounded-lg bg-white/5 px-3 py-2 border border-white/10">
                  <span className="block text-slate-400">アクティブユーザー率</span>
                  <span className="mt-0.5 block font-semibold">{activeRate}%</span>
                </div>
                <div className="rounded-lg bg-white/5 px-3 py-2 border border-white/10">
                  <span className="block text-slate-400">過去24時間の監査イベント</span>
                  <span className="mt-0.5 block font-semibold">{auditEventCount24h} 件</span>
                </div>
              </div>
            </div>

            <div className="flex flex-col items-center justify-center border-t border-white/10 pt-6 md:border-l md:border-t-0 md:pt-0 md:pl-6">
              <div className="relative flex size-32 items-center justify-center rounded-full border-4 border-slate-800">
                {/* 円形の進捗ゲージを簡易表現 */}
                <div className="text-center">
                  <span className="block text-4xl font-extrabold tracking-tight text-white">
                    {securityScore}
                  </span>
                  <span className="block text-[0.625rem] font-bold uppercase tracking-wider text-slate-400">
                    Security Score
                  </span>
                </div>
              </div>
              <span className="mt-3 text-xs font-semibold text-slate-400">
                推奨タスク {securityScore === 100 ? 'すべて完了' : '残り 2 件'}
              </span>
            </div>
          </div>
        </Card>
      </section>

      {/* システムサマリー (ビジュアルと価値を再検討した MetricCards) */}
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4 mb-6" aria-label="サマリー">
        <DashboardMetricCard
          label="総ユーザー"
          value={userCount}
          icon={IconUsers}
          tone="blue"
          extra={
            <div className="mt-3 border-t border-slate-100 pt-3">
              <div className="flex justify-between text-[0.68rem] font-semibold text-slate-500 mb-1">
                <span>有効率 {activeRate}%</span>
                <span>無効: {disabledUserCount}</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-slate-100 overflow-hidden">
                <div
                  className="h-full bg-blue-600 rounded-full transition-all"
                  style={{ width: `${activeRate}%` }}
                />
              </div>
            </div>
          }
        />
        <DashboardMetricCard
          label="登録アプリケーション"
          value={clientCount}
          icon={IconKey}
          tone="violet"
          extra={
            <div className="mt-3 border-t border-slate-100 pt-3">
              <div className="flex justify-between text-[0.68rem] text-slate-500">
                <span>連携クライアント</span>
                <span className="font-semibold text-slate-900">{clientCount} 個</span>
              </div>
              <p className="mt-1 text-[0.625rem] text-slate-400">
                OIDC RP または SAML SP の接続がアクティブです。
              </p>
            </div>
          }
        />
        <DashboardMetricCard
          label="付与済みの同意"
          value={grantedConsentCount}
          icon={IconCheckupList}
          tone="green"
          extra={
            <div className="mt-3 border-t border-slate-100 pt-3">
              <div className="flex justify-between text-[0.68rem] text-slate-500">
                <span>認可された同意</span>
                <span className="font-semibold text-slate-900">{grantedConsentCount} 件</span>
              </div>
              <p className="mt-1 text-[0.625rem] text-slate-400">
                エンドユーザーがアプリに権限付与しています。
              </p>
            </div>
          }
        />
        <DashboardMetricCard
          label="監査イベント (24h)"
          value={auditEventCount24h}
          icon={IconActivity}
          tone="amber"
          extra={
            <div className="mt-3 border-t border-slate-100 pt-3">
              <div className="flex items-center gap-1.5 text-[0.68rem] font-semibold">
                <span
                  className={cn(
                    'inline-block size-2 rounded-full',
                    auditEventCount24h > 50 ? 'bg-amber-500 animate-pulse' : 'bg-emerald-500',
                  )}
                />
                <span className="text-slate-500">
                  {auditEventCount24h > 50 ? 'トラフィック上昇傾向' : 'アクティビティ正常'}
                </span>
              </div>
            </div>
          }
        />
      </section>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        {/* 左カラム: 直近の監査イベント & 推奨タスク */}
        <div className="grid gap-6">
          {/* 直近の監査イベント */}
          <Card className="overflow-hidden shadow-sm">
            <div className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
              <div>
                <h2 className="text-sm font-semibold text-slate-900">直近の監査イベント</h2>
                <p className="mt-0.5 text-xs text-slate-500">
                  テナント内で過去 24 時間に記録された主要アクションです。
                </p>
              </div>
              <a
                href={tenantURL('/admin/audit_events')}
                className="inline-flex items-center gap-1 text-xs font-semibold text-blue-700 hover:text-blue-800"
              >
                すべて表示
                <IconChevronRight size={14} aria-hidden="true" />
              </a>
            </div>
            {recentEvents.length === 0 ? (
              <div className="px-5 py-10 text-center text-sm text-slate-500">
                直近 24 時間に記録された監査イベントはありません。
              </div>
            ) : (
              <ul className="divide-y divide-slate-100">
                {recentEvents.map((event) => (
                  <li key={event.id}>
                    <a
                      href={eventLink(event)}
                      className="flex items-start gap-3 px-5 py-3.5 transition-colors hover:bg-slate-50"
                    >
                      <span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-500">
                        <IconActivity size={16} aria-hidden="true" />
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-semibold text-slate-900">
                          {friendlyEventName(event.type)}
                        </p>
                        <p className="mt-0.5 truncate text-xs text-slate-500">
                          {formatDateTime(event.occurred_at)} · {summarizeActor(event)}
                        </p>
                      </div>
                      <IconChevronRight
                        size={16}
                        className="mt-1 shrink-0 text-slate-400"
                        aria-hidden="true"
                      />
                    </a>
                  </li>
                ))}
              </ul>
            )}
          </Card>

          {/* セキュリティ推奨タスク (Okta / IAM風) */}
          <Card className="p-5 shadow-sm">
            <div className="flex items-start gap-2.5">
              <IconShieldCheck className="text-blue-600 shrink-0 mt-0.5" size={20} />
              <div>
                <h2 className="text-sm font-semibold text-slate-900">推奨セキュリティ構成</h2>
                <p className="mt-0.5 text-xs text-slate-500">
                  IdMagic
                  が推奨する初期セキュリティ構成タスクです。テナントを保護するために実施してください。
                </p>
              </div>
            </div>

            <ul className="mt-4 grid gap-3.5 sm:grid-cols-2">
              <SecurityTaskCard
                title="MFA（二要素認証）の強制化"
                description="不正アクセス防止のため、サインインポリシーでMFA認証を「必須」に設定することを強く推奨します。"
                href={tenantURL('/admin/sign-in-policy')}
                actionLabel="ポリシーを設定"
              />
              <SecurityTaskCard
                title="外部 IDP とのドメイン連携"
                description="Microsoft 365 などの Entra ID ドメインとのフェデレーションを行い、ログイン統合を行います。"
                href={tenantURL('/admin/federation/entra')}
                actionLabel="フェデレーションを構成"
              />
            </ul>
          </Card>
        </div>

        {/* 右カラム: クイックリンク (実務に絞ったもの) */}
        <div className="grid gap-6 self-start">
          <Card className="p-5 shadow-sm bg-slate-50/30">
            <div className="flex items-center gap-2 mb-4">
              <IconLayoutGrid size={18} className="text-slate-500" />
              <h2 className="text-sm font-semibold text-slate-900">管理者クイックリンク</h2>
            </div>
            <ul className="grid gap-2">
              <DashboardQuickLink
                href={tenantURL('/admin/users/new')}
                icon={IconUserPlus}
                label="ユーザーを追加"
                description="新しい組織アカウントを作成します。"
              />
              <DashboardQuickLink
                href={tenantURL('/admin/applications')}
                icon={IconKey}
                label="アプリケーション管理"
                description="OIDC クライアントや SAML SP を管理します。"
              />
              <DashboardQuickLink
                href={tenantURL('/admin/sign-in-policy')}
                icon={IconShieldLock}
                label="サインインポリシー"
                description="パスワードポリシーやMFA要求ルールを設定します。"
              />
              <DashboardQuickLink
                href={tenantURL('/admin/audit_events')}
                icon={IconActivity}
                label="監査イベントビュー"
                description="テナント内の全イベントをフィルタ・追跡します。"
              />
              {actorRoles.includes('system_admin') ? (
                <DashboardQuickLink
                  href="/system"
                  icon={IconShieldCheck}
                  label="システムコンソール"
                  description="全テナントの監視とマルチテナント管理を行います。"
                />
              ) : null}
            </ul>
          </Card>
        </div>
      </div>
    </AdminShell>
  )
}

export function DashboardMetricCard({
  label,
  value,
  icon: Icon,
  tone,
  extra,
}: {
  label: string
  value: number
  icon: typeof IconUsers
  tone: 'blue' | 'green' | 'violet' | 'amber'
  extra?: React.ReactNode
}) {
  const tones = {
    blue: 'bg-blue-50 text-blue-700 ring-blue-100',
    green: 'bg-emerald-50 text-emerald-700 ring-emerald-100',
    violet: 'bg-indigo-50 text-indigo-700 ring-indigo-100',
    amber: 'bg-amber-50 text-amber-700 ring-amber-100',
  }
  return (
    <Card className="group p-5 transition-[border-color,box-shadow,transform] hover:-translate-y-0.5 hover:border-slate-300 hover:shadow-md">
      <div className="flex items-center gap-4">
        <span
          className={cn(
            'flex size-11 shrink-0 items-center justify-center rounded-xl ring-1',
            tones[tone],
          )}
        >
          <Icon size={22} stroke={1.8} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-3xl font-bold tracking-tight text-slate-950">{value}</p>
          <p className="truncate text-xs font-semibold text-slate-500">{label}</p>
        </div>
      </div>
      {extra}
    </Card>
  )
}

export function DashboardQuickLink({
  href,
  icon: Icon,
  label,
  description,
}: {
  href: string
  icon: typeof IconUsers
  label: string
  description: string
}) {
  return (
    <li>
      <a
        href={href}
        className="flex items-start gap-3 rounded-lg border border-slate-200/80 bg-white p-3 transition-[background-color,border-color,box-shadow] hover:border-slate-300 hover:bg-slate-50 hover:shadow-xs"
      >
        <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-700">
          <Icon size={18} stroke={1.8} aria-hidden="true" />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-semibold text-slate-900">{label}</span>
          <span className="mt-0.5 block text-xs leading-relaxed text-slate-500">{description}</span>
        </span>
        <IconArrowRight size={16} className="mt-1 shrink-0 text-slate-400" aria-hidden="true" />
      </a>
    </li>
  )
}

export function SecurityTaskCard({
  title,
  description,
  href,
  actionLabel,
}: {
  title: string
  description: string
  href: string
  actionLabel: string
}) {
  return (
    <li className="flex flex-col justify-between rounded-lg border border-slate-200 bg-slate-50/50 p-4 transition-all hover:bg-slate-50">
      <div>
        <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
          <span className="inline-block size-1.5 rounded-full bg-blue-600" />
          {title}
        </h3>
        <p className="mt-1.5 text-[0.68rem] leading-relaxed text-slate-500">{description}</p>
      </div>
      <div className="mt-4 flex justify-end">
        <a
          href={href}
          className="inline-flex items-center gap-1 text-[0.68rem] font-bold text-blue-700 hover:text-blue-800"
        >
          {actionLabel}
          <IconArrowRight size={12} />
        </a>
      </div>
    </li>
  )
}

function summarizeActor(event: AdminAuditEvent): string {
  const payload = event.payload as { actor_sub?: string; sub?: string; target_sub?: string }
  const actor = payload.actor_sub ?? payload.sub
  const target = payload.target_sub
  if (actor && target && actor !== target) return `${actor} → ${target}`
  if (actor) return actor
  if (target) return target
  return event.tenant_id
}

function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('ja-JP', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function friendlyEventName(type: string): string {
  const map: Record<string, string> = {
    // ユーザー・グループ関連
    UserCreated: 'ユーザーの作成',
    UserUpdated: 'ユーザー情報の更新',
    UserDeleted: 'ユーザーの削除',
    UserDisabled: 'ユーザーの無効化',
    UserEnabled: 'ユーザーの有効化',
    UserGroupAdded: 'ユーザーをグループに追加',
    UserGroupRemoved: 'ユーザーをグループから削除',
    UserRequiredActionSet: '強制アクションの付与',
    UserRequiredActionCleared: '強制アクションの解除',
    // 認証・セッション関連
    UserAuthenticated: 'ユーザー認証の成功',
    LoginSucceeded: 'サインイン成功',
    LoginFailed: 'サインイン失敗',
    LoginThrottled: '連続サインイン失敗によるアカウントロック',
    RefreshTokenIssued: 'トークン更新',
    TokenRevoked: 'トークンの無効化',
    MfaEnrolled: 'MFA（二要素認証）有効化',
    MfaBypassed: 'MFA一時解除',
    PasswordResetRequested: 'パスワードリセット要求',
    PasswordResetCompleted: 'パスワード変更完了',
    EmailChangeRequested: 'メールアドレス変更要求',
    EmailChangeConfirmed: 'メールアドレス変更完了',
    // アプリ関連
    ClientCreated: 'アプリケーションの登録',
    ClientUpdated: 'アプリケーション設定の更新',
    ClientDeleted: 'アプリケーションの削除',
    ConsentGranted: '同意の付与',
    ConsentRevoked: '同意の解除',
    // OAuth/OIDCイベント関連
    AuthorizationCodeIssued: '認可コードの発行',
    AuthorizationCodeRedeemed: '認可コードの検証 (トークン交換)',
    AccessTokenIssued: 'アクセストークンの発行',
    PARRequestRegistered: 'PAR (Push Authorization Request) 登録',
    ParRequestRegistered: 'PAR (Push Authorization Request) 登録',
    DeviceCodeIssued: 'デバイスコードの発行',
    DeviceCodeAuthorized: 'デバイスコードの認可',
    RPLogoutInitiated: 'ログアウト要求の開始',
    RpLogoutInitiated: 'ログアウト要求の開始',
    // その他
    AgentCreated: 'エージェントの作成',
    AgentUpdated: 'エージェント情報の更新',
    AgentDeleted: 'エージェントの削除',
    SigningKeyRotated: '署名鍵のローテーション',
    TenantUpdated: 'テナント設定の更新',
  }

  // 大文字小文字のゆらぎに対応するため、小文字に直してマッチング
  const lowerType = type.toLowerCase()
  const found = Object.entries(map).find(([key]) => key.toLowerCase() === lowerType)
  if (found) return found[1]

  // マップにない場合のフォールバック: PascalCaseをスペース区切りにする
  return type.replace(/([A-Z])/g, ' $1').trim()
}

function eventLink(event: AdminAuditEvent): string {
  const payload = event.payload as {
    user_id?: string
    sub?: string
    client_id?: string
    application_id?: string
  }
  const userId = payload.user_id || payload.sub
  const clientId = payload.client_id || payload.application_id

  if (event.type.startsWith('User') && userId) {
    return tenantURL(`/admin/users/${encodeURIComponent(userId)}`)
  }
  if (event.type.startsWith('Client') && clientId) {
    return tenantURL(`/admin/applications/${encodeURIComponent(clientId)}`)
  }
  if (event.type.startsWith('Application') && clientId) {
    return tenantURL(`/admin/applications/${encodeURIComponent(clientId)}`)
  }
  return `${tenantURL('/admin/audit_events')}?type=${encodeURIComponent(event.type)}`
}

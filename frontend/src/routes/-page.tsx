import { useEffect, type ReactNode } from 'react'

// markPage は描画したページ種別を <meta name="idmagic:page"> で DOM に表明する。
// SPA dispatcher の分岐を E2E から機械的に検証するための不変条件マーカー (wi-22)。
function markPage(kind: string) {
  let meta = document.head.querySelector<HTMLMetaElement>('meta[name="idmagic:page"]')
  if (!meta) {
    meta = document.createElement('meta')
    meta.name = 'idmagic:page'
    document.head.appendChild(meta)
  }
  meta.content = kind
}

// PAGE_TITLES は各ページ種別 (PageMarker kind) を、現在地が分かるタブタイトルへ
// 対応づける正本 (wi-126 §3)。ポータル別のブランド接尾辞を付け、利用者がタブだけで
// 管理コンソール / マイページ / システム管理 / ログインのどこにいるか分かるようにする。
// 未定義の kind は素の "IdMagic" にフォールバックする。
const BRAND = 'IdMagic'
const ADMIN = `${BRAND} 管理コンソール`
const ACCOUNT = `${BRAND} マイページ`
const SYSTEM = `${BRAND} システム管理`

const PAGE_TITLES: Record<string, string> = {
  // 管理コンソール
  'admin-dashboard': `ダッシュボード | ${ADMIN}`,
  'admin-users': `ユーザー | ${ADMIN}`,
  'admin-user-detail': `ユーザー詳細 | ${ADMIN}`,
  'admin-user-create': `ユーザーを追加 | ${ADMIN}`,
  'admin-user-edit': `ユーザーを編集 | ${ADMIN}`,
  'admin-agents': `エージェント | ${ADMIN}`,
  'admin-agent-detail': `エージェント詳細 | ${ADMIN}`,
  'admin-applications': `アプリケーション | ${ADMIN}`,
  'admin-application-detail': `アプリケーション詳細 | ${ADMIN}`,
  'admin-application-edit': `アプリケーション編集 | ${ADMIN}`,
  'admin-groups': `グループ | ${ADMIN}`,
  'admin-group-detail': `グループ詳細 | ${ADMIN}`,
  'admin-group-create': `グループを追加 | ${ADMIN}`,
  'admin-group-edit': `グループを編集 | ${ADMIN}`,
  'admin-roles': `ロール | ${ADMIN}`,
  'admin-role-detail': `ロール詳細 | ${ADMIN}`,
  'admin-consents': `同意 | ${ADMIN}`,
  'admin-audit-events': `監査イベント | ${ADMIN}`,
  'admin-authz-detail-types': `認可詳細の種類 | ${ADMIN}`,
  'admin-keys': `署名鍵 | ${ADMIN}`,
  'admin-settings': `設定 | ${ADMIN}`,
  'admin-sign-in-policy': `サインインポリシー | ${ADMIN}`,
  'admin-entra-federation': `Entra ID 連携 | ${ADMIN}`,
  'admin-entra-federation-add': `ドメインフェデレーションを追加 | ${ADMIN}`,
  'admin-tenant-attributes': `テナント属性 | ${ADMIN}`,
  // マイページ
  'account-home': `アカウント情報 | ${ACCOUNT}`,
  'account-profile': `プロフィール | ${ACCOUNT}`,
  'account-profile-edit': `プロフィール編集 | ${ACCOUNT}`,
  'account-emails': `メールアドレス | ${ACCOUNT}`,
  'account-security': `セキュリティ | ${ACCOUNT}`,
  'account-applications': `連携アプリ | ${ACCOUNT}`,
  'account-apps': `連携アプリ | ${ACCOUNT}`,
  'account-activity': `アクティビティ | ${ACCOUNT}`,
  'account-data': `データ | ${ACCOUNT}`,
  'change-password': `パスワード変更 | ${ACCOUNT}`,
  'email-verify': `メールアドレスの確認 | ${ACCOUNT}`,
  // システム管理
  'system-tenants': `テナント | ${SYSTEM}`,
  'system-key-health': `署名鍵の健全性 | ${SYSTEM}`,
  // ログイン / 認証フロー
  login: `ログイン | ${BRAND}`,
  totp: `二要素認証 | ${BRAND}`,
  consent: `アクセスの許可 | ${BRAND}`,
  device: `デバイス認証 | ${BRAND}`,
  'forgot-password': `パスワードの再設定 | ${BRAND}`,
  'reset-password': `パスワードの再設定 | ${BRAND}`,
  callback: `サインイン処理中 | ${BRAND}`,
  status: `ステータス | ${BRAND}`,
  home: BRAND,
  error: BRAND,
}

function markTitle(kind: string) {
  document.title = PAGE_TITLES[kind] ?? BRAND
}

export function PageMarker({ kind, children }: { kind: string; children: ReactNode }) {
  useEffect(() => {
    markPage(kind)
    markTitle(kind)
  }, [kind])
  return children
}

export function markErrorPage() {
  markPage('error')
  markTitle('error')
}

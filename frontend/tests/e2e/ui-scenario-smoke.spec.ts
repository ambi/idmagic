// wi-75: SCL に追加済みの UI シナリオを、まず画面到達性と主要導線の
// ブラウザ E2E として固定する。低レベル usecase の重複検査ではなく、
// SPA route loader、OIDC RP ログイン、サイドバー遷移、フォーム送信の接続を検証する。
import { test } from 'bun:test'
import {
  clickNavLinkByAnyText,
  demo,
  hasText,
  uiOrigin,
  waitForLocationPath,
  waitForPage,
  waitForAnyText,
  navigateAndLogin,
} from './fixtures'

test('login assistance pages render and forgot password has enumeration-safe success copy', async () => {
  const view = new Bun.WebView({ width: 1280, height: 1600 })
  try {
    await view.navigate(`${uiOrigin}/forgot_password`)
    await waitForPage(view, 'forgot-password')
    await view.click('input[name="email"]')
    await view.type(demo.email)
    await view.click('button[type="submit"]')

    await waitForAnyText(view, ['アカウントが確認できた場合', 'If an account exists'])

    await view.navigate(`${uiOrigin}/reset_password`)
    await waitForPage(view, 'reset-password')

    await waitForAnyText(view, ['リセットリンクが不正です。', 'The reset link is invalid.'])
  } finally {
    view.close()
  }
}, 60_000)

test('account portal scenarios are reachable after account-audience login', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    await navigateAndLogin(view, '/account', 'account-home')

    const pages = [
      [['アプリ', 'Apps'], '/account/apps', 'account-apps'],
      [['アカウント情報', 'Profile'], '/account/profile', 'account-profile'],
      [['メールアドレス', 'Email addresses'], '/account/emails', 'account-emails'],
      [['セキュリティ', 'Security'], '/account/security', 'account-security'],
      [['アクティビティ', 'Activity'], '/account/activity', 'account-activity'],
      [['接続済みアプリ', 'Connected apps'], '/account/applications', 'account-applications'],
      [['データとプライバシー', 'Data and privacy'], '/account/data', 'account-data'],
    ] as const

    for (const [label, path, marker] of pages) {
      await clickNavLinkByAnyText(view, ['マイページメニュー', 'Account navigation'], [...label])
      await waitForLocationPath(view, path)
      await waitForPage(view, marker)
    }
  } finally {
    view.close()
  }
}, 90_000)

test('admin console scenarios are reachable after admin-audience login', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2200 })
  try {
    await navigateAndLogin(view, '/admin', 'admin-dashboard')

    const pages = [
      [['アプリケーション', 'Applications'], '/admin/applications', 'admin-applications'],
      [['グループ', 'Groups'], '/admin/groups', 'admin-groups'],
      [['エージェント', 'Agents'], '/admin/agents', 'admin-agents'],
      [['監査イベント', 'Audit events'], '/admin/audit_events', 'admin-audit-events'],
      [
        ['MCP リソースサーバー', 'MCP resource servers'],
        '/admin/mcp-resource-servers',
        'admin-mcp-resource-servers',
      ],
      [['署名鍵', 'Signing keys'], '/admin/keys', 'admin-keys'],
      [['ユーザー属性', 'User attributes'], '/admin/tenant/attributes', 'admin-tenant-attributes'],
      [['設定', 'Settings'], '/admin/settings', 'admin-settings'],
    ] as const

    for (const [label, path, marker] of pages) {
      await clickNavLinkByAnyText(view, ['管理メニュー', 'Admin navigation'], [...label])
      await waitForLocationPath(view, path)
      await waitForPage(view, marker)
    }
  } finally {
    view.close()
  }
}, 90_000)

// EX-SYSTEM-020-01、EX-SYSTEM-020-03: テナント横断の監査とジョブはシステムコンソールだけに
// あり、テナント管理コンソールには横断への入口が無い (REQ-SYSTEM-020)。経路も配線も本物を
// 通すので、画面の分離と API の分離がどちらも成立していないと通らない。
test('system console cross-tenant surfaces are reachable and the admin console has no cross-tenant toggle', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2200 })
  try {
    await navigateAndLogin(view, '/system/tenants', 'system-tenants', demo.systemAdminUsername)

    const systemPages = [
      [['監査イベント', 'Audit events'], '/system/audit-events', 'system-audit-events'],
      [['非同期ジョブ', 'Background jobs'], '/system/jobs', 'system-jobs'],
    ] as const

    for (const [label, path, marker] of systemPages) {
      await clickNavLinkByAnyText(view, ['システムメニュー', 'System navigation'], [...label])
      await waitForLocationPath(view, path)
      await waitForPage(view, marker)
    }

    // 同じ操作者がテナント管理コンソールへ移ると、横断の切替はどこにも無い。
    for (const [path, marker] of [
      ['/admin/jobs', 'admin-jobs'],
      ['/admin/audit_events', 'admin-audit-events'],
    ] as const) {
      await view.navigate(`${uiOrigin}${path}`)
      await waitForPage(view, marker)
      for (const wording of ['全テナント横断', 'Across all tenants']) {
        if (await hasText(view, wording)) {
          throw new Error(`${path} still offers a cross-tenant toggle: ${wording}`)
        }
      }
    }
  } finally {
    view.close()
  }
}, 120_000)

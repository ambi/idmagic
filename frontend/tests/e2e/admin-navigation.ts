import { POLL_INTERVAL_MS } from './polling'

// 製品のメニュー定義から生成せず、E2E がたどる表示名と遷移先を固定する。
export const adminNavigationPages = [
  {
    key: 'applications',
    label: ['アプリケーション', 'Applications'],
    group: ['アプリケーション連携', 'Application integrations'],
    path: '/admin/applications',
    marker: 'admin-applications',
  },
  {
    key: 'groups',
    label: ['グループ', 'Groups'],
    group: ['ディレクトリ', 'Directory'],
    path: '/admin/groups',
    marker: 'admin-groups',
  },
  {
    key: 'agents',
    label: ['エージェント一覧', 'Agent list'],
    group: ['エージェント', 'Agents'],
    path: '/admin/agents',
    marker: 'admin-agents',
  },
  {
    key: 'audit-events',
    label: ['監査イベント', 'Audit events'],
    group: ['運用と設定', 'Operations and settings'],
    path: '/admin/audit_events',
    marker: 'admin-audit-events',
  },
  {
    key: 'mcp-resource-servers',
    label: ['MCP リソースサーバー', 'MCP resource servers'],
    group: ['エージェント', 'Agents'],
    path: '/admin/mcp-resource-servers',
    marker: 'admin-mcp-resource-servers',
  },
  {
    key: 'cross-app-access',
    label: ['アプリ間アクセス', 'Cross-app access'],
    group: ['アクセス制御', 'Access control'],
    path: '/admin/cross-app-access',
    marker: 'admin-cross-app-access',
  },
  {
    key: 'keys',
    label: ['署名鍵', 'Signing keys'],
    group: ['運用と設定', 'Operations and settings'],
    path: '/admin/keys',
    marker: 'admin-keys',
  },
  {
    key: 'tenant-attributes',
    label: ['ユーザー属性', 'User attributes'],
    group: ['ディレクトリ', 'Directory'],
    path: '/admin/tenant/attributes',
    marker: 'admin-tenant-attributes',
  },
  {
    key: 'settings',
    label: ['設定', 'Settings'],
    group: ['運用と設定', 'Operations and settings'],
    path: '/admin/settings',
    marker: 'admin-settings',
  },
] as const

export async function clickAdminNavigation(
  view: Pick<Bun.WebView, 'evaluate'>,
  page: (typeof adminNavigationPages)[number],
): Promise<void> {
  const state = await view.evaluate(`(() => {
    const nav = [...document.querySelectorAll('nav')]
      .find((node) => ['管理メニュー', 'Admin navigation'].includes(node.getAttribute('aria-label')))
    const link = [...(nav?.querySelectorAll('a') ?? [])]
      .find((node) => ${JSON.stringify(page.label)}.includes(node.textContent.trim()))
    if (link) { link.click(); return 'clicked' }
    const section = [...(nav?.querySelectorAll('section') ?? [])]
      .find((node) => ${JSON.stringify(page.group)}.includes(node.getAttribute('aria-label')))
    const trigger = section?.querySelector('button[aria-haspopup="menu"]')
    if (!trigger) return 'missing'
    trigger.click()
    return 'opened'
  })()`)
  if (state === 'clicked') return
  if (state === 'opened') {
    const deadline = Date.now() + 10_000
    while (Date.now() < deadline) {
      const clicked = await view.evaluate(`(() => {
        const link = [...document.querySelectorAll('[role="menuitem"]')]
          .find((node) => ${JSON.stringify(page.label)}.includes(node.textContent.trim()))
        if (!link) return false
        link.click()
        return true
      })()`)
      if (clicked === true) return
      await Bun.sleep(POLL_INTERVAL_MS)
    }
  }
  throw new Error(`admin navigation link not found: ${page.path}`)
}

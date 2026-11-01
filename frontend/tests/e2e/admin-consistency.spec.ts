import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  clickEnabledButtonByText,
  clickLinkByText,
  clickSummaryByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

//spec:covers REQ-FRONTEND-002: 一覧から専用の追加画面を経由し、保存した接続先を再取得する。
test('connector creation uses a dedicated route', async () => {
  const view = await openWebView({ width: 1440, height: 1100 })
  try {
    await navigateAndLogin(view, '/admin/token-vault', 'admin-token-vault')
    expect(await view.evaluate(`document.querySelectorAll('main form').length`)).toBe(0)
    await clickLinkByText(view, 'Add destination')
    await waitForPage(view, 'admin-token-vault-create')
    expect(
      await view.evaluate(
        `document.querySelector('#service-connection-client_secret').getAttribute('autocomplete')`,
      ),
    ).toBe('new-password')
    expect(
      await view.evaluate(`document.querySelector('#service-connection-revocation_endpoint').type`),
    ).toBe('url')
    const layout = await view.evaluate<{
      twoColumns: boolean
      sameRow: boolean
      width: number
    }>(`(() => {
      const name = document.querySelector('#service-connection-name').getBoundingClientRect()
      const client = document.querySelector('#service-connection-client_id').getBoundingClientRect()
      return { twoColumns: client.left > name.left + 100, sameRow: Math.abs(client.top - name.top) < 5, width: document.querySelector('main form').getBoundingClientRect().width }
    })()`)
    expect(layout.twoColumns).toBe(true)
    expect(layout.sameRow).toBe(true)
    expect(layout.width).toBeGreaterThan(750)

    const name = `Calendar ${Date.now()}`
    await setInputValue(view, '#service-connection-name', name)
    await setInputValue(view, '#service-connection-client_id', 'calendar-agent')
    await setInputValue(
      view,
      '#service-connection-authorization_endpoint',
      'https://calendar.example/authorize',
    )
    await setInputValue(
      view,
      '#service-connection-token_endpoint',
      'https://calendar.example/token',
    )
    await setInputValue(view, '#service-connection-client_secret', 'example-machine-secret')
    await setInputValue(view, '#service-connection-allowed_scopes', 'read')
    await clickButtonByText(view, 'Add destination')
    await waitForPage(view, 'admin-token-vault')

    await waitForText(view, name)
    expect(await view.evaluate(`document.querySelectorAll('main form').length`)).toBe(0)
  } finally {
    view.close()
  }
}, 60_000)

//spec:covers REQ-FRONTEND-001: 件数と更新の配置、エージェントの代理実行先の分類を正式なルートで観測する。
test('resource lists share counts and refresh while agent destinations stay grouped', async () => {
  const view = await openWebView({ width: 1440, height: 1100 })
  try {
    await navigateAndLogin(view, '/admin/groups', 'admin-groups')
    for (const [path, marker] of [
      ['groups', 'admin-groups'],
      ['agents', 'admin-agents'],
      ['users', 'admin-users'],
      ['roles', 'admin-roles'],
    ]) {
      await view.navigate(`${uiOrigin}/admin/${path}`)
      await waitForPage(view, marker)
      expect(
        await view.evaluate(
          `document.querySelector('main section[aria-label="List count and refresh"]')?.textContent?.includes('total')`,
        ),
      ).toBe(true)
      await clickButtonByText(view, 'Refresh')
      await waitForText(view, 'Refresh')
    }
    await view.navigate(`${uiOrigin}/admin/token-vault`)
    await waitForPage(view, 'admin-token-vault')
    expect(
      await view.evaluate(`(() => {
      const link = document.querySelector('aside a[href$="/admin/token-vault"]')
      return link?.closest('details')?.querySelector('summary')?.textContent
    })()`),
    ).toBe('Agents')
  } finally {
    view.close()
  }
}, 60_000)

//spec:covers REQ-APITOKENS-005: ポータルの正式な入口から個人用トークンを発行し、本人が確認して失効する。
test('personal token issuance and revocation work from the account portal', async () => {
  const view = await openWebView({ width: 1280, height: 1100 })
  try {
    await navigateAndLogin(view, '/account/api-tokens', 'account-api-tokens')
    await waitForText(view, 'Connection info')
    await clickLinkByText(view, 'Issue token')
    await waitForPage(view, 'account-api-token-create')
    const description = `Personal ${Date.now()}`
    await setInputValue(view, '#token-desc', description)
    await clickSummaryByText(view, 'Account API')
    expect(
      await view.evaluate(`document.querySelector('input[value="users:write"]') === null`),
    ).toBe(true)
    await view.click('input[value="account:read"]')
    await clickButtonByText(view, 'Issue token')
    await waitForText(view, 'Back to token list')
    await clickLinkByText(view, 'Back to token list')
    await waitForPage(view, 'account-api-tokens')
    await waitForText(view, description)
    await clickEnabledButtonByText(view, 'Revoke')
    expect(await view.evaluate(`document.querySelector('[role="dialog"]') !== null`)).toBe(true)
    await view.click('[role="dialog"] button:last-child')
    await waitForText(view, 'Revoked')
  } finally {
    view.close()
  }
}, 60_000)

test('creation forms use one column on a narrow screen', async () => {
  const view = await openWebView({ width: 390, height: 844 })
  try {
    await navigateAndLogin(view, '/admin/token-vault/new', 'admin-token-vault-create')
    const layout = await view.evaluate<{ stacked: boolean; overflow: boolean }>(`(() => {
      const name = document.querySelector('#service-connection-name').getBoundingClientRect()
      const client = document.querySelector('#service-connection-client_id').getBoundingClientRect()
      return { stacked: client.top > name.top + 30, overflow: document.documentElement.scrollWidth > window.innerWidth }
    })()`)
    expect(layout.stacked).toBe(true)
    expect(layout.overflow).toBe(false)
  } finally {
    view.close()
  }
}, 60_000)

import { expect, test } from 'bun:test'
import {
  navigateAndLogin,
  openWebView,
  setInputValue,
  setSelectValue,
  waitForText,
  waitForPage,
  uiOrigin,
} from './fixtures'

//spec:covers REQ-OAUTH2-091: 管理画面からマネージド型を登録して公開 URL を得る。配置用の設定と秘密は表示しない。
test('administrator registers a hosted MCP connection', async () => {
  const view = await openWebView({ width: 1440, height: 1800 })
  try {
    const suffix = String(Date.now())
    await navigateAndLogin(
      view,
      '/realms/default/admin/mcp-resource-servers/new',
      'admin-mcp-resource-server-create',
    )
    // 上流の登録も公開画面から作る。
    const upstream = `https://managed-${suffix}.example/mcp`
    await setInputValue(view, '#name', `Upstream-${suffix}`)
    await setInputValue(view, '#resource', upstream)
    await setSelectValue(view, '#evaluator-client', '')
    await view.evaluate(`document.querySelector('form button[type="submit"]').click()`)
    await waitForPage(view, 'admin-mcp-resource-servers')
    await view.navigate(`${uiOrigin}/realms/default/admin/mcp-resource-servers/new`)
    await waitForPage(view, 'admin-mcp-resource-server-create')
    await setSelectValue(view, '#hosting-mode', 'managed')
    expect(await view.evaluate(`document.querySelector('#resource')===null`)).toBe(true)
    expect(await view.evaluate(`document.querySelector('#connector-id')===null`)).toBe(true)
    await setInputValue(view, '#name', `Managed-${suffix}`)
    await setInputValue(view, '#upstream-url', upstream)
    await view.evaluate(`document.querySelector('form button[type="submit"]').click()`)
    await waitForText(view, 'No gateway startup is needed.')
    const resource = String(
      await view.evaluate(`document.querySelector('#result-public-url')?.textContent`),
    )
    expect(resource).toContain('/realms/default/mcp/')
    expect(await view.evaluate(`document.querySelector('#result-client-secret')===null`)).toBe(true)
    expect(await view.evaluate(`document.querySelector('#result-gateway-config')===null`)).toBe(
      true,
    )
    expect(
      String(
        await view.evaluate(`document.querySelector('#result-gateway-public-key')?.textContent`),
      ),
    ).toContain('"kty": "RSA"')
    await waitForText(view, 'thumbprint (JKT)')
  } finally {
    await view.close()
  }
}, 90_000)

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

//spec:covers REQ-OAUTH2-091, REQ-OAUTH2-100: 管理画面から登録して接続案内を得て、再訪時にも同じ案内を取得する。
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
    await waitForText(view, 'MCP 2026-07-28 stateless HTTP')
    await waitForText(view, 'Do not distribute the evaluator client credentials')
    await view.navigate(`${uiOrigin}/realms/default/admin/mcp-resource-servers`)
    await waitForPage(view, 'admin-mcp-resource-servers')
    await view.evaluate(
      `document.querySelector('button[aria-label="Show details for Managed-${suffix}"]').click()`,
    )
    await waitForText(view, 'MCP 2026-07-28 stateless HTTP')
    expect(
      String(await view.evaluate(`document.querySelector('#result-public-url')?.textContent`)),
    ).toBe(resource)
  } finally {
    await view.close()
  }
}, 90_000)

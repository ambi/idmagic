import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  clickLinkByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  setSelectValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

//spec:covers REQ-AUTHORIZATION-017, REQ-OAUTH2-064: モデル画面を使わず、接続先と操作を登録し、Agent 起点の付与と接続先起点の取消を同じ保存へ反映する。
test('administrator assigns registered operations from resource connections', async () => {
  const view = await openWebView({ width: 1440, height: 1800 })
  try {
    const suffix = String(Date.now())
    const connection = `Documents-${suffix}`
    await navigateAndLogin(
      view,
      '/realms/default/admin/mcp-resource-servers/new',
      'admin-mcp-resource-server-create',
    )
    await setInputValue(view, '#resource', `https://catalog-${suffix}.example/mcp`)
    await setInputValue(view, '#name', connection)
    await setInputValue(view, '#scopes', 'documents:read')
    // Gateway を使わない接続先として登録し、登録後に一覧へ戻る。
    await setSelectValue(view, '#evaluator-client', '')
    await clickButtonByText(view, 'Register tool')
    await setInputValue(view, '#tool-name-0', 'read_document')
    await setInputValue(view, '#tool-label-0', 'Read documents')
    expect(await view.evaluate(`document.querySelector('#rule-type-0')===null`)).toBe(true)
    await view.evaluate(`document.querySelector('form button[type="submit"]').click()`)
    await waitForPage(view, 'admin-mcp-resource-servers')
    await view.navigate(`${uiOrigin}/realms/default/admin/agents/new`)
    await waitForPage(view, 'admin-agent-create')
    const agentName = `Catalog assistant-${suffix}`
    await setInputValue(view, '#agent-name', agentName)
    // 認証用クライアントはこのテストでは作らない。
    await view.evaluate(
      `document.querySelector('input[name="auth-method"][value="later"]').click()`,
    )
    await clickButtonByText(view, 'Register')
    await waitForPage(view, 'admin-agent-detail')
    const detailPath = String(await view.evaluate('window.location.pathname'))
    await view.evaluate(`document.querySelector('main a[href$="/resources"]').click()`)
    await waitForPage(view, 'admin-agent-resources')
    await clickLinkByText(view, 'Edit MCP resource servers and operations')
    await waitForPage(view, 'admin-agent-resources-edit')
    const serverID = String(
      await view.evaluate(
        `[...document.querySelector('#resource-connection').options].find(option=>option.textContent===${JSON.stringify(connection)})?.value`,
      ),
    )
    await setSelectValue(view, '#resource-connection', serverID)
    await view.evaluate(
      `[...document.querySelectorAll('label')].find(node=>node.textContent.includes('Read documents'))?.querySelector('input[type=checkbox]')?.click()`,
    )
    await clickButtonByText(view, 'Save')
    await waitForPage(view, 'admin-agent-detail')
    await waitForText(view, 'MCP resource server and operation assignments saved.')
    await view.evaluate(`document.querySelector('main a[href$="/resources"]').click()`)
    await waitForPage(view, 'admin-agent-resources')
    await waitForText(view, connection)
    await waitForText(view, 'Read documents')
    expect(await view.evaluate(`document.querySelector('form')===null`)).toBe(true)
    await view.navigate(
      `${uiOrigin}/realms/default/admin/mcp-resource-servers/${encodeURIComponent(serverID)}/access`,
    )
    await waitForPage(view, 'admin-mcp-agent-access')
    await waitForText(view, agentName)
    await view.evaluate(`document.querySelector('tbody a').click()`)
    await waitForPage(view, 'admin-agent-resources-edit')
    expect(await view.evaluate(`document.querySelector('#resource-connection').value`)).toBe(
      serverID,
    )
    await view.evaluate(
      `[...document.querySelectorAll('label')].find(node=>node.textContent.includes('Read documents'))?.querySelector('input[type=checkbox]')?.click()`,
    )
    await clickButtonByText(view, 'Save')
    await waitForPage(view, 'admin-agent-detail')
    expect(String(await view.evaluate('window.location.pathname'))).toBe(detailPath)
    await view.evaluate(`document.querySelector('main a[href$="/resources"]').click()`)
    await waitForPage(view, 'admin-agent-resources')
    await waitForText(view, 'No operations have been assigned yet.')
  } finally {
    view.close()
  }
}, 90_000)

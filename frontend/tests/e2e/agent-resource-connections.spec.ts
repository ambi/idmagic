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

//spec:covers REQ-AUTHORIZATION-017, REQ-OAUTH2-064, REQ-IDMANAGEMENT-106: モデル画面を使わず、付与と取消を同じ保存へ反映し、Agent 詳細から権限の材料を再取得する。
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
    await clickLinkByText(view, 'Access materials')
    await waitForPage(view, 'admin-agent-access')
    await waitForText(view, 'does not guarantee permission to execute')
    await waitForText(view, 'No delegation grants.')
    const assignedRow = await view.evaluate(
      `[...document.querySelectorAll('tbody tr')].find(row=>row.textContent.includes(${JSON.stringify(connection)}))?.textContent`,
    )
    expect(String(assignedRow)).toContain('Read documents')
    expect(String(assignedRow)).toContain('Direct operation assignment present')
    expect(String(assignedRow)).toContain('documents:read')
    expect(await view.evaluate(`document.querySelector('form')===null`)).toBe(true)
    await clickLinkByText(view, 'Agent details')
    await waitForPage(view, 'admin-agent-detail')
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
    await view.navigate(`${uiOrigin}${detailPath}/access`)
    await waitForPage(view, 'admin-agent-access')
    const removedRow = await view.evaluate(
      `[...document.querySelectorAll('tbody tr')].find(row=>row.textContent.includes(${JSON.stringify(connection)}))?.textContent`,
    )
    expect(String(removedRow)).toContain('No direct operation assignment')
    await view.navigate(
      `${uiOrigin}/realms/default/admin/mcp-resource-servers/${encodeURIComponent(serverID)}/edit`,
    )
    await waitForPage(view, 'admin-mcp-resource-server-edit')
    await setInputValue(view, '#tool-label-0', 'Read reviewed documents')
    await clickButtonByText(view, 'Update')
    await waitForPage(view, 'admin-mcp-resource-servers')
    await view.navigate(`${uiOrigin}${detailPath}/access`)
    await waitForPage(view, 'admin-agent-access')
    await waitForText(view, 'Read reviewed documents')
  } finally {
    view.close()
  }
}, 90_000)

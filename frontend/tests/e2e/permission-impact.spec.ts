import { expect, test } from 'bun:test'
import {
  authorizePath,
  clickElementByAriaLabel,
  clickSelector,
  clickSummaryByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  setSelectValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

//spec:covers REQ-OAUTH2-097, REQ-OAUTH2-098: 管理画面で保存した申告を再取得し、本人が許可する前に登録値と未登録の注意を読める。
test('permission impact is saved and shown before consent', async () => {
  const view = await openWebView({ width: 1440, height: 1800 })
  try {
    const suffix = String(Date.now())
    const name = `Impact-${suffix}`
    await navigateAndLogin(
      view,
      '/realms/default/admin/mcp-resource-servers/new',
      'admin-mcp-resource-server-create',
    )
    await setInputValue(view, '#resource', `https://impact-${suffix}.example/mcp`)
    await setInputValue(view, '#name', name)
    await setSelectValue(view, '#evaluator-client', '')
    await clickSummaryByText(view, 'The MCP server checks tokens itself')
    await setInputValue(view, '#scopes', 'openid')
    await setSelectValue(view, '#scope-impact-0-changes', 'reversible')
    await setSelectValue(view, '#scope-impact-0-personal-data', 'false')
    await clickSelector(view, 'form button[type="submit"]')
    await waitForPage(view, 'admin-mcp-resource-servers')
    await waitForText(view, name)
    await clickElementByAriaLabel(view, `Show details for ${name}`)
    await clickSelector(view, 'main a[href$="/edit"]')
    await waitForPage(view, 'admin-mcp-resource-server-edit')
    expect(await view.evaluate("document.querySelector('#scope-impact-0-changes').value")).toBe(
      'reversible',
    )
    expect(
      await view.evaluate("document.querySelector('#scope-impact-0-personal-data').value"),
    ).toBe('false')
    await view.navigate(`${uiOrigin}${authorizePath('impact-consent')}&prompt=consent`)
    await waitForPage(view, 'consent')
    await waitForText(view, 'Reversible changes')
    await waitForText(view, 'Does not handle personal data')
    await waitForText(view, 'Impact not registered')
    await waitForText(view, 'Review as an irreversible change.')
    await waitForText(view, 'These are claims made by the registrant.')
    expect(
      await view.evaluate('document.querySelector(\'meta[name="idmagic:page"]\').content'),
    ).toBe('consent')
  } finally {
    view.close()
  }
}, 90_000)

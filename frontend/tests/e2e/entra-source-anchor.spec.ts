import { expect, test } from 'bun:test'
import {
  openWebView,
  navigateAndLogin,
  clickButtonByText,
  setInputValue,
  waitForText,
} from './fixtures'

//spec:covers REQ-SYSTEM-011: 正式な Entra の追加画面から送った構成の拒否を、日本語の辞書で表示すること。
//spec:covers REQ-WSFEDERATION-001: 定義のない sourceAnchor 属性の構成を拒否し、属性の設定を直す案内を表示すること。
test('Entra configuration displays a localized sourceAnchor refusal', async () => {
  const view = await openWebView({ width: 1280, height: 1200 })
  try {
    await navigateAndLogin(view, '/admin/federation/entra/new', 'admin-entra-federation-add')
    await clickButtonByText(view, '日本語')
    await setInputValue(view, '#entra_domain', 'contoso.com')
    await setInputValue(view, '#entra_source_anchor', 'undefined_source_anchor')
    await clickButtonByText(view, '保存')
    await waitForText(view, 'sourceAnchor 属性を一意の属性に設定してください。')
    const alert = await view.evaluate(`document.querySelector('[role="alert"]')?.textContent ?? ''`)
    expect(alert).toBe('sourceAnchor 属性を一意の属性に設定してください。')
  } finally {
    view.close()
  }
}, 60_000)

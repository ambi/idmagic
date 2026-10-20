import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  navigateAndLogin,
  openWebView,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

async function fill(view: Bun.WebView, label: string, value: string) {
  const changed = await view.evaluate(`(() => {
    const label = [...document.querySelectorAll('label')].find(el => el.textContent?.startsWith(${JSON.stringify(label)}))
    const input = label?.querySelector('input')
    if (!(input instanceof HTMLInputElement)) return false
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(input, ${JSON.stringify(value)})
    input.dispatchEvent(new Event('input', { bubbles: true }))
    input.dispatchEvent(new Event('change', { bubbles: true }))
    return true
  })()`)
  expect(changed).toBe(true)
}

//spec:covers REQ-AUTHENTICATION-041: 正式なフォームから候補取得へ到達し、拒否されても手動設定を保存できる。
// 候補の成功と適用は UI 単体と HTTP 契約で検査する。ブラウザーでは実際のサーバーの取得先検査を通す。
test('metadata input assistance rejects a local issuer and keeps manual configuration usable', async () => {
  const view = await openWebView({ width: 1280, height: 1800 })
  try {
    await navigateAndLogin(view, '/admin/identity-providers/new', 'admin-identity-provider-create')
    await clickButtonByText(view, 'English')
    await fill(view, 'Display name', 'Metadata preview E2E')
    await fill(view, 'Issuer URL', 'https://127.0.0.1')
    await clickButtonByText(view, 'Fetch metadata')
    await waitForText(view, 'Could not fetch metadata candidates. Check the issuer.')
    expect(await view.evaluate('window.location.pathname')).toContain('/identity-providers/new')
    await fill(view, 'Issuer URL', 'https://manual.example')
    await fill(view, 'Client ID', 'manual-client')
    await fill(view, 'Authorization endpoint', 'https://manual.example/auth')
    await fill(view, 'Token endpoint', 'https://manual.example/token')
    await fill(view, 'JWKS URI', 'https://manual.example/jwks')
    await clickButtonByText(view, 'Create connection')
    await waitForPage(view, 'admin-identity-provider-detail')
    await waitForText(view, 'https://manual.example/auth')
    await view.navigate(`${uiOrigin}/admin/identity-providers`)
    await waitForPage(view, 'admin-identity-providers')
    await waitForText(view, 'Metadata preview E2E')
  } finally {
    view.close()
  }
}, 90_000)

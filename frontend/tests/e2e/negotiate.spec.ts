import { expect, test } from 'bun:test'
import {
  authorizePath,
  clickButtonByAnyText,
  loginFromCurrentPage,
  openWebView,
  uiOrigin,
  waitForPage,
  waitForUrl,
} from './fixtures'

//spec:covers REQ-AUTHENTICATION-043: AD に参加していないブラウザーと未設定の配備では、チャレンジを受けず既存の認可を完了できる。
test('a browser without Windows authentication completes normal sign-in', async () => {
  const view = await openWebView({ width: 1280, height: 1600 })
  try {
    await view.navigate(`${uiOrigin}${authorizePath('normal-without-ad')}`)
    await waitForPage(view, 'login')
    const availability = await view.evaluate<{
      enabled: boolean
      automatic: boolean
      challenge: string | null
    }>(`(async () => {
      const response = await fetch('/realms/default/api/auth/negotiate');
      return { ...(await response.json()), challenge: response.headers.get('WWW-Authenticate') };
    })()`)
    expect(availability).toEqual({ enabled: false, automatic: false, challenge: null })
    expect(
      await view.evaluate<boolean>(`!!document.querySelector('input[name="username"]:enabled')`),
    ).toBe(true)
    await loginFromCurrentPage(view)
    await waitForPage(view, 'consent')
    await clickButtonByAnyText(view, ['許可', 'Allow'])
    await waitForUrl(view, /localhost:3000\/callback/)
    const callback = new URL(view.url)
    expect(callback.searchParams.get('code')).toBeTruthy()
    expect(callback.searchParams.get('state')).toBe('normal-without-ad')
  } finally {
    view.close()
  }
}, 60_000)

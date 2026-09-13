import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { LocaleProvider } from '../../lib/i18n'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { demoLoginEnabled } from './demoLogin'
import { HomePage } from './HomePage'
import { informationalPagesDictionary } from './InformationalPages.i18n'

describe('DemoLoginAffordance', () => {
  afterEach(() => {
    restoreGlobals()
  })

  // 出す条件と出さない条件を対で見る。出る側だけを見る検査は、常に出す実装を通す。
  // その実装では、配備したフロントエンドがデモ利用者の資格情報を画面に書いたままになる。
  //
  //spec:covers EX-SYSTEM-006-01: ビルド済みのフロントエンドで VITE_DEMO_LOGIN_ENABLED=true のとき DemoLoginAffordance を表示すること。
  //spec:covers EX-SYSTEM-006-02: VITE_DEMO_LOGIN_ENABLED が未設定または true 以外のとき DemoLoginAffordance を表示しないこと。
  //spec:covers EX-SYSTEM-007-01: Vite 開発サーバーでの実行時は VITE_DEMO_LOGIN_ENABLED の設定にかかわらず DemoLoginAffordance を表示すること。
  it('follows the startup configuration, and the Vite dev server needs none', () => {
    // ビルド済みの配備 (DEV は false)。
    expect(demoLoginEnabled({ DEV: false, VITE_DEMO_LOGIN_ENABLED: 'true' })).toBe(true)
    expect(demoLoginEnabled({ DEV: false })).toBe(false)
    expect(demoLoginEnabled({ DEV: false, VITE_DEMO_LOGIN_ENABLED: 'false' })).toBe(false)
    expect(demoLoginEnabled({ DEV: false, VITE_DEMO_LOGIN_ENABLED: '1' })).toBe(false)

    // Vite 開発サーバー。設定が無くても、`true` 以外であっても表示する。
    expect(demoLoginEnabled({ DEV: true })).toBe(true)
    expect(demoLoginEnabled({ DEV: true, VITE_DEMO_LOGIN_ENABLED: 'false' })).toBe(true)
  })

  //spec:covers EX-SYSTEM-006-01: 有効にした HomePage が DemoLoginAffordance を表示し、選択すると code + PKCE の認可リクエストへ進むこと。
  //spec:covers EX-SYSTEM-006-02: 無効にした HomePage が DemoLoginAffordance を表示せず、代わりにアプリケーションから開始する案内を出すこと。
  it('starts an authorization_code request when selected and is absent when disabled', async () => {
    const assign = mock()
    stubGlobal('location', {
      ...window.location,
      origin: 'http://localhost:5173',
      pathname: '/',
      search: '',
      assign,
    })

    const { unmount } = render(
      <LocaleProvider initialLocale="en">
        <HomePage demoEnabled />
      </LocaleProvider>,
    )
    const affordance = await screen.findByRole('button', {
      name: new RegExp(informationalPagesDictionary.en.startDemo, 'i'),
    })
    fireEvent.click(affordance)
    await waitFor(() => expect(assign).toHaveBeenCalled())
    const target = new URL(String(assign.mock.calls[0]?.[0]), 'http://localhost:5173')
    expect(target.pathname.endsWith('/authorize')).toBe(true)
    expect(target.searchParams.get('response_type')).toBe('code')
    expect(target.searchParams.get('code_challenge_method')).toBe('S256')
    expect(target.searchParams.get('code_challenge')).toBeTruthy()
    expect(target.searchParams.get('redirect_uri')?.endsWith('/callback')).toBe(true)
    unmount()

    render(
      <LocaleProvider initialLocale="en">
        <HomePage demoEnabled={false} />
      </LocaleProvider>,
    )
    expect(
      screen.queryByRole('button', {
        name: new RegExp(informationalPagesDictionary.en.startDemo, 'i'),
      }),
    ).toBeNull()
    expect(
      screen.getByText(informationalPagesDictionary.en.startFromApplication),
    ).toBeInTheDocument()
  })
})

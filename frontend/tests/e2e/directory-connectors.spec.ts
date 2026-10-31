import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  clickLinkByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  waitForPage,
  waitForText,
} from './fixtures'

const shownToken = (view: Bun.WebView) =>
  view.evaluate(`document.querySelector('code')?.textContent ?? ''`) as Promise<string>

const bodyText = (view: Bun.WebView) => view.evaluate(`document.body.innerText`) as Promise<string>

//spec:covers REQ-SOURCING-023: 正式な画面が作成と再発行の登録トークンを一度だけ表示し、再読み込みと URL と保存領域に残さず、無効化で操作を消すこと。
test('Directory connector creation shows the enrollment token once', async () => {
  const view = await openWebView({ width: 1280, height: 1200 })
  const displayName = `e2e-connector-${Date.now()}`
  try {
    await navigateAndLogin(
      view,
      '/admin/directory-connectors/new',
      'admin-directory-connector-create',
    )
    await clickButtonByText(view, '日本語')
    await setInputValue(view, '#display_name', displayName)
    await clickButtonByText(view, '作成')
    await waitForText(view, 'この登録トークンは再表示できません')

    const created = await shownToken(view)
    expect(created).toStartWith('idmagic_dce_')
    const leaked = await view.evaluate(`JSON.stringify([
      location.href,
      JSON.stringify(localStorage),
      JSON.stringify(sessionStorage),
    ])`)
    expect(leaked).not.toContain(created)

    await clickLinkByText(view, 'Connector の詳細へ')
    await waitForPage(view, 'admin-directory-connector-detail')
    await waitForText(view, displayName)
    expect(await bodyText(view)).not.toContain(created)
    const detailURL = (await view.evaluate('location.href')) as string

    await clickButtonByText(view, '登録トークンを再発行')
    await waitForText(view, 'この登録トークンは再表示できません')
    const reissued = await shownToken(view)
    expect(reissued).toStartWith('idmagic_dce_')
    expect(reissued).not.toBe(created)

    await view.navigate(detailURL)
    await waitForPage(view, 'admin-directory-connector-detail')
    await waitForText(view, displayName)
    expect(await bodyText(view)).not.toContain(reissued)

    await clickButtonByText(view, '無効化')
    await waitForText(view, '無効化は取り消せません')
    await clickButtonByText(view, '無効化する')
    await waitForText(view, '無効化時刻')
    const disabled = await bodyText(view)
    expect(disabled).not.toContain('登録トークンを再発行')

    await view.navigate(detailURL.replace(/\/[^/]+$/, ''))
    await waitForPage(view, 'admin-directory-connectors')
    await waitForText(view, displayName)
  } finally {
    view.close()
  }
}, 90_000)

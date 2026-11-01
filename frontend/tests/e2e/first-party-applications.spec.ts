import { expect, test } from 'bun:test'
import { navigateAndLogin, openWebView, uiOrigin, waitForPage, waitForText } from './fixtures'

//spec:covers REQ-APPLICATION-017: seed から HTTP の射影を経てブラウザーへ届く保護状態を確認する。
test('built-in applications hide deletion while ordinary applications allow it', async () => {
  const view = await openWebView({ width: 1440, height: 2000 })
  try {
    await navigateAndLogin(view, '/admin/applications', 'admin-applications')
    for (const [id, name, protectedApp] of [
      ['00000000-0000-4000-8000-000000000101', 'IdMagic Admin Console', true],
      ['00000000-0000-4000-8000-000000000102', 'IdMagic Account Portal', true],
      ['00000000-0000-4000-8000-000000000103', 'Demo Client', false],
    ] as const) {
      await view.navigate(`${uiOrigin}/admin/applications/${id}`)
      await waitForPage(view, 'admin-application-detail')
      await waitForText(view, name)
      const deleteShown = await view.evaluate(`(() => [...document.querySelectorAll('button')]
        .some((button) => button.textContent?.trim() === 'Delete'))()`)
      expect(deleteShown).toBe(!protectedApp)
    }
  } finally {
    view.close()
  }
}, 60_000)

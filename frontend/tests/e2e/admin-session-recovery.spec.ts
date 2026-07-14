// wi-116: dev サーバ再起動後の管理コンソール復旧導線を E2E で固定する。
// memory persistence で dev サーバを再起動すると、ブラウザには stale な access token
// (sessionStorage) が残る一方、サーバ側の署名鍵/セッションは失われる。管理画面をリロード
// すると保持トークンでの /api/auth/account が 401 になるが、行き止まりの
// 「認証を続行できません」ではなく、保持トークンを破棄して再認可し、元の画面
// (/admin/users) へ戻れることを検証する。
//
// サーバ再起動そのものは模さず、ブラウザ側に「もう検証できないトークン」を注入して
// 401 を再現する (署名鍵が回転して過去トークンが無効化された状態と等価)。
import { afterAll, beforeAll, expect, test } from 'bun:test'
import {
  loginFromCurrentPage,
  metaPage,
  navigateAndLogin,
  startE2EEnvironment,
  stopE2EEnvironment,
  uiOrigin,
  waitForLocationPath,
  waitForPage,
} from './fixtures'

// waitForAnyPage は、複数のページ種別のいずれかに到達するまで待って、到達したものを返す。
async function waitForAnyPage(
  view: Bun.WebView,
  kinds: string[],
  timeoutMs = 30_000,
): Promise<string> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      const current = await metaPage(view)
      if (typeof current === 'string' && kinds.includes(current)) {
        return current
      }
    } catch {
      // 遷移中は evaluate が失敗しうる。リトライする。
    }
    await Bun.sleep(150)
  }
  throw new Error(`timeout waiting for any of ${kinds.join(', ')}, last url=${view.url}`)
}

beforeAll(async () => {
  await startE2EEnvironment()
}, 180_000)

afterAll(async () => {
  await stopE2EEnvironment()
}, 30_000)

test('admin console recovers from a stale token and returns to the original page', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    // 管理コンソールの内側の画面 (/admin/users) に OIDC RP としてログインして到達する。
    await navigateAndLogin(view, '/admin/users', 'admin-users')

    // dev サーバ再起動後の状態を模す: 期限は未来だがサーバでは検証できない
    // access token を sessionStorage に残す (refresh token は持たせない)。
    await view.evaluate(
      `sessionStorage.setItem('ra_oidc_token_admin', JSON.stringify({ accessToken: 'stale.access.token', expiresAt: Date.now() + 600000 }))`,
    )

    // 直前に開いていた管理画面をリロードする。stale トークンでの /api/auth/account は
    // 401 になる。行き止まりの 'error' 画面ではなく復旧すること (= 再認可) を検証する。
    // サーバ側 login session が生きている本ハーネスでは再認可は silent に進むが、
    // dev サーバ再起動時のように session も失われていれば 'login' 画面が挟まる。
    // どちらの経路でも保持トークンを破棄して元画面へ戻れることが本 WI の要件。
    await view.navigate(`${uiOrigin}/admin/users`)
    const reached = await waitForAnyPage(view, ['login', 'admin-users'])
    if (reached === 'login') {
      await loginFromCurrentPage(view)
    }

    // 元の画面パス (/admin/users) へ戻る。
    await waitForLocationPath(view, '/admin/users')
    await waitForPage(view, 'admin-users')

    // 保持していた stale トークンは破棄され、新しいトークンで再認可されている。
    const stored = (await view.evaluate(`sessionStorage.getItem('ra_oidc_token_admin')`)) as
      | string
      | null
    expect(stored).not.toBeNull()
    expect(stored ?? '').not.toContain('stale.access.token')
  } finally {
    view.close()
  }
}, 90_000)

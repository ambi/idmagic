import { afterAll, beforeAll, test } from 'bun:test'

import {
  clickButtonByText,
  loginFromCurrentPage,
  startE2EEnvironment,
  stopE2EEnvironment,
  uiOrigin,
  waitForAnyText,
  waitForPage,
} from './fixtures'

beforeAll(async () => {
  await startE2EEnvironment()
}, 180_000)

afterAll(async () => {
  await stopE2EEnvironment()
}, 30_000)

// REQ-SYSTEM-010: 一つの言語選択が認証、アカウント、管理の実画面へ継続して反映される。
test('selected locale renders authentication account and admin surfaces', async () => {
  const view = new Bun.WebView({ width: 1280, height: 1600 })
  try {
    await view.navigate(`${uiOrigin}/account`)
    await waitForPage(view, 'login')
    await clickButtonByText(view, '日本語')
    await waitForAnyText(view, ['ログイン'])

    await loginFromCurrentPage(view)
    await waitForPage(view, 'account-home', 30_000)
    await waitForAnyText(view, ['ユーザー名'])

    await view.navigate(`${uiOrigin}/admin`)
    await waitForPage(view, 'admin-dashboard', 30_000)
    await waitForAnyText(view, ['ダッシュボード'])
  } finally {
    view.close()
  }
}, 90_000)

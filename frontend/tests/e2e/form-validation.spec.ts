// wi-55197: Vite が束ねた本番と同じ経路の画面で、共通のフォームの検証（Zod の設定と
// FieldError）が動くことを確かめる。規則の両側と境界値は src/lib/form.test.ts と各画面の
// 単体テストが固定するので、ここではブラウザーとの統合だけを見る。

// 主要ユースケース追跡: WCAG22-LABELS-ERRORS。
import { expect, test } from 'bun:test'
import { navigateAndLogin, openWebView, POLL_INTERVAL_MS, setInputValue } from './fixtures'

type FieldState = { invalid: string | null; description: string; createDisabled: boolean }

async function usernameState(view: Bun.WebView): Promise<FieldState> {
  const state = await view.evaluate(`(() => {
    const input = document.querySelector('#preferred_username')
    const describedBy = input?.getAttribute('aria-describedby') ?? ''
    const description = describedBy
      .split(' ')
      .map((id) => document.getElementById(id)?.textContent ?? '')
      .join(' ')
      .trim()
    const create = document.querySelector('button[type="submit"]')
    return {
      invalid: input?.getAttribute('aria-invalid') ?? null,
      description,
      createDisabled: create instanceof HTMLButtonElement ? create.disabled : false,
    }
  })()`)
  return state as FieldState
}

async function waitForUsernameState(
  view: Bun.WebView,
  accept: (state: FieldState) => boolean,
  timeoutMs = 10_000,
): Promise<FieldState> {
  const deadline = Date.now() + timeoutMs
  let last = await usernameState(view)
  while (Date.now() < deadline) {
    last = await usernameState(view)
    if (accept(last)) return last
    await Bun.sleep(POLL_INTERVAL_MS)
  }
  throw new Error(`timeout waiting for username state, last=${JSON.stringify(last)}`)
}

test('user creation marks a blank username invalid in the browser', async () => {
  const view = await openWebView({ width: 1280, height: 1600 })
  try {
    await navigateAndLogin(view, '/admin/users/new', 'admin-user-create')
    await setInputValue(view, '#password', 'correct horse battery staple')
    await setInputValue(view, '#preferred_username', '   ')

    const blank = await waitForUsernameState(view, (state) => state.invalid === 'true')
    expect(blank.description).not.toBe('')
    expect(blank.createDisabled).toBe(true)

    await setInputValue(view, '#preferred_username', 'form-validation-e2e')
    const filled = await waitForUsernameState(view, (state) => state.invalid === null)
    expect(filled.description).toBe('')
    expect(filled.createDisabled).toBe(false)
  } finally {
    view.close()
  }
})

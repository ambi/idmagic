// WebAuthn の登録とサインインの二段目で、IdMagic の画面がサーバーへ送る本文を、本物の Chrome と
// CDP の仮想認証器で記録する。`mise run record-webauthn` から使い、記録は Go のテストだけが読む。
//
// E2E と同じスタック（Go の API、Vite、初期データ）を起動し、投入済みの利用者で次を行う。
// 1. パスワードでログインし、アカウントのセキュリティの画面で「パスキーを登録」を押す。
// 2. Cookie を消し、パスワードでログインし直し、二段目で「パスキーで認証」を押す。
// 画面の fetch を CDP で注入したスクリプトで包み、`/webauthn` を含む要求と応答の本文を記録する。
// 記録の値は challenge、origin、RP ID、認証器の鍵に縛られるので、書き換えずに保存する。
import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import {
  clickButtonByAnyText,
  demo,
  detectWebViewSupport,
  hasText,
  loginFromCurrentPage,
  metaPage,
  navigateAndLogin,
  openWebView,
  startE2EEnvironment,
  stopE2EEnvironment,
  uiOrigin,
} from '../e2e/fixtures'

const output = resolve(
  import.meta.dir,
  '../../../backend/authentication/internal/webauthn/usecases/testdata/chrome-virtual-authenticator.json',
)

type Captured = {
  url: string
  method: string
  requestBody: string | null
  status: number
  responseBody: string
}

// 画面の fetch を包む。ページの遷移をまたいで残すため、記録は localStorage に置く。
const captureScript = `(() => {
  const key = '__idmagicWebAuthnRecords'
  const original = window.fetch.bind(window)
  window.fetch = async (input, init) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    const response = await original(input, init)
    if (url.includes('/webauthn')) {
      const responseBody = await response.clone().text()
      const records = JSON.parse(localStorage.getItem(key) || '[]')
      records.push({
        url: new URL(url, location.href).pathname,
        method: (init && init.method) || 'GET',
        requestBody: init && typeof init.body === 'string' ? init.body : null,
        status: response.status,
        responseBody,
      })
      localStorage.setItem(key, JSON.stringify(records))
    }
    return response
  }
})()`

async function readRecords(view: Bun.WebView): Promise<Captured[]> {
  const raw = (await view.evaluate(
    `localStorage.getItem('__idmagicWebAuthnRecords') || '[]'`,
  )) as string
  return JSON.parse(raw) as Captured[]
}

async function waitForRecord(
  view: Bun.WebView,
  path: string,
  timeoutMs = 30_000,
): Promise<Captured> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const found = (await readRecords(view)).find((r) => r.url.endsWith(path))
    if (found) return found
    await Bun.sleep(200)
  }
  throw new Error(`timeout waiting for a request to ${path}`)
}

function decode(body: string | null): unknown {
  if (body === null || body === '') return null
  try {
    return JSON.parse(body)
  } catch {
    return body
  }
}

const authenticatorOptions = {
  protocol: 'ctap2',
  transport: 'internal',
  hasResidentKey: true,
  hasUserVerification: true,
  isUserVerified: true,
  automaticPresenceSimulation: true,
}

async function main(): Promise<void> {
  await detectWebViewSupport()
  await startE2EEnvironment()
  try {
    const view = await openWebView({ width: 1280, height: 900 })
    // CDP のセッションは最初の遷移で開く。
    await view.navigate('about:blank')
    const browser = (await view.cdp('Browser.getVersion')) as { product: string; userAgent: string }
    await view.cdp('Page.addScriptToEvaluateOnNewDocument', { source: captureScript })
    await view.cdp('WebAuthn.enable', { enableUI: false })
    await view.cdp('WebAuthn.addVirtualAuthenticator', { options: authenticatorOptions })

    // システムコンソールは MFA で成立した認証を求める（REQ-SYSTEM-033）ので、システム管理者で登録し、
    // システムコンソールを開いてサインインの二段目でパスキーを使う。
    await navigateAndLogin(view, '/account/security', 'account-security', demo.systemAdminUsername)
    await clickButtonByAnyText(view, ['パスキーを登録', 'Register passkey']).catch(
      async (error) => {
        const text = await view.evaluate(`document.body.innerText.slice(0, 1500)`)
        throw new Error(`${error}\n--- page ---\n${text}`)
      },
    )
    const registration = await waitForRecord(view, '/mfa/webauthn/register/finish')
    if (registration.status !== 200 && registration.status !== 204) {
      throw new Error(
        `registration finished with ${registration.status}: ${registration.responseBody}`,
      )
    }

    await view.navigate(`${uiOrigin}/system/tenants`)
    const deadline = Date.now() + 60_000
    let usedPasskey = false
    while (Date.now() < deadline) {
      const page = await metaPage(view).catch(() => null)
      if (page === 'system-tenants') break
      if (page === 'login') {
        await loginFromCurrentPage(view, demo.systemAdminUsername)
        await Bun.sleep(500)
      } else if (page === 'totp' && !usedPasskey) {
        // 二段目の画面は既定で認証アプリの方法を開くことがあるので、先にパスキーの方法を選ぶ。
        if (
          !(await hasText(view, 'パスキーで認証')) &&
          !(await hasText(view, 'Authenticate with passkey'))
        ) {
          await clickButtonByAnyText(view, ['パスキー', 'Passkey'])
        }
        await clickButtonByAnyText(view, ['パスキーで認証', 'Authenticate with passkey'])
        usedPasskey = true
      } else {
        await Bun.sleep(200)
      }
    }
    if (!usedPasskey) {
      const text = await view.evaluate(`document.body.innerText.slice(0, 800)`)
      throw new Error(`the system console did not ask for the second factor\n${text}`)
    }
    const assertion = await waitForRecord(view, '/api/auth/webauthn')
    if (assertion.status >= 400) {
      throw new Error(`assertion finished with ${assertion.status}: ${assertion.responseBody}`)
    }

    const records = await readRecords(view)
    const exchanges = records.map((r, i) => ({
      name: `${String(i + 1).padStart(2, '0')}-${r.url.split('/').filter(Boolean).slice(-2).join('-')}`,
      source: `IdMagic の画面が ${r.method} ${r.url} へ送った要求と、サーバーの応答`,
      request: { method: r.method, path: r.url, headers: {}, body: decode(r.requestBody) },
      response: { status: r.status, headers: {}, body: decode(r.responseBody) },
    }))
    const recording = {
      counterpart: `${browser.product} の CDP の仮想認証器（${JSON.stringify(authenticatorOptions)}）`,
      counterpart_version: `${browser.product}（${browser.userAgent}）`,
      retrieved_at: new Date().toISOString().slice(0, 10),
      method:
        'mise run record-webauthn が E2E と同じスタック（WEBAUTHN_RP_ID=localhost、WEBAUTHN_RP_ORIGINS=' +
        uiOrigin +
        '）を起動し、Bun.WebView の Chrome バックエンドに CDP の WebAuthn.addVirtualAuthenticator で認証器を足した。' +
        `投入済みの利用者 ${demo.username} でパスワードのログインの後、アカウントのセキュリティの画面で「パスキーを登録」を押し、` +
        'Cookie を消してパスワードでログインし直し、二段目で「パスキーで認証」を押した。' +
        '画面の fetch を Page.addScriptToEvaluateOnNewDocument で包み、/webauthn を含む要求と応答の本文をそのまま記録した。' +
        'CSRF のトークンとセッションの Cookie はヘッダーにあり、本文に含まないので記録しない。' +
        '値は challenge、origin、RP ID、仮想認証器の鍵に縛られるので書き換えていない。' +
        '仮想認証器の応答であり、実機の認証器（Touch ID、セキュリティキー）の attestation や拡張の癖は保証しない。',
      setup: { rp_id: 'localhost', rp_origin: uiOrigin, username: demo.username },
      exchanges,
    }
    mkdirSync(dirname(output), { recursive: true })
    writeFileSync(output, `${JSON.stringify(recording, null, 2)}\n`)
    console.log(`recorded ${exchanges.length} exchanges into ${output}`)
  } finally {
    await stopE2EEnvironment()
  }
}

await main()

// wi-75: 到達性スモークから一段進め、主要なブラウザ操作が API と接続されて
// ユーザー可視の成功状態へ到達することを検証する。
import { createHmac } from 'node:crypto'
import { afterAll, beforeAll, expect, test } from 'bun:test'
import {
  authorizePath,
  clickButtonByAnyText,
  clickButtonByText,
  clickElementByAriaLabel,
  clickLinkByText,
  demo,
  navigateAndLogin,
  selectDropdownOption,
  setCheckboxValue,
  setInputValue,
  setSelectValue,
  setSelectValueAt,
  startE2EEnvironment,
  stopE2EEnvironment,
  uiOrigin,
  waitForLocationHref,
  waitForPage,
  waitForUrl,
  waitForEmailURL,
  waitForText,
} from './fixtures'

function totpCode(secret: string, now = Date.now()): string {
  const counter = Math.floor(now / 1000 / 30)
  const key = decodeBase32(secret.replace(/\s+/g, ''))
  const message = Buffer.alloc(8)
  message.writeBigUInt64BE(BigInt(counter))
  const digest = createHmac('sha1', key).update(message).digest()
  const offset = digest[digest.length - 1] & 0x0f
  const binary =
    ((digest[offset] & 0x7f) << 24) |
    ((digest[offset + 1] & 0xff) << 16) |
    ((digest[offset + 2] & 0xff) << 8) |
    (digest[offset + 3] & 0xff)
  return String(binary % 1_000_000).padStart(6, '0')
}

function decodeBase32(value: string): Buffer {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
  let bits = ''
  for (const char of value.toUpperCase().replace(/=+$/, '')) {
    const index = alphabet.indexOf(char)
    if (index < 0) continue
    bits += index.toString(2).padStart(5, '0')
  }
  const bytes: number[] = []
  for (let i = 0; i + 8 <= bits.length; i += 8) {
    bytes.push(Number.parseInt(bits.slice(i, i + 8), 2))
  }
  return Buffer.from(bytes)
}

beforeAll(async () => {
  await startE2EEnvironment()
}, 180_000)

afterAll(() => {
  stopE2EEnvironment()
})

test('account profile can be updated from the browser', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    await navigateAndLogin(view, '/account/profile', 'account-profile')

    const suffix = String(Date.now())
    const displayName = `Alice E2E ${suffix}`
    await clickLinkByText(view, 'Edit')
    await waitForPage(view, 'account-profile-edit')
    await setInputValue(view, '#name', displayName)
    await setInputValue(view, '#given-name', 'Alice')
    await setInputValue(view, '#family-name', `Scenario ${suffix}`)
    await clickButtonByText(view, 'Save')

    await waitForText(view, 'Your profile has been updated.')
    await waitForText(view, displayName)
  } finally {
    view.close()
  }
}, 60_000)

test('account data export is triggered from the browser', async () => {
  const view = new Bun.WebView({ width: 1280, height: 1600 })
  try {
    await navigateAndLogin(view, '/account/data', 'account-data')
    await view.evaluate(`(() => {
      window.__raDownloadClicked = false
      const original = HTMLAnchorElement.prototype.click
      HTMLAnchorElement.prototype.click = function () {
        window.__raDownloadClicked = true
        return original.call(this)
      }
    })()`)

    await clickButtonByText(view, 'Download data (JSON)')

    const deadline = Date.now() + 10_000
    while (Date.now() < deadline) {
      if ((await view.evaluate('window.__raDownloadClicked === true')) === true) {
        return
      }
      await Bun.sleep(150)
    }
    throw new Error('timeout waiting for data export download trigger')
  } finally {
    view.close()
  }
}, 60_000)

test('admin general settings can be updated from the browser', async () => {
  const view = new Bun.WebView({ width: 1280, height: 1800 })
  try {
    await navigateAndLogin(view, '/admin/settings', 'admin-settings')

    const displayName = `Default organization ${Date.now()}`
    await clickButtonByText(view, 'Edit')
    await setInputValue(view, '#display-name', displayName)
    await clickButtonByText(view, 'Save')

    await waitForText(view, 'Updated the display name.')
    await waitForText(view, displayName)
  } finally {
    view.close()
  }
}, 60_000)

test('admin signing key rotation action is available to tenant admins', async () => {
  const view = new Bun.WebView({ width: 1280, height: 1800 })
  try {
    await navigateAndLogin(view, '/admin/keys', 'admin-keys')
    await waitForText(view, 'Signing keys')
    expect(
      await view.evaluate(`(() => [...document.querySelectorAll('button')]
        .some((button) => (button.textContent ?? '').includes('Rotate')))()`),
    ).toBe(true)
  } finally {
    view.close()
  }
}, 60_000)

test('account connected application consent can be revoked from the browser', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    // 先に account audience でログインして browser session を確立する。新規 WebView
    // から直接 /authorize を開くとログインコンテキストが作られる前に SPA route を読むため、
    // consent の受理経路を安定して観測できない。
    await navigateAndLogin(view, '/account/applications', 'account-applications')
    await view.navigate(`${uiOrigin}${authorizePath(`consent-revoke-${Date.now()}`)}`)
    const deadline = Date.now() + 15_000
    let needsConsent = false
    while (Date.now() < deadline) {
      if (view.url.includes('localhost:3000/callback')) break
      if (
        (await view.evaluate(`document.querySelector('meta[name="idmagic:page"]')?.content`)) ===
        'consent'
      ) {
        needsConsent = true
        break
      }
      await Bun.sleep(150)
    }
    if (needsConsent) await clickButtonByAnyText(view, ['許可して続行', 'Allow and continue'])
    await waitForUrl(view, /localhost:3000\/callback/)

    await view.navigate(`${uiOrigin}/account/applications`)
    await waitForPage(view, 'account-applications')
    // demo-client は client_name を持たないため、Application カタログ名 "Demo Client" へ
    // 解決される (wi-141)。UUID (ADR-084) は補助表記に留まる。
    await waitForText(view, 'Demo Client')
    await clickButtonByText(view, 'Revoke access')
    await waitForText(view, 'Access for “Demo Client” has been revoked.')
    await waitForText(view, 'No applications have been granted access.')
  } finally {
    view.close()
  }
}, 60_000)

test('account TOTP enrollment and removal step-up work from the browser', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2200 })
  try {
    await navigateAndLogin(view, '/account/security', 'account-security')

    await clickButtonByText(view, 'Set up authenticator app')
    await waitForText(view, 'Setup key')
    const secret = String(
      await view.evaluate('document.querySelector("#totp-secret")?.value ?? ""'),
    )
    expect(secret).not.toBe('')
    await setInputValue(view, '#totp-code', totpCode(secret))
    await clickButtonByText(view, 'Complete enrollment')
    await waitForText(view, 'Authenticator app enrolled.')

    await setInputValue(view, '#remove-code', totpCode(secret))
    await clickButtonByText(view, 'Remove authenticator app')

    const deadline = Date.now() + 10_000
    while (Date.now() < deadline) {
      if (
        await view.evaluate(
          `document.body.textContent?.includes('Re-authenticate to verify your identity') ?? false`,
        )
      ) {
        await setInputValue(view, '#step-up-credential', demo.password)
        await clickButtonByText(view, 'Re-authenticate and continue')
      }
      if (
        await view.evaluate(
          `document.body.textContent?.includes('Authenticator app removed.') ?? false`,
        )
      ) {
        return
      }
      await Bun.sleep(150)
    }
    throw new Error('timeout waiting for TOTP removal')
  } finally {
    view.close()
  }
}, 60_000)

test('account session list can revoke a different browser session', async () => {
  const first = new Bun.WebView({ width: 1280, height: 1800 })
  const second = new Bun.WebView({ width: 1280, height: 1200 })
  try {
    await navigateAndLogin(first, '/account', 'account-home')
    await navigateAndLogin(second, '/account', 'account-home')

    await first.navigate(`${uiOrigin}/account/activity`)
    await waitForPage(first, 'account-activity')
    await waitForText(first, 'End other sessions')
    const beforeCount = Number(
      await first.evaluate(`(() => [...document.querySelectorAll('button')]
        .filter((button) => (button.textContent ?? '').trim() === 'End').length)()`),
    )
    expect(beforeCount).toBeGreaterThan(0)
    const clicked = await first.evaluate(`(() => {
      const target = [...document.querySelectorAll('button')]
        .find((button) => (button.textContent ?? '').trim() === 'End')
      if (!target) return false
      target.click()
      return true
    })()`)
    expect(clicked).toBe(true)
    const deadline = Date.now() + 10_000
    while (Date.now() < deadline) {
      const afterCount = Number(
        await first.evaluate(`(() => [...document.querySelectorAll('button')]
          .filter((button) => (button.textContent ?? '').trim() === 'End').length)()`),
      )
      if (afterCount < beforeCount) return
      await Bun.sleep(150)
    }
    throw new Error('timeout waiting for revoked session row count to decrease')
  } finally {
    first.close()
    second.close()
  }
}, 60_000)

test('admin audit log can be filtered and export can be triggered', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    await navigateAndLogin(view, '/admin/audit_events', 'admin-audit-events')
    await view.evaluate(`(() => {
      window.__raAuditExportURL = ''
      window.open = (url) => {
        window.__raAuditExportURL = String(url ?? '')
        return null
      }
    })()`)

    // wi-147: イベントカテゴリとユーザー ID (操作者) は同じ検索条件一覧の行として操作する。
    // 既定行 (field=quick.category) の値 select は 2 番目の <select>。
    await setSelectValueAt(view, 'select', 1, 'authentication')
    await clickButtonByText(view, 'Add condition')
    // 追加した行 (既定 field=event.type) の種類 select は 3 番目の <select>。
    await setSelectValueAt(view, 'select', 2, 'quick.actor_id')
    await setInputValue(
      view,
      'input[placeholder="e.g., usr_... (the actor\'s user ID)"]',
      '00000000-0000-4000-8000-000000000001',
    )
    await clickButtonByText(view, 'Filter')
    await waitForText(view, 'UserAuthenticated')

    // wi-147: 検索実行後は URL が同期し、共有 URL / reload で同じ検索結果を復元できる。
    const searchURL = await waitForLocationHref(
      view,
      /category=authentication.*sub=00000000-0000-4000-8000-000000000001/,
    )

    await clickButtonByText(view, 'Export')
    const exportURL = await view.evaluate('window.__raAuditExportURL ?? ""')
    expect(String(exportURL)).toContain('/api/admin/audit_events/export')
    expect(String(exportURL)).toContain('category=authentication')
    expect(String(exportURL)).toContain('user_id=00000000-0000-4000-8000-000000000001')

    await view.navigate(searchURL)
    await waitForPage(view, 'admin-audit-events', 30_000)
    await waitForText(view, 'UserAuthenticated')
  } finally {
    view.close()
  }
}, 60_000)

test('admin user attribute schema can add and delete a custom attribute', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2200 })
  try {
    await navigateAndLogin(view, '/admin/tenant/attributes', 'admin-tenant-attributes')

    const key = `e2e_attr_${Date.now()}`
    await clickButtonByText(view, 'Add attribute')
    await setInputValue(view, '#attr-label', 'E2E attribute')
    await setInputValue(view, '#attr-key', key)
    await setSelectValue(view, '#attr-type', 'string')
    await setSelectValue(view, '#attr-visibility', 'self_readable')
    await setCheckboxValue(view, '#attr-editable', true)
    await clickButtonByText(view, 'Save')

    await waitForText(view, 'The attribute has been added.')
    await waitForText(view, key)

    await clickElementByAriaLabel(view, `Delete ${key}`)
    await waitForText(view, 'The attribute has been deleted.')
    await waitForText(view, 'There are no custom attributes yet.')
  } finally {
    view.close()
  }
}, 60_000)

test('account email change confirms through the local SMTP sink', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    await navigateAndLogin(view, '/account/emails', 'account-emails')

    const nextEmail = `alice.e2e.${Date.now()}@example.com`
    await clickButtonByText(view, 'Change')
    await setInputValue(view, '#new-email', nextEmail)
    await clickButtonByText(view, 'Send confirmation email')

    const deadline = Date.now() + 10_000
    while (Date.now() < deadline) {
      if (
        await view.evaluate(
          `document.body.textContent?.includes('Re-authenticate to verify your identity') ?? false`,
        )
      ) {
        await setInputValue(view, '#step-up-credential', demo.password)
        await clickButtonByText(view, 'Re-authenticate and continue')
        break
      }
      if (
        await view.evaluate(
          `document.body.textContent?.includes(${JSON.stringify(nextEmail)}) ?? false`,
        )
      ) {
        break
      }
      await Bun.sleep(150)
    }

    await waitForText(view, nextEmail)
    const verifyURL = await waitForEmailURL(nextEmail, '/account/email/verify')
    await view.navigate(verifyURL)
    await waitForPage(view, 'email-verify')
    await clickButtonByText(view, 'Confirm email address')
    await waitForText(view, 'Your email address has been confirmed.')
    demo.email = nextEmail
  } finally {
    view.close()
  }
}, 60_000)

test('password reset succeeds through the local SMTP sink without external mail', async () => {
  const view = new Bun.WebView({ width: 1280, height: 1800 })
  try {
    const suffix = Date.now()
    const username = `reset-e2e-${suffix}`
    const email = `reset.e2e.${suffix}@example.com`
    const initialPassword = `initial-password-${suffix}`
    const nextPassword = `reset-password-${suffix}`

    await navigateAndLogin(view, '/admin/users', 'admin-users')
    await clickLinkByText(view, 'Add user')
    await waitForPage(view, 'admin-user-create')
    await setInputValue(view, 'input[name="preferred_username"]', username)
    await setInputValue(view, 'input[name="name"]', 'Reset E2E')
    await setInputValue(view, 'input[name="email"]', email)
    await setInputValue(view, 'input[name="password"]', initialPassword)
    await setCheckboxValue(view, 'input[name="email_verified"]', true)
    await clickButtonByText(view, 'Create')
    await waitForPage(view, 'admin-user-detail')
    await waitForText(view, username)

    await view.navigate(`${uiOrigin}/forgot_password`)
    await waitForPage(view, 'forgot-password')
    await setInputValue(view, 'input[name="email"]', email)
    await clickButtonByText(view, 'Send reset link')
    await waitForText(view, 'If an account exists, we sent a password reset email.')

    const resetURL = await waitForEmailURL(email, '/reset_password')
    await view.navigate(resetURL)
    await waitForPage(view, 'reset-password')
    await setInputValue(view, 'input[name="new_password"]', nextPassword)
    await clickButtonByText(view, 'Update password')
    await waitForText(view, 'Your password was updated. You can sign in now.')
  } finally {
    view.close()
  }
}, 60_000)

test('admin application lifecycle and agent credential binding work from the browser', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2400 })
  try {
    const suffix = Date.now()
    const appName = `E2E OIDC App ${suffix}`
    const agentName = `e2e-agent-${suffix}`

    await navigateAndLogin(view, '/admin/applications', 'admin-applications')
    await clickButtonByText(view, 'Add application')
    await setInputValue(view, '#app-name', appName)
    await setInputValue(view, '#app-redirects', `https://client.example.test/callback/${suffix}`)
    await setInputValue(view, '#app-oidc-scope', 'openid profile email')
    await clickButtonByText(view, 'Create')
    await waitForText(view, 'The client has been created.')

    const clientID = String(
      await view.evaluate(`(() => {
        const values = [...document.querySelectorAll('code')]
          .map((node) => node.textContent?.trim() ?? '')
          .filter(Boolean)
        return values[0] ?? ''
      })()`),
    )
    expect(clientID).not.toBe('')

    await clickButtonByText(view, 'Stored')
    await waitForUrl(view, /\/admin\/applications\/[^/]+$/)
    const appDetailURL = view.url
    await waitForText(view, appName)
    await waitForText(view, clientID)
    await clickLinkByText(view, 'Edit')
    await waitForUrl(view, /\/admin\/applications\/[^/]+\/edit$/)
    await selectDropdownOption(view, 'Select…', demo.username)
    await clickButtonByText(view, 'Assign')
    await waitForText(view, demo.username)

    await view.navigate(`${uiOrigin}/admin/agents`)
    await waitForPage(view, 'admin-agents')
    await clickButtonByText(view, 'Register')
    await setInputValue(view, '#agent-name', agentName)
    await setInputValue(view, '#agent-description', 'E2E credential binding')
    await setSelectValue(view, '#agent-kind', 'supervised')
    await setInputValue(view, '#agent-roles', 'e2e:read, e2e:write')
    await view.click('form button[type="submit"]')
    await waitForText(view, 'The agent has been registered.')
    await waitForText(view, agentName)

    await setInputValue(view, 'input[aria-label="client_id to bind"]', clientID)
    await clickButtonByText(view, 'Bind')
    await waitForText(view, clientID)

    await clickButtonByText(view, 'Unbind')
    await waitForText(view, 'No credentials are bound.')

    await view.navigate(appDetailURL)
    await waitForText(view, appName)
    await clickButtonByText(view, 'Delete')
    await clickButtonByText(view, 'Confirm deletion')
    await waitForPage(view, 'admin-applications')
  } finally {
    view.close()
  }
}, 90_000)

test('admin user list opens a user detail page', async () => {
  const view = new Bun.WebView({ width: 1280, height: 1800 })
  try {
    await navigateAndLogin(view, '/admin/users', 'admin-users')
    // 一覧で先頭ユーザーが選択され、右ペインの「詳細」から専用詳細画面へ遷移する。
    await view.click('a[href^="/admin/users/"]')
    await waitForPage(view, 'admin-user-detail')
    await waitForUrl(view, /\/admin\/users\/[^/]+$/)
    await waitForText(view, 'User ID')
  } finally {
    view.close()
  }
}, 60_000)

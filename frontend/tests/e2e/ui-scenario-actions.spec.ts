// wi-75: 到達性スモークから一段進め、主要なブラウザ操作が API と接続されて
// ユーザー可視の成功状態へ到達することを検証する。
import { createHmac } from 'node:crypto'

// 主要ユースケース追跡: REQ-AUTHENTICATION-013。
import { afterAll, beforeAll, expect, test } from 'bun:test'
import {
  authorizePath,
  clickButtonByAnyText,
  clickEnabledButtonByText,
  clickButtonByText,
  clickElementByAriaLabel,
  clickLinkByText,
  clickMenuItemByText,
  clickSummaryByText,
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
  waitForInputValue,
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

async function waitForPaginationPage(
  view: Bun.WebView,
  expected: number,
  timeoutMs = 15_000,
): Promise<number> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const position = String(
      await view.evaluate(`(() => [...document.querySelectorAll('span')]
        .map((element) => (element.textContent ?? '').trim())
        .find((text) => /^\\d+ \\/ \\d+$/.test(text)) ?? '')()`),
    )
    const match = position.match(/^(\d+) \/ (\d+)$/)
    if (match && Number(match[1]) === expected) return Number(match[2])
    await Bun.sleep(150)
  }
  throw new Error(`timeout waiting for pagination page ${expected}`)
}

beforeAll(async () => {
  await startE2EEnvironment()
}, 180_000)

afterAll(async () => {
  await stopE2EEnvironment()
}, 30_000)

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

test('admin can create a shared SAML identity provider profile', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2200 })
  try {
    await navigateAndLogin(
      view,
      '/admin/settings/saml-idp-profiles/new',
      'admin-saml-idp-profile-create',
    )

    const profileName = `SAML partner ${Date.now()}`
    await setInputValue(view, '#saml-profile-name', profileName)
    await clickEnabledButtonByText(view, 'Create')

    await waitForInputValue(view, profileName)
    await waitForText(view, 'Shared (multiple SPs)')
  } finally {
    view.close()
  }
}, 60_000)

test('admin API access token lifecycle works with selected SCIM scopes', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2200 })
  try {
    await navigateAndLogin(view, '/admin/settings', 'admin-settings')
    await clickButtonByText(view, 'API access tokens')
    await waitForText(view, 'Connection info')

    await clickButtonByText(view, 'Issue token')
    await setInputValue(view, '#token-desc', `SCIM E2E ${Date.now()}`)
    await clickSummaryByText(view, 'SCIM 2.0 API')
    await view.click('input[value="scim:users:read"]')
    await view.click('input[value="scim:users:write"]')
    await clickButtonByText(view, 'Issue token')

    await waitForText(view, 'Issued the API access token.')
    expect(
      await view.evaluate(`(() => [...document.querySelectorAll('input')]
        .map((input) => input.value)
        .find((value) => value.split('.').length === 3) ?? '')()`),
    ).toMatch(/^[^.]+\.[^.]+\.[^.]+$/)
    await waitForText(view, 'scim:users:read')
    await waitForText(view, 'scim:users:write')

    await clickButtonByText(view, 'Revoke')
    await waitForText(view, 'Revoked the token.')
  } finally {
    view.close()
  }
}, 60_000)

test('admin MCP resource server lifecycle works from the browser', async () => {
  const view = new Bun.WebView({ width: 1280, height: 1800 })
  try {
    await navigateAndLogin(view, '/admin/mcp-resource-servers', 'admin-mcp-resource-servers')

    const resource = `https://mcp-${Date.now()}.example.com`
    // wi-314 T014: register/edit now live on dedicated routes reached via links,
    // not an inline form on the list page, so the resulting notice from the old
    // same-page flow no longer carries across the redirect back to the list.
    await clickLinkByText(view, 'Add resource server')
    await waitForUrl(view, /\/admin\/mcp-resource-servers\/new$/)
    await setInputValue(view, '#resource', resource)
    await setInputValue(view, '#name', 'MCP E2E')
    await setInputValue(view, '#scopes', 'mcp.read, mcp.write')
    await clickButtonByText(view, 'Register')
    await waitForText(view, resource)

    await clickLinkByText(view, 'Edit')
    await waitForUrl(view, /\/admin\/mcp-resource-servers\/[^/]+\/edit$/)
    await setInputValue(view, '#name', 'MCP E2E updated')
    await setInputValue(view, '#scopes', 'mcp.read')
    await setSelectValue(view, '#state', 'Disabled')
    await clickButtonByText(view, 'Update')
    // 'Disabled' is also literal <option> text in the still-current edit form, so wait
    // for the post-save redirect back to the list before asserting the committed state.
    await waitForUrl(view, /\/admin\/mcp-resource-servers$/)
    await waitForText(view, 'Disabled')

    await clickElementByAriaLabel(view, `Delete: ${resource}`)
    await waitForText(view, `${resource} has been deleted.`)
    await waitForText(view, 'No MCP resource servers have been registered yet.')
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
      await view.evaluate<boolean>(`(() => [...document.querySelectorAll('button')]
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
    if (needsConsent) await clickButtonByAnyText(view, ['許可', 'Allow'])
    await waitForUrl(view, /localhost:3000\/callback/)

    await view.navigate(`${uiOrigin}/account/applications`)
    await waitForPage(view, 'account-applications')
    // demo-client は client_name を持たないため、Application カタログ名 "Demo Client" へ
    // 解決される (wi-141)。UUID は補助表記に留まる。
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
      if (afterCount < beforeCount) {
        // 行が消えるのは画面上の状態にすぎない。読み直してサーバーの一覧からも
        // 消えていることを確かめ、失効が実際に保存されたことまで観測する。
        await first.navigate(`${uiOrigin}/account/activity`)
        await waitForPage(first, 'account-activity')
        await waitForText(first, 'End other sessions')
        const reloadedCount = Number(
          await first.evaluate(`(() => [...document.querySelectorAll('button')]
            .filter((button) => (button.textContent ?? '').trim() === 'End').length)()`),
        )
        expect(reloadedCount).toBeLessThan(beforeCount)
        return
      }
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
    expect(String(exportURL)).toContain('/api/admin/v1/audit_events/export')
    expect(String(exportURL)).toContain('category=authentication')
    expect(String(exportURL)).toContain('user_id=00000000-0000-4000-8000-000000000001')

    await view.navigate(searchURL)
    await waitForPage(view, 'admin-audit-events', 30_000)
    await waitForText(view, 'UserAuthenticated')
  } finally {
    view.close()
  }
}, 60_000)

test('admin audit pagination preserves addressable history and supports both ends', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    await navigateAndLogin(view, '/admin/audit_events?limit=1', 'admin-audit-events')
    const totalPages = await waitForPaginationPage(view, 1)
    expect(totalPages).toBeGreaterThan(1)

    await clickEnabledButtonByText(view, 'Next')
    const secondURL = await waitForLocationHref(view, /limit=1.*cursor=/)
    await waitForPaginationPage(view, 2)

    await view.evaluate('history.back()')
    await waitForLocationHref(view, /\/admin\/audit_events\?limit=1$/)
    await waitForPaginationPage(view, 1)
    await view.evaluate('history.forward()')
    await waitForLocationHref(view, /limit=1.*cursor=/)
    await waitForPaginationPage(view, 2)

    await view.navigate(secondURL)
    await waitForPage(view, 'admin-audit-events', 30_000)
    await waitForPaginationPage(view, 2)

    await clickEnabledButtonByText(view, 'First')
    await waitForLocationHref(view, /\/admin\/audit_events\?limit=1$/)
    await waitForPaginationPage(view, 1)

    await clickEnabledButtonByText(view, 'Last')
    await waitForLocationHref(view, /limit=1.*cursor=/)
    await waitForPaginationPage(view, totalPages)
    const lastDisabled = await view.evaluate(`(() => [...document.querySelectorAll('button')]
      .find((button) => (button.textContent ?? '').trim() === 'Last')?.disabled ?? false)()`)
    expect(lastDisabled).toBe(true)
  } finally {
    view.close()
  }
}, 90_000)

test('admin user attribute schema can add and delete a custom attribute', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2200 })
  try {
    await navigateAndLogin(view, '/admin/tenant/attributes', 'admin-tenant-attributes')

    const key = `e2e_attr_${Date.now()}`
    // wi-314 T015: adding an attribute now happens on a dedicated route reached via
    // a link, redirecting back to the list on success rather than showing a
    // same-page notice, so the new key showing up in the list is the success signal.
    await clickLinkByText(view, 'Add user attribute')
    await waitForUrl(view, /\/admin\/tenant\/attributes\/new$/)
    await setInputValue(view, '#attr-label', 'E2E attribute')
    await setInputValue(view, '#attr-key', key)
    await setSelectValue(view, '#attr-type', 'string')
    await setSelectValue(view, '#attr-visibility', 'self_readable')
    await setCheckboxValue(view, '#attr-editable', true)
    await clickButtonByText(view, 'Save')

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
    await clickLinkByText(view, 'Add application')
    await waitForUrl(view, /\/admin\/applications\/new$/)
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
    // Excludes /new: that path also matches [^/]+$, so without the negative lookahead
    // this would resolve immediately against the still-current create-page URL instead
    // of waiting for the post-creation redirect to the real detail page.
    await waitForUrl(view, /\/admin\/applications\/(?!new$)[^/]+$/)
    const appDetailURL = view.url
    await waitForText(view, appName)
    await waitForText(view, clientID)
    await clickLinkByText(view, 'Edit')
    await waitForUrl(view, /\/admin\/applications\/[^/]+\/edit$/)
    await waitForText(view, demo.username)
    await selectDropdownOption(view, 'Select…', demo.username)
    await clickButtonByText(view, 'Assign')
    await waitForText(view, demo.username)

    await view.navigate(`${uiOrigin}/admin/agents`)
    await waitForPage(view, 'admin-agents')
    // wi-314 T012: registering redirects straight to the new agent's own detail
    // page (no list-page notice to wait for), and credential bind/unbind moved
    // off the detail page onto the edit page, so it must be reached explicitly.
    await clickLinkByText(view, 'Add agent')
    await waitForUrl(view, /\/admin\/agents\/new$/)
    await setInputValue(view, '#agent-name', agentName)
    await setInputValue(view, '#agent-description', 'E2E credential binding')
    await setSelectValue(view, '#agent-kind', 'supervised')
    await setInputValue(view, '#agent-roles', 'e2e:read, e2e:write')
    await view.click('form button[type="submit"]')
    await waitForText(view, agentName)

    await clickLinkByText(view, 'Edit')
    await waitForUrl(view, /\/admin\/agents\/[^/]+\/edit$/)
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
    await view.click('aside a[href*="/admin/users/"]:not([href$="/edit"])')
    await waitForPage(view, 'admin-user-detail')
    await waitForUrl(view, /\/admin\/users\/[^/]+$/)
    await waitForText(view, 'User ID')
  } finally {
    view.close()
  }
}, 60_000)

// wi-143 / 第2層: 管理者による認証器リセット。alice (admin) は自身を対象に
// self-service で TOTP を登録し、admin console からその TOTP をリセットする。
// mfa_enrolled が false へ戻り、再登録を要求する通知が出て、ユーザー操作メニューが
// (削除済みのため) 「MFA 登録を承認」側の選択肢へ切り替わることを確認する。
// (Enrollment-required flow への実際の接続 = 次回ログインでの強制は、
// backend/shared/http/server_http の Go E2E テストで固定済み。)
test('admin can reset a user authenticator from the browser', async () => {
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

    // alice の seed 済み UUID (seed/manifests/test.yaml) に直接遷移し、一覧の並びに
    // 依存せず対象を確定する。
    await view.navigate(`${uiOrigin}/admin/users/00000000-0000-4000-8000-000000000001`)
    await waitForPage(view, 'admin-user-detail')
    await waitForText(view, 'User ID')

    await clickElementByAriaLabel(view, 'User actions')
    await clickMenuItemByText(view, 'Reset authenticators')
    await waitForText(view, 'Emergency recovery action')
    // setCheckboxValue はこの base-ui 制御コンポーネントの onChange を確実に発火
    // しないため、実クリックで切り替える。
    await view.click('#reset-target-totp')
    await clickButtonByText(view, 'Reset')
    await waitForText(view, 'The authenticators have been reset.')

    await clickElementByAriaLabel(view, 'User actions')
    await waitForText(view, 'Approve MFA enrollment for 15 minutes')
  } finally {
    view.close()
  }
}, 60_000)

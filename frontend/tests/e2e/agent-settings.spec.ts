import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  clickLinkByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

//spec:covers REQ-OAUTH2-077, REQ-IDMANAGEMENT-075: UI から独立した編集を保存し、再取得した詳細と成功通知を確認する。
test('administrator edits agent settings in separate workflows', async () => {
  const view = await openWebView({ width: 1440, height: 1800 })
  try {
    const name = `settings-${Date.now()}`
    await navigateAndLogin(view, '/realms/default/admin/agents/new', 'admin-agent-create')
    expect(await view.evaluate(`document.querySelector('#agent-roles') === null`)).toBe(true)
    await setInputValue(view, '#agent-name', name)
    // 認証用クライアントはこのテストでは作らない。
    await view.evaluate(
      `document.querySelector('input[name="auth-method"][value="later"]').click()`,
    )
    await clickButtonByText(view, 'Register')
    await waitForPage(view, 'admin-agent-detail')
    expect(
      await view.evaluate(
        `Array.from(document.querySelectorAll('dt')).some(node => node.textContent.trim() === 'Roles')`,
      ),
    ).toBe(false)
    const detailPath = String(await view.evaluate('window.location.pathname'))

    await clickLinkByText(view, 'Edit basic information')
    await waitForPage(view, 'admin-agent-edit')
    expect(
      await view.evaluate(
        `document.querySelector('#guardrail-ttl') === null && document.querySelector('#agent-credential-client') === null && document.querySelector('#agent-editor-roles') === null`,
      ),
    ).toBe(true)
    await setInputValue(view, '#agent-editor-name', `${name}-cancelled`)
    await clickLinkByText(view, 'Cancel')
    await waitForPage(view, 'admin-agent-detail')
    expect(
      await view.evaluate(
        `document.body.textContent.includes(${JSON.stringify(`${name}-cancelled`)})`,
      ),
    ).toBe(false)

    await clickLinkByText(view, 'Edit basic information')
    await waitForPage(view, 'admin-agent-edit')
    await setInputValue(view, '#agent-editor-description', 'saved basic information')
    await clickButtonByText(view, 'Save')
    await waitForPage(view, 'admin-agent-detail')
    await waitForText(view, 'saved basic information')
    expect(
      await view.evaluate(
        `document.querySelector('[role="status"]')?.textContent.includes('The agent has been updated.')`,
      ),
    ).toBe(true)

    await clickLinkByText(view, 'Edit guardrails')
    await waitForPage(view, 'admin-agent-guardrail-edit')
    await setInputValue(view, '#guardrail-ttl', '90')
    await clickButtonByText(view, 'Save guardrails')
    await waitForPage(view, 'admin-agent-detail')
    await waitForText(view, 'Saved.')
    await waitForText(view, 'Maximum token lifetime (seconds)')
    expect(
      await view.evaluate(`(() => {
      const notice = document.querySelector('[role="status"]')
      return notice?.classList.contains('bg-emerald-50') && !document.querySelector('[role="alert"]') && [...document.querySelectorAll('dd')].some(node => node.textContent === '90') && !document.querySelector('#guardrail-ttl')
    })()`),
    ).toBe(true)

    await view.navigate(`${uiOrigin}${detailPath}`)
    await waitForPage(view, 'admin-agent-detail')
    expect(await view.evaluate(`document.querySelector('[role="status"]') === null`)).toBe(true)
    await clickLinkByText(view, 'Edit guardrails')
    await waitForPage(view, 'admin-agent-guardrail-edit')
    await waitForText(view, 'Maximum token lifetime (seconds)')
    expect(await view.evaluate(`document.querySelector('#guardrail-ttl').value`)).toBe('90')
    await setInputValue(view, '#guardrail-ttl', '120')
    await clickLinkByText(view, 'Cancel')
    await waitForPage(view, 'admin-agent-detail')
    await waitForText(view, 'Maximum token lifetime (seconds)')
    expect(
      await view.evaluate(
        `[...document.querySelectorAll('dd')].some(node => node.textContent === '90')`,
      ),
    ).toBe(true)

    await clickLinkByText(view, 'Manage credentials')
    await waitForPage(view, 'admin-agent-credentials')
    await waitForText(view, 'Each addition and removal takes effect immediately.')
    expect(
      await view.evaluate(
        `!!document.querySelector('#agent-public-key') && !document.querySelector('#agent-credential-client') && ![...document.querySelectorAll('button')].some(node => node.textContent === 'Save' || node.textContent === 'Cancel')`,
      ),
    ).toBe(true)
  } finally {
    view.close()
  }
}, 60_000)

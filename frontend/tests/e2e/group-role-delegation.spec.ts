import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

//spec:covers REQ-IDMANAGEMENT-093: ブラウザーで動的グループを作り、承認と取消しの結果を再読み込みして確認する。
test('administrator approves automatic business roles', async () => {
  const view = await openWebView({ width: 1440, height: 2400 })
  try {
    await navigateAndLogin(view, '/realms/default/admin/groups/new', 'admin-group-create')
    await setInputValue(view, '#group-name', `delegation-${Date.now()}`)
    await view.evaluate(`(() => {
      const select = document.querySelector('#group-membership-type')
      select.value = 'dynamic'
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })()`)
    await setInputValue(view, '#group-dynamic-rule', 'user.email_verified == true')
    await setInputValue(view, '#group-roles', 'catalog:read')
    await clickButtonByText(view, 'Create')
    await waitForPage(view, 'admin-group-detail')
    const detailURL = String(await view.evaluate('window.location.pathname'))
    await view.navigate(`${uiOrigin}${detailURL}/edit`)
    await waitForPage(view, 'admin-group-edit')
    await waitForText(view, 'Not approved. Automatic membership does not grant business roles.')
    await clickButtonByText(view, 'Approve business role delegation')
    await waitForText(view, 'Delegation is approved for the current roles and rule.')
    await view.navigate(`${uiOrigin}${detailURL}/edit`)
    await waitForPage(view, 'admin-group-edit')
    await waitForText(view, 'Delegation is approved for the current roles and rule.')
    expect(await view.evaluate(`document.body.textContent.includes('catalog:read')`)).toBe(true)
    await setInputValue(
      view,
      'textarea[aria-label="CEL expression"]',
      'user.email_verified != false',
    )
    await clickButtonByText(view, 'Save rule')
    await waitForText(view, 'Settings have changed. Business roles require approval again.')
    await clickButtonByText(view, 'Approve business role delegation')
    await waitForText(view, 'Delegation is approved for the current roles and rule.')
    await clickButtonByText(view, 'Revoke delegation')
    await waitForText(view, 'Not approved. Automatic membership does not grant business roles.')
    await view.navigate(`${uiOrigin}${detailURL}/edit`)
    await waitForPage(view, 'admin-group-edit')
    await waitForText(view, 'Not approved. Automatic membership does not grant business roles.')
  } finally {
    view.close()
  }
}, 60_000)

import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  clickEnabledButtonByText,
  demo,
  navigateAndLogin,
  openWebView,
  setInputValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

//spec:covers REQ-TENANCY-046, REQ-TENANCY-012: realm と UUID が異なる新規テナントの上限をブラウザーから保存し、再読み込みして確認する。
test('system console saves tenant quotas by realm', async () => {
  const view = await openWebView({ width: 1440, height: 2400 })
  try {
    await navigateAndLogin(view, '/system/tenants', 'system-tenants', demo.systemAdminUsername)
    const realm = `quota-${Date.now()}`
    await clickButtonByText(view, 'New tenant')
    await setInputValue(view, '#tenant-realm', realm)
    await setInputValue(view, '#tenant-display', 'Quota acceptance')
    await clickButtonByText(view, 'Create')
    await waitForText(view, 'Created the tenant.')

    const inputID = String(await view.evaluate(`document.querySelector('input[id^="qu-"]')?.id`))
    expect(inputID).toMatch(/^qu-[0-9a-f-]{36}$/)
    await setInputValue(view, `#${inputID}`, '321')
    await clickEnabledButtonByText(view, 'Save')
    await waitForText(view, 'Updated the tenant.')

    await view.navigate(`${uiOrigin}/system/tenants`)
    await waitForPage(view, 'system-tenants')
    const selected = await view.evaluate(`(() => {
      const cell = [...document.querySelectorAll('td')].find((node) => node.textContent?.trim() === ${JSON.stringify(realm)})
      if (!cell) return false
      cell.click()
      return true
    })()`)
    expect(selected).toBe(true)
    await waitForText(view, 'Users: 0 / 321')
  } finally {
    view.close()
  }
}, 60_000)

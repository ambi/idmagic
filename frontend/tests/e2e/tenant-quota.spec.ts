import { expect, test } from 'bun:test'
import {
  clickEnabledButtonByText,
  clickLinkByText,
  demo,
  navigateAndLogin,
  openWebView,
  setInputValue,
  waitForLocationPath,
  waitForPage,
  waitForText,
} from './fixtures'

//spec:covers REQ-FRONTEND-003, REQ-TENANCY-046, REQ-TENANCY-012: 一覧から作成画面でテナントを作り、詳細から上限の変更画面で保存すると、詳細へ戻って再取得した上限が表示される。
test('system console saves tenant quotas by realm', async () => {
  const view = await openWebView({ width: 1440, height: 2400 })
  try {
    await navigateAndLogin(view, '/system/tenants', 'system-tenants', demo.systemAdminUsername)
    const realm = `quota-${Date.now()}`
    await clickLinkByText(view, 'New tenant')
    await waitForPage(view, 'system-tenant-create')
    await setInputValue(view, '#tenant-realm', realm)
    await setInputValue(view, '#tenant-display', 'Quota acceptance')
    await clickEnabledButtonByText(view, 'Create')

    await waitForPage(view, 'system-tenant-detail')
    await waitForLocationPath(view, `/system/tenants/${realm}`)
    await waitForText(view, 'Created the tenant.')

    await clickLinkByText(view, 'Edit quotas')
    await waitForPage(view, 'system-tenant-quota')
    await setInputValue(view, '#tenant-quota-users', '321')
    await clickEnabledButtonByText(view, 'Save')

    await waitForPage(view, 'system-tenant-detail')
    await waitForText(view, 'Updated the tenant.')
    const limit = await view.evaluate(`(() => {
      const row = [...document.querySelectorAll('tr')].find((node) => node.querySelector('td')?.textContent?.trim() === 'Users')
      return row?.querySelectorAll('td')[2]?.textContent?.trim()
    })()`)
    expect(limit).toBe('321')
  } finally {
    view.close()
  }
}, 60_000)

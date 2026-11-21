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

//spec:covers REQ-IDGOVERNANCE-026: ブラウザーから対象を準備し、計画の作成と確認、再読み込み、編集と次回の停止を行う。
test('administrator creates and manages an access review plan', async () => {
  const view = await openWebView({ width: 1440, height: 1800 })
  try {
    const name = `access-check-${Date.now()}`
    const groupName = `${name}-team`
    await navigateAndLogin(view, '/realms/default/admin/groups/new', 'admin-group-create')
    await setInputValue(view, '#group-name', groupName)
    await clickButtonByText(view, 'Create')
    await waitForPage(view, 'admin-group-detail')
    const groupID = String(await view.evaluate('window.location.pathname.split("/").pop()'))
    await view.navigate(`${uiOrigin}/realms/default/admin/access-reviews/new`)
    await waitForPage(view, 'admin-access-review-create')
    await setInputValue(view, '#review-name', name)
    await view.evaluate(`(() => {
      const resource = Array.from(document.querySelectorAll('input[type="checkbox"]')).find(input => input.value === ${JSON.stringify(groupID)})
      if (!resource) throw new Error('Review resource is missing')
      resource.click()
      for (const id of ['review-reviewer', 'review-fallback']) {
        const select = document.getElementById(id)
        select.value = Array.from(select.options).find(option => option.value).value
        select.dispatchEvent(new Event('change', { bubbles: true }))
      }
      const decision = document.getElementById('review-default')
      decision.value = 'revoke'
      decision.dispatchEvent(new Event('change', { bubbles: true }))
    })()`)
    await clickButtonByText(view, 'Save')
    await waitForText(view, 'Confirm default revocation')
    await clickButtonByText(view, 'Confirm')
    await waitForPage(view, 'admin-access-review-detail')
    await waitForText(view, groupName)
    const detailURL = String(await view.evaluate('window.location.pathname'))
    await view.navigate(`${uiOrigin}${detailURL}`)
    await waitForPage(view, 'admin-access-review-detail')
    expect(await view.evaluate(`document.querySelector('#review-name') === null`)).toBe(true)
    await view.navigate(`${uiOrigin}${detailURL}/edit`)
    await waitForPage(view, 'admin-access-review-edit')
    await setInputValue(view, '#review-name', `${name}-edited`)
    await clickButtonByText(view, 'Save')
    await waitForText(view, 'Confirm default revocation')
    await clickButtonByText(view, 'Confirm')
    await waitForPage(view, 'admin-access-review-detail')
    await waitForText(view, `${name}-edited`)
    await clickButtonByText(view, 'Stop future reviews')
    await waitForText(view, 'Reviews already started will continue')
    await clickButtonByText(view, 'Confirm')
    await waitForText(view, 'Enable future reviews')
    await view.navigate(`${uiOrigin}${detailURL}`)
    await waitForPage(view, 'admin-access-review-detail')
    await waitForText(view, 'Enable future reviews')
    await view.navigate(`${uiOrigin}/realms/default/account/access-reviews`)
    await waitForPage(view, 'account-access-reviews')
    await waitForText(view, 'No reviews are assigned to you.')
  } finally {
    view.close()
  }
}, 90_000)

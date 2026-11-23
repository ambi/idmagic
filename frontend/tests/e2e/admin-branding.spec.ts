import { expect, test } from 'bun:test'
import type { Branding } from '../../src/types'
import {
  clickButtonByText,
  clickElementByAriaLabel,
  navigateAndLogin,
  navigateAndLoginAsSystemAdmin,
  loginFromCurrentPage,
  metaPage,
  openWebView,
  setInputValue,
  uiOrigin,
  waitForPage,
  waitForText,
  waitForAnyText,
} from './fixtures'

//spec:covers REQ-FRONTEND-005, REQ-TENANCY-004: 管理 API の保存からブラウザーの配色、ロゴ、ポータル、focus とシステム画面の隔離までを観測する。
test('saved branding themes admin pages and leaves the system console unchanged', async () => {
  let view = await openWebView({ width: 1280, height: 960 })
  let restore: { branding: Branding; token: string; csrf: string } | undefined
  try {
    await navigateAndLogin(view, '/admin', 'admin-dashboard')
    restore = await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      const account = await api.request('/api/auth/account')
      const branding = await api.getBranding()
      const token = JSON.parse(sessionStorage.getItem('ra_oidc_token_admin')).accessToken
      await api.updateBranding(account.csrf_token, {
        product_name: 'Acme Admin', primary_color: '#ffff00', accent_color: '#663399'
      })
      return { branding, token, csrf: account.csrf_token }
    })()`)
    for (const [path, marker] of [
      ['/admin', 'admin-dashboard'],
      ['/admin/users', 'admin-users'],
      ['/admin/settings', 'admin-settings'],
    ]) {
      await view.navigate(`${uiOrigin}${path}`)
      await waitForPage(view, marker!)
      await waitForText(view, 'Acme Admin')
      await view.evaluate(
        'Promise.all(document.getAnimations().map(animation => animation.finished))',
      )
      const colors = await view.evaluate<{
        title: string
        header: string
        selected: string
        overflow: boolean
      }>(`(() => {
        const selected = document.querySelector('aside a[aria-current="page"]')
        return {
          title: getComputedStyle(document.querySelector('h1')).color,
          header: getComputedStyle(document.querySelector('header')).backgroundColor,
          selected: selected ? getComputedStyle(selected).color : '',
          overflow: document.documentElement.scrollWidth > innerWidth
        }
      })()`)
      expect(colors.title).toBe('rgb(2, 6, 23)')
      expect(colors.header).toBe('rgb(243, 239, 247)')
      expect(colors.selected).toBe('rgb(102, 51, 153)')
      expect(colors.overflow).toBe(false)
      await Bun.write(`../.verification/admin-branding-${marker}.png`, await view.screenshot())
    }

    await clickButtonByText(view, 'Branding')
    await setInputValue(view, '#branding-product-name', 'Acme Updated')
    await clickButtonByText(view, 'Save')
    await waitForText(view, 'Acme Updated')
    expect(
      await view.evaluate(
        `document.querySelector('.app-header').textContent.includes('Acme Updated')`,
      ),
    ).toBe(true)

    await view.navigate(`${uiOrigin}/admin/users`)
    await waitForPage(view, 'admin-users')
    await waitForText(view, 'Acme Updated')
    await view.evaluate(
      'Promise.all(document.getAnimations().map(animation => animation.finished))',
    )
    expect(
      await view.evaluate(`(() => {
      const action = document.querySelector('[data-slot="button"][data-variant="default"]')
      const color = getComputedStyle(action)
      return { background: color.backgroundColor, foreground: color.color }
    })()`),
    ).toEqual({ background: 'rgb(255, 255, 0)', foreground: 'rgb(0, 0, 0)' })
    await clickElementByAriaLabel(view, 'Account menu')
    expect(
      await view.evaluate(
        `getComputedStyle(document.querySelector('[role="menu"]')).getPropertyValue('--primary').trim()`,
      ),
    ).toBe('#ffff00')
    await view.press('Escape')
    await view.evaluate(`document.activeElement.blur()`)
    await view.press('Tab')
    await view.evaluate(
      'Promise.all(document.getAnimations().map(animation => animation.finished))',
    )
    const focus = await view.evaluate<{
      visible: boolean
      outline: string
      width: string
      element: string
    }>(`(() => {
      const element = document.activeElement
      const style = getComputedStyle(element)
      return { visible: element.matches(':focus-visible'), outline: style.outlineColor, width: style.outlineWidth, element: element.outerHTML.slice(0, 500) }
    })()`)
    expect(focus.visible).toBe(true)
    expect(focus).toMatchObject({ outline: 'rgb(102, 51, 153)' })
    expect(focus.width).toBe('2px')

    await view.resize(390, 844)
    await clickElementByAriaLabel(view, 'Admin navigation')
    expect(
      await view.evaluate(
        `getComputedStyle(document.querySelector('[role="menu"]')).getPropertyValue('--primary').trim()`,
      ),
    ).toBe('#ffff00')
    expect(await view.evaluate('document.documentElement.scrollWidth > innerWidth')).toBe(false)
    await Bun.write('../.verification/admin-branding-mobile.png', await view.screenshot())

    // 別のログインセッションでも保存済みのテーマを持ち込まない。
    view.close()
    view = await openWebView({ width: 1280, height: 960 })
    await navigateAndLoginAsSystemAdmin(view, '/system/tenants', 'system-tenants')
    expect(await view.evaluate(`document.querySelector('[data-admin-theme]') === null`)).toBe(true)
    expect(
      await view.evaluate(`document.querySelector('header').textContent.includes('Acme Updated')`),
    ).toBe(false)
    expect(
      await view.evaluate(
        `getComputedStyle(document.querySelector('header')).getPropertyValue('--primary').trim()`,
      ),
    ).toBe('#020617')
  } finally {
    try {
      if (restore) {
        const response = await fetch(`${uiOrigin}/realms/default/api/admin/v1/tenant/branding`, {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${restore.token}`,
            'X-CSRF-Token': restore.csrf,
          },
          body: JSON.stringify({
            product_name: restore.branding.product_name ?? '',
            primary_color: restore.branding.primary_color ?? '',
            accent_color: restore.branding.accent_color ?? '',
          }),
        })
        expect(response.status).toBe(200)
      }
    } finally {
      view.close()
    }
  }
}, 120_000)

//spec:covers REQ-SYSTEM-037: 二つの編集画面で古い保存を拒否し、入力を残して最新の読み直しを促す。
test('two branding editors detect a lost update and keep the rejected draft', async () => {
  const first = await openWebView({ width: 1280, height: 960 })
  const second = await openWebView({ width: 1280, height: 960 })
  let original: Branding | undefined
  try {
    for (const view of [first, second]) {
      await view.navigate(`${uiOrigin}/admin/settings`)
      await waitForAnyText(view, ['Gathers the settings', 'Sign in'])
      if ((await metaPage(view)) === 'login') await loginFromCurrentPage(view)
      await waitForPage(view, 'admin-settings')
      await clickButtonByText(view, 'Branding')
      await waitForText(view, 'Product name')
    }
    original = await first.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      return api.getBranding()
    })()`)
    await setInputValue(first, '#branding-product-name', 'Conditional First')
    await setInputValue(second, '#branding-product-name', 'Conditional Second')
    await clickButtonByText(first, 'Save')
    await waitForText(first, 'Updated the branding settings.')
    await clickButtonByText(second, 'Save')
    await waitForText(second, 'Your changes were not saved.')
    expect(await second.evaluate(`document.querySelector('#branding-product-name').value`)).toBe(
      'Conditional Second',
    )
    expect(
      await second.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      return (await api.getBranding()).product_name
    })()`),
    ).toBe('Conditional First')
    await Bun.write('../.verification/branding-edit-conflict.png', await second.screenshot())
  } finally {
    try {
      if (original) {
        await first.evaluate(`(async () => {
          const api = await import('/src/api.ts')
          const account = await api.request('/api/auth/account')
          return api.updateBranding(account.csrf_token, ${JSON.stringify(original)})
        })()`)
      }
    } finally {
      first.close()
      second.close()
    }
  }
}, 120_000)

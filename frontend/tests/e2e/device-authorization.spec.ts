import { expect, test } from 'bun:test'
import {
  clickButtonByAnyText,
  demo,
  navigateAndLogin,
  openWebView,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

async function deviceAdminRequest<T = void>(
  view: Bun.WebView,
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  return view.evaluate<T>(`(async () => {
    const {request, adminRequest} = await import('/src/api/core.ts')
    const context = await request('/api/auth/device')
    return await request(${JSON.stringify(path)}, adminRequest(context.csrf_token, ${JSON.stringify(method)}, ${JSON.stringify(body) ?? 'undefined'}))
  })()`)
}

//spec:covers REQ-APPLICATION-011, REQ-OAUTH2-027: 関門の拒否を承認画面に表示し、割り当て後の再承認からトークンを取得できる。
test('device approval shows access denial and succeeds after assignment', async () => {
  const view = await openWebView({ width: 1280, height: 1600 })
  try {
    await navigateAndLogin(view, '/admin/applications', 'admin-applications')
    const users = await deviceAdminRequest<{
      users: { id: string; preferred_username: string }[]
    }>(view, 'GET', '/api/admin/v1/users?query=alice')
    const userID = users.users.find(
      (user: { preferred_username: string }) => user.preferred_username === demo.username,
    )?.id
    expect(userID).toBeTruthy()
    const created = await deviceAdminRequest<{
      application: { id: string }
      client_id: string
      client_secret: string
    }>(view, 'POST', '/api/admin/v1/applications', {
      name: `Device browser ${Date.now()}`,
      type: 'oidc',
      redirect_uris: [demo.redirectUri],
      scope: 'openid profile',
    })
    const appPath = `/api/admin/v1/applications/${created.application.id}`
    await deviceAdminRequest(view, 'PATCH', `${appPath}/oidc`, {
      grant_types: ['urn:ietf:params:oauth:grant-type:device_code'],
    })
    const authorization = await fetch(`${uiOrigin}/realms/default/device_authorization`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        Authorization: `Basic ${btoa(`${created.client_id}:${created.client_secret}`)}`,
      },
      body: new URLSearchParams({ scope: 'openid' }),
    })
    expect(authorization.status).toBe(200)
    const issued = await authorization.json()
    await view.navigate(`${uiOrigin}/realms/default/device?user_code=${issued.user_code}`)
    await waitForPage(view, 'device')
    await clickButtonByAnyText(view, ['このデバイスを承認', 'Approve this device'])
    await waitForText(view, 'This action is not permitted.')
    expect(await view.evaluate('document.querySelector("[role=alert]")?.textContent')).toBeTruthy()
    expect(await view.evaluate('location.pathname')).toBe('/realms/default/device')

    await deviceAdminRequest(view, 'POST', `${appPath}/assignments`, {
      subject_type: 'user',
      subject_id: userID,
    })
    await clickButtonByAnyText(view, ['このデバイスを承認', 'Approve this device'])
    await waitForPage(view, 'status')
    expect(await view.evaluate('new URLSearchParams(location.search).get("state")')).toBe(
      'approved',
    )
    const exchanged = await fetch(`${uiOrigin}/realms/default/token`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        Authorization: `Basic ${btoa(`${created.client_id}:${created.client_secret}`)}`,
      },
      body: new URLSearchParams({
        grant_type: 'urn:ietf:params:oauth:grant-type:device_code',
        device_code: issued.device_code,
      }),
    })
    expect(exchanged.status).toBe(200)
    expect((await exchanged.json()).access_token).toBeTruthy()
  } finally {
    view.close()
  }
}, 60_000)

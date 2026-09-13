import { afterEach, beforeEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import {
  completeLoginFromCallback,
  currentBearer,
  ensureLoggedIn,
  logout,
  markPortalAuthenticated,
  recoverPortalSession,
} from './oidc'

const accountKey = 'ra_oidc_token_account'

describe('portal OIDC client', () => {
  const originalLocation = window.location

  beforeEach(() => {
    sessionStorage.clear()
    stubGlobal(
      'fetch',
      mock().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          access_token: 'renewed',
          refresh_token: 'refresh',
          expires_in: 600,
        }),
      }),
    )
    stubGlobal('location', {
      ...originalLocation,
      pathname: '/realms/acme/account',
      search: '',
      assign: mock(),
      replace: mock(),
    })
  })

  afterEach(() => restoreGlobals())

  it('uses a fresh session as the bearer without a network request', async () => {
    sessionStorage.setItem(
      accountKey,
      JSON.stringify({ accessToken: 'fresh', expiresAt: Date.now() + 120_000 }),
    )
    await ensureLoggedIn('account', '/account')
    expect(currentBearer()).toBe('fresh')
    expect(fetch).not.toHaveBeenCalled()
  })

  it('refreshes an expired session and persists the replacement token', async () => {
    sessionStorage.setItem(
      accountKey,
      JSON.stringify({ accessToken: 'old', refreshToken: 'refresh', expiresAt: 0 }),
    )
    await ensureLoggedIn('account', '/account')
    expect(currentBearer()).toBe('renewed')
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/token'),
      expect.objectContaining({
        body: expect.stringContaining('grant_type=refresh_token'),
      }),
    )
    expect(JSON.parse(sessionStorage.getItem(accountKey) ?? '{}')).toMatchObject({
      accessToken: 'renewed',
    })
  })

  it('exchanges a valid callback and redirects to its safe return path', async () => {
    sessionStorage.setItem(
      'ra_oidc_login',
      JSON.stringify({
        state: 'expected',
        verifier: 'verifier',
        audience: 'account',
        returnTo: '/account/profile',
      }),
    )
    stubGlobal('location', {
      ...originalLocation,
      pathname: '/realms/acme/callback',
      search: '?code=code&state=expected',
      assign: mock(),
      replace: mock(),
    })
    await expect(completeLoginFromCallback()).resolves.toBe(true)
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/token'), expect.anything())
    expect(window.location.replace).toHaveBeenCalledWith('/account/profile')
  })

  it('recognizes non-portal callbacks, including a mismatched stale login state', async () => {
    await expect(completeLoginFromCallback()).resolves.toBe(false)
    sessionStorage.setItem(
      'ra_oidc_login',
      JSON.stringify({ state: 'expected', verifier: 'v', audience: 'admin', returnTo: '/admin' }),
    )
    stubGlobal('location', {
      ...originalLocation,
      search: '?code=code&state=wrong',
      assign: mock(),
    })
    // A state mismatch means this /callback landing belongs to an unrelated flow
    // (e.g. the local demo authorization sharing the same route), not a corrupted
    // portal login — it must fall back rather than throw.
    await expect(completeLoginFromCallback()).resolves.toBe(false)
  })

  it('rejects a matched-state callback that carries an error param', async () => {
    sessionStorage.setItem(
      'ra_oidc_login',
      JSON.stringify({ state: 'expected', verifier: 'v', audience: 'admin', returnTo: '/admin' }),
    )
    stubGlobal('location', {
      ...originalLocation,
      search: '?error=access_denied&state=expected',
      assign: mock(),
    })
    await expect(completeLoginFromCallback()).rejects.toThrow('access_denied')
  })

  // 401 からの復旧を 4 点で見る。保持していたトークンと進行中の callback state を捨てる
  // こと、同一オリジン相対の return_to を保つこと、再認可を始めること、そしてそれを
  // 1 回だけにすることである。再認可の開始だけを見る検査は、古いトークンを残す実装と、
  // 401 のたびに再認可を繰り返して /authorize と 401 のあいだを往復する実装を通す。
  //
  //spec:covers EX-SYSTEM-015-01: 管理 API の 401 で保持していたアクセストークン・リフレッシュトークン・callback の state を破棄し、直前の画面への同一オリジン相対の return_to を保ったまま再認可を 1 回だけ開始すること。
  it('discards the stale session and re-authorizes once, keeping the return path', async () => {
    sessionStorage.setItem(
      accountKey,
      JSON.stringify({ accessToken: 'stale', refreshToken: 'refresh', expiresAt: 0 }),
    )
    sessionStorage.setItem(
      'ra_oidc_login',
      JSON.stringify({ state: 'old', verifier: 'old', audience: 'account', returnTo: '/account' }),
    )

    // 復旧はリダイレクトで終わるので解決しない。await せず、リダイレクトが起きたことを
    // 待って観測する。
    void recoverPortalSession('account', '/account/security')
    const replace = window.location.replace as ReturnType<typeof mock>
    for (let attempt = 0; attempt < 50 && replace.mock.calls.length === 0; attempt += 1) {
      await Bun.sleep(1)
    }

    // 捨てたもの: 保持していたトークン。
    expect(sessionStorage.getItem(accountKey)).toBeNull()
    // 置き換えたもの: 進行中だった callback state と、保つべき return_to。
    const login = JSON.parse(sessionStorage.getItem('ra_oidc_login') ?? '{}')
    expect(login.state).not.toBe('old')
    expect(login.returnTo).toBe('/account/security')

    // 始めたもの: 同じオリジンの /authorize へ 1 回だけ。
    expect(replace).toHaveBeenCalledTimes(1)
    const redirect = String(replace.mock.calls[0]?.[0])
    // 同一オリジン相対であること。絶対 URL を組み立てる実装は、テナントのホストを
    // 取り違えたときに別オリジンへ資格情報を渡す。
    expect(redirect.startsWith('/')).toBe(true)
    const target = new URL(redirect, 'http://localhost')
    expect(target.pathname.endsWith('/authorize')).toBe(true)
    expect(target.searchParams.get('state')).toBe(login.state)
    expect(target.searchParams.get('response_type')).toBe('code')

    // 2 度目は始めない。再ログイン直後もなお 401 なら行き止まり画面へ委ねる。
    expect(await recoverPortalSession('account', '/account/security')).toBe(false)
    expect(replace).toHaveBeenCalledTimes(1)
  })

  //spec:covers EX-SYSTEM-015-02: 再認可から復旧できないとき、再認可を繰り返さず false を返して再ログイン導線へ委ね、保持状態を残さないこと。
  it('suppresses repeated 401 recovery and clears stale state', async () => {
    sessionStorage.setItem('ra_oidc_reauth_account', String(Date.now()))
    sessionStorage.setItem(accountKey, JSON.stringify({ accessToken: 'stale', expiresAt: 0 }))
    await expect(recoverPortalSession('account', '/account')).resolves.toBe(false)
    expect(sessionStorage.getItem(accountKey)).toBeNull()
    markPortalAuthenticated('account')
    expect(sessionStorage.getItem('ra_oidc_reauth_account')).toBeNull()
  })

  it('revokes the saved token before ending the portal session', async () => {
    sessionStorage.setItem(
      accountKey,
      JSON.stringify({ accessToken: 'access', refreshToken: 'refresh', expiresAt: 0 }),
    )
    await logout('account')
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/revoke'),
      expect.objectContaining({ method: 'POST' }),
    )
    expect(window.location.assign).toHaveBeenCalledWith(expect.stringContaining('/end_session'))
  })
})

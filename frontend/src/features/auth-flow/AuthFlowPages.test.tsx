import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ForgotPasswordPage } from './ForgotPasswordPage'
import { LoginPage } from './LoginPage'
import { ResetPasswordPage } from './ResetPasswordPage'
import { ConsentPage } from './ConsentPage'
import { DevicePage } from './DevicePage'
import { EmailVerifyPage } from './EmailVerifyPage'
import { TotpPage } from './TotpPage'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

const assertionCredential = () => ({
  id: 'credential',
  rawId: new Uint8Array([1, 2, 3]).buffer,
  type: 'public-key',
  getClientExtensionResults: () => ({}),
  response: {
    authenticatorData: new Uint8Array([1]).buffer,
    clientDataJSON: new Uint8Array([2]).buffer,
    signature: new Uint8Array([3]).buffer,
    userHandle: null,
  },
})

describe('auth-flow pages', () => {
  const originalLocation = window.location

  beforeEach(() => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { next: '/continue' })))
  })

  afterEach(() => vi.unstubAllGlobals())

  it('submits login credentials and continues the browser flow', async () => {
    render(<LoginPage csrfToken="csrf" returnTo="/return" />)
    fireEvent.change(screen.getByLabelText('ユーザー名'), { target: { value: 'alice' } })
    fireEvent.change(screen.getByLabelText('パスワード'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'ログインして続行' }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/login'),
      expect.objectContaining({
        body: JSON.stringify({ username: 'alice', password: 'secret', return_to: '/return' }),
      }),
    )
  })

  it('shows a returned login error and allows a retry', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        // AuthShell が mount 時に取得する /api/branding を最初に消費する。
        .mockResolvedValueOnce(response(200, {}))
        .mockResolvedValueOnce(
          response(401, { error: 'invalid_credentials', message: '認証情報が違います' }),
        )
        .mockResolvedValueOnce(response(200, { next: '/continue' })),
    )
    render(<LoginPage csrfToken="csrf" />)
    fireEvent.change(screen.getByLabelText('ユーザー名'), { target: { value: 'alice' } })
    fireEvent.change(screen.getByLabelText('パスワード'), { target: { value: 'wrong' } })
    fireEvent.click(screen.getByRole('button', { name: 'ログインして続行' }))
    expect(await screen.findByText('認証情報が違います')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'ログインして続行' }))
    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
  })

  it('renders configured footer link labels as text', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        response(200, {
          footer_link_1: {
            label: '<img src=x onerror=alert(1)>',
            url: 'https://help.example.com',
          },
        }),
      ),
    )
    render(<LoginPage csrfToken="csrf" />)

    const link = await screen.findByRole('link', { name: '<img src=x onerror=alert(1)>' })
    expect(link).toHaveAttribute('href', 'https://help.example.com')
    expect(link.querySelector('img')).toBeNull()
  })

  it('shows only the generic reset-request confirmation after a successful submit', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(204)))
    render(<ForgotPasswordPage csrfToken="csrf" />)
    fireEvent.change(screen.getByLabelText('メールアドレス'), {
      target: { value: 'alice@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'リセットリンクを送信' }))

    expect(await screen.findByText(/アカウントが確認できた場合/)).toBeInTheDocument()
    expect(screen.getByLabelText('メールアドレス')).toBeDisabled()
  })

  it('shows an API error for a failed reset request', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(500, { message: '一時的に利用できません' })),
    )
    render(<ForgotPasswordPage csrfToken="csrf" />)
    fireEvent.change(screen.getByLabelText('メールアドレス'), {
      target: { value: 'alice@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'リセットリンクを送信' }))

    expect(await screen.findByText('一時的に利用できません')).toBeInTheDocument()
  })

  it('completes a password reset and translates password-policy errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200)))
    const { unmount } = render(<ResetPasswordPage csrfToken="csrf" token="reset-token" />)
    fireEvent.change(screen.getByLabelText('新しいパスワード'), {
      target: { value: 'a long new password' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'パスワードを更新' }))
    expect(
      await screen.findByText('パスワードを更新しました。ログインできます。'),
    ).toBeInTheDocument()
    unmount()

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(400, { error: 'password_policy' })))
    render(<ResetPasswordPage csrfToken="csrf" token="reset-token" />)
    fireEvent.change(screen.getByLabelText('新しいパスワード'), {
      target: { value: 'a long new password' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'パスワードを更新' }))
    expect(await screen.findByText(/12文字以上の、最近使用していない/)).toBeInTheDocument()
  })

  it('continues an allowed consent request and exposes a denied request failure', async () => {
    const props = { csrfToken: 'csrf', clientName: 'Portal', scopes: ['openid'] }
    const { unmount } = render(<ConsentPage {...props} />)
    fireEvent.click(screen.getByRole('button', { name: '許可して続行' }))
    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    unmount()

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(403, { message: '許可できません' })))
    render(<ConsentPage {...props} />)
    fireEvent.click(screen.getByRole('button', { name: '許可しない' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('許可できません')
  })

  it('also exposes a failure when allowing consent fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(403, { message: '許可を保存できませんでした' })),
    )
    render(<ConsentPage csrfToken="csrf" clientName="Portal" scopes={['openid']} />)
    fireEvent.click(screen.getByRole('button', { name: '許可して続行' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('許可を保存できませんでした')
  })

  it('confirms an email change and retains an actionable error on failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(204)))
    const { unmount } = render(<EmailVerifyPage csrfToken="csrf" token="verification-token" />)
    fireEvent.click(screen.getByRole('button', { name: 'メールアドレスを確認する' }))
    expect(await screen.findByText(/メールアドレスを確認しました/)).toBeInTheDocument()
    unmount()

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(400, { message: 'リンクの有効期限が切れています' })),
    )
    render(<EmailVerifyPage csrfToken="csrf" token="verification-token" />)
    fireEvent.click(screen.getByRole('button', { name: 'メールアドレスを確認する' }))
    expect(await screen.findByText('リンクの有効期限が切れています')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'メールアドレスを確認する' })).toBeEnabled()
  })

  it('submits a selected recovery code and keeps methods separate', async () => {
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp', 'recovery_code']} />)
    fireEvent.click(screen.getByRole('button', { name: 'リカバリコード' }))
    fireEvent.change(screen.getByLabelText('リカバリコード'), {
      target: { value: 'recovery-code' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'リカバリコードを確認' }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/recovery-code'),
      expect.objectContaining({ body: expect.stringContaining('recovery-code') }),
    )
  })

  it('submits a TOTP code and continues the browser flow', async () => {
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp']} />)
    fireEvent.change(screen.getByLabelText('確認コード'), { target: { value: '123456' } })
    fireEvent.click(screen.getByRole('button', { name: 'コードを確認' }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/totp'),
      expect.objectContaining({ body: expect.stringContaining('123456') }),
    )
  })

  it('shows a returned error for an invalid TOTP code', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(400, { message: 'コードが正しくありません' })),
    )
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp']} />)
    fireEvent.change(screen.getByLabelText('確認コード'), { target: { value: '000000' } })
    fireEvent.click(screen.getByRole('button', { name: 'コードを確認' }))

    expect(await screen.findByText('コードが正しくありません')).toBeInTheDocument()
  })

  it('shows a returned error for an invalid recovery code', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(400, { message: 'リカバリコードが正しくありません' })),
    )
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp', 'recovery_code']} />)
    fireEvent.click(screen.getByRole('button', { name: 'リカバリコード' }))
    fireEvent.change(screen.getByLabelText('リカバリコード'), { target: { value: 'wrong-code' } })
    fireEvent.click(screen.getByRole('button', { name: 'リカバリコードを確認' }))

    expect(await screen.findByText('リカバリコードが正しくありません')).toBeInTheDocument()
  })

  it('authenticates with a passkey and continues the browser flow', async () => {
    vi.stubGlobal('navigator', {
      credentials: { get: vi.fn().mockResolvedValue(assertionCredential()) },
    })
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => {
        if (url.includes('/webauthn/challenge')) {
          return Promise.resolve(response(200, { publicKey: { challenge: 'Y2hhbGxlbmdl' } }))
        }
        return Promise.resolve(response(200, { next: '/continue' }))
      }),
    )
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp', 'webauthn']} />)
    fireEvent.click(screen.getByRole('button', { name: 'パスキー' }))
    fireEvent.click(screen.getByRole('button', { name: 'パスキーで認証' }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
  })

  it('shows a cancellation message when the passkey prompt is dismissed', async () => {
    vi.stubGlobal('navigator', {
      credentials: {
        get: vi.fn().mockRejectedValue(new DOMException('cancelled', 'NotAllowedError')),
      },
    })
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(200, { publicKey: { challenge: 'Y2hhbGxlbmdl' } })),
    )
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp', 'webauthn']} />)
    fireEvent.click(screen.getByRole('button', { name: 'パスキー' }))
    fireEvent.click(screen.getByRole('button', { name: 'パスキーで認証' }))

    expect(await screen.findByText('パスキー認証がキャンセルされました。')).toBeInTheDocument()
  })
})

describe('DevicePage', () => {
  const originalLocation = window.location

  beforeEach(() => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
  })

  afterEach(() => vi.unstubAllGlobals())

  it('allows a device connection and continues the browser flow', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { next: '/continue' })))
    render(<DevicePage csrfToken="csrf" userCode="ABCDEFGH" />)
    fireEvent.click(screen.getByRole('button', { name: 'このデバイスを承認' }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/device'),
      expect.objectContaining({ body: expect.stringContaining('"action":"allow"') }),
    )
  })

  it('denies a device connection and continues the browser flow', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { next: '/continue' })))
    render(<DevicePage csrfToken="csrf" userCode="ABCDEFGH" />)
    fireEvent.click(screen.getByRole('button', { name: '接続を拒否' }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/device'),
      expect.objectContaining({ body: expect.stringContaining('"action":"deny"') }),
    )
  })

  it('shows an error when the device request cannot be processed', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(400, { message: 'コードが見つかりません' })),
    )
    render(<DevicePage csrfToken="csrf" userCode="ABCDEFGH" />)
    fireEvent.click(screen.getByRole('button', { name: 'このデバイスを承認' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('コードが見つかりません')
  })

  it('redirects to the status page when re-authentication is required', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(403, { error: 'authentication_required' })),
    )
    render(<DevicePage csrfToken="csrf" userCode="ABCDEFGH" />)
    fireEvent.click(screen.getByRole('button', { name: 'このデバイスを承認' }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/status?state=authentication-required'),
    )
  })
})

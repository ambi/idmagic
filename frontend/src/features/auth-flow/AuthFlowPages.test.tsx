import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ForgotPasswordPage } from './ForgotPasswordPage'
import { LoginPage } from './LoginPage'
import { ResetPasswordPage } from './ResetPasswordPage'
import { ConsentPage } from './ConsentPage'
import { EmailVerifyPage } from './EmailVerifyPage'
import { TotpPage } from './TotpPage'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
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
})

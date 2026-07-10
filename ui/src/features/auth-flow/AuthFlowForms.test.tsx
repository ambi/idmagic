import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ForgotPasswordFormPresentation } from './ForgotPasswordPage'
import { LoginFormPresentation } from './LoginPage'
import { ResetPasswordFormPresentation } from './ResetPasswordPage'
import { EmailVerificationAction } from './EmailVerifyPage'
import { ConsentActionsPresentation } from './ConsentPage'
import { DeviceCodeFormPresentation, normalizeDeviceCode } from './DevicePage'
import { availableSecondFactorMethods } from './TotpPage'
import { CallbackPage } from './CallbackPage'
import { HomePage } from './HomePage'
import { StatusPage } from './StatusPage'
import { renderWithRouter } from '../../test/renderWithRouter'

describe('LoginFormPresentation', () => {
  it('toggles password visibility through the container callback', () => {
    const onTogglePassword = vi.fn()
    render(
      <LoginFormPresentation
        submitting={false}
        showPassword={false}
        onSubmit={vi.fn()}
        onTogglePassword={onTogglePassword}
      />,
    )

    expect(screen.getByLabelText('パスワード')).toHaveAttribute('type', 'password')
    fireEvent.click(screen.getByRole('button', { name: 'パスワードを表示' }))
    expect(onTogglePassword).toHaveBeenCalledOnce()
  })

  it('disables inputs and submit while submitting', () => {
    render(
      <LoginFormPresentation
        submitting
        showPassword={false}
        onSubmit={vi.fn()}
        onTogglePassword={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('ユーザー名')).toBeDisabled()
    expect(screen.getByRole('button', { name: /確認しています/ })).toBeDisabled()
  })
})

describe('ForgotPasswordFormPresentation', () => {
  it('prevents a duplicate reset request after submission', () => {
    render(<ForgotPasswordFormPresentation submitting={false} submitted onSubmit={vi.fn()} />)

    expect(screen.getByLabelText('メールアドレス')).toBeDisabled()
    expect(screen.getByRole('button', { name: /リセットリンクを送信/ })).toBeDisabled()
  })
})

describe('ResetPasswordFormPresentation', () => {
  it('requires a valid reset token before enabling submission', () => {
    render(<ResetPasswordFormPresentation token="" submitting={false} onSubmit={vi.fn()} />)

    expect(screen.getByLabelText('新しいパスワード')).toBeDisabled()
    expect(screen.getByRole('button', { name: /パスワードを更新/ })).toBeDisabled()
  })
})

describe('EmailVerificationAction', () => {
  it('shows an invalid-link error when no token is available', () => {
    render(<EmailVerificationAction token="" state="idle" onConfirm={vi.fn()} />)

    expect(screen.getByText(/確認リンクが正しくありません/)).toBeInTheDocument()
  })
})

describe('availableSecondFactorMethods', () => {
  it('preserves supported methods in the configured order and falls back to TOTP', () => {
    expect(availableSecondFactorMethods(['recovery_code', 'webauthn'])).toEqual([
      'webauthn',
      'recovery_code',
    ])
    expect(availableSecondFactorMethods(['unknown'])).toEqual(['totp'])
  })
})

describe('DeviceCodeFormPresentation', () => {
  it('normalizes the code and only enables actions once it is complete', () => {
    const onCodeChange = vi.fn()
    render(
      <DeviceCodeFormPresentation
        code="AB"
        error=""
        submitting={false}
        onCodeChange={onCodeChange}
        onSubmit={vi.fn()}
      />,
    )

    expect(screen.getByRole('button', { name: /このデバイスを承認/ })).toBeDisabled()
    fireEvent.change(screen.getByLabelText('デバイスコード'), { target: { value: 'ab-cd efgh!' } })
    expect(onCodeChange).toHaveBeenCalledWith('ABCDEFGH')
    expect(normalizeDeviceCode('ab-cd efgh!')).toBe('ABCDEFGH')
  })
})

describe('ConsentActionsPresentation', () => {
  it('delegates both choices and prevents duplicate requests while busy', () => {
    const onConsent = vi.fn()
    const { rerender } = render(
      <ConsentActionsPresentation error="" submitting={false} onConsent={onConsent} />,
    )
    fireEvent.click(screen.getByRole('button', { name: '許可して続行' }))
    fireEvent.click(screen.getByRole('button', { name: '許可しない' }))
    expect(onConsent).toHaveBeenNthCalledWith(1, 'allow')
    expect(onConsent).toHaveBeenNthCalledWith(2, 'deny')

    rerender(<ConsentActionsPresentation error="失敗しました" submitting onConsent={onConsent} />)
    expect(screen.getByRole('alert')).toHaveTextContent('失敗しました')
    expect(screen.getByRole('button', { name: /処理しています/ })).toBeDisabled()
  })
})

describe('static auth-flow pages', () => {
  it('shows the callback success action only for a successful authorization', async () => {
    await renderWithRouter(<CallbackPage code="authorization-code" />)

    expect(screen.getByText('ローカルデモ認証が完了しました')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '管理コンソールを開く' })).toHaveAttribute(
      'href',
      '/admin',
    )
  })

  it('renders the callback failure supplied by the authorization server', async () => {
    await renderWithRouter(
      <CallbackPage error="access_denied" errorDescription="ユーザーが拒否しました" />,
    )

    expect(screen.getByText('認証を完了できませんでした')).toBeInTheDocument()
    expect(screen.getByText('ユーザーが拒否しました')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: '管理コンソールを開く' })).not.toBeInTheDocument()
  })

  it('renders demo guidance only when the local demo is enabled', async () => {
    const { unmount } = await renderWithRouter(<HomePage demoEnabled />)

    expect(screen.getByRole('button', { name: 'ローカルデモ認証を開始' })).toBeInTheDocument()
    expect(screen.getByText(/デモユーザー/)).toBeInTheDocument()
    unmount()

    await renderWithRouter(<HomePage demoEnabled={false} />)
    expect(
      screen.getByText('利用するアプリケーションからログインを開始してください。'),
    ).toBeInTheDocument()
  })

  it('shows sign-in links only after a signed-out status', async () => {
    await renderWithRouter(<StatusPage status="signed-out" />)

    expect(screen.getByText('ログアウトしました')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'マイページにログイン' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '管理コンソールにログイン' })).toBeInTheDocument()
  })
})

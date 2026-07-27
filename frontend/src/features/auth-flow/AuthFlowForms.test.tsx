import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, mock } from 'bun:test'
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
import { loginPageDictionary } from './LoginPage.i18n'
import { passwordRecoveryDictionary } from './PasswordRecoveryPages.i18n'
import { emailVerifyPageDictionary } from './EmailVerifyPage.i18n'
import { devicePageDictionary } from './DevicePage.i18n'
import { consentPageDictionary } from './ConsentPage.i18n'

const loginT = loginPageDictionary.en
const recoveryT = passwordRecoveryDictionary.en
const emailT = emailVerifyPageDictionary.en
const deviceT = devicePageDictionary.en
const consentT = consentPageDictionary.en

describe('LoginFormPresentation', () => {
  it('toggles password visibility through the container callback', () => {
    const onTogglePassword = mock()
    render(
      <LoginFormPresentation
        submitting={false}
        showPassword={false}
        onSubmit={mock()}
        onTogglePassword={onTogglePassword}
      />,
    )

    expect(screen.getByLabelText(loginT.passwordLabel)).toHaveAttribute('type', 'password')
    fireEvent.click(screen.getByRole('button', { name: loginT.showPassword }))
    expect(onTogglePassword).toHaveBeenCalledTimes(1)
  })

  it('disables inputs and submit while submitting', () => {
    render(
      <LoginFormPresentation
        submitting
        showPassword={false}
        onSubmit={mock()}
        onTogglePassword={mock()}
      />,
    )

    expect(screen.getByLabelText(loginT.usernameLabel)).toBeDisabled()
    expect(screen.getByRole('button', { name: loginT.submitting })).toBeDisabled()
  })
})

describe('ForgotPasswordFormPresentation', () => {
  it('prevents a duplicate reset request after submission', () => {
    render(<ForgotPasswordFormPresentation submitting={false} submitted onSubmit={mock()} />)

    expect(screen.getByLabelText(recoveryT.emailAddress)).toBeDisabled()
    expect(screen.getByRole('button', { name: recoveryT.sendResetLink })).toBeDisabled()
  })
})

describe('ResetPasswordFormPresentation', () => {
  it('requires a valid reset token before enabling submission', () => {
    render(<ResetPasswordFormPresentation token="" submitting={false} onSubmit={mock()} />)

    expect(screen.getByLabelText(recoveryT.newPassword)).toBeDisabled()
    expect(screen.getByRole('button', { name: recoveryT.updatePassword })).toBeDisabled()
  })
})

describe('EmailVerificationAction', () => {
  it('shows an invalid-link error when no token is available', () => {
    render(<EmailVerificationAction token="" state="idle" onConfirm={mock()} />)

    expect(screen.getByText(emailT.invalidLink)).toBeInTheDocument()
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
    const onCodeChange = mock()
    render(
      <DeviceCodeFormPresentation
        code="AB"
        error=""
        submitting={false}
        onCodeChange={onCodeChange}
        onSubmit={mock()}
      />,
    )

    expect(screen.getByRole('button', { name: deviceT.approve })).toBeDisabled()
    fireEvent.change(screen.getByLabelText(deviceT.codeLabel), {
      target: { value: 'ab-cd efgh!' },
    })
    expect(onCodeChange).toHaveBeenCalledWith('ABCDEFGH')
    expect(normalizeDeviceCode('ab-cd efgh!')).toBe('ABCDEFGH')
  })
})

describe('ConsentActionsPresentation', () => {
  it('delegates both choices and prevents duplicate requests while busy', () => {
    const onConsent = mock()
    const { rerender } = render(
      <ConsentActionsPresentation error="" submitting={false} onConsent={onConsent} />,
    )
    fireEvent.click(screen.getByRole('button', { name: consentT.allow }))
    fireEvent.click(screen.getByRole('button', { name: consentT.deny }))
    expect(onConsent).toHaveBeenNthCalledWith(1, 'allow')
    expect(onConsent).toHaveBeenNthCalledWith(2, 'deny')

    rerender(<ConsentActionsPresentation error="Request failed" submitting onConsent={onConsent} />)
    expect(screen.getByRole('alert')).toHaveTextContent('Request failed')
    expect(screen.getByRole('button', { name: consentT.processing })).toBeDisabled()
  })
})

describe('static auth-flow pages', () => {
  it('shows the callback success action only for a successful authorization', async () => {
    await renderWithRouter(<CallbackPage code="authorization-code" />)

    expect(screen.getByText('Local demo authorization is complete')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open admin console' })).toHaveAttribute(
      'href',
      '/admin',
    )
  })

  it('renders the callback failure supplied by the authorization server', async () => {
    await renderWithRouter(
      <CallbackPage error="access_denied" errorDescription="ユーザーが拒否しました" />,
    )

    expect(screen.getByText('Could not complete authentication')).toBeInTheDocument()
    expect(screen.getByText('ユーザーが拒否しました')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Open admin console' })).not.toBeInTheDocument()
  })

  it('renders demo guidance only when the local demo is enabled', async () => {
    const { unmount } = await renderWithRouter(<HomePage demoEnabled />)

    expect(
      screen.getByRole('button', { name: 'Start local demo authorization' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/Demo user/)).toBeInTheDocument()
    unmount()

    await renderWithRouter(<HomePage demoEnabled={false} />)
    expect(screen.getByText('Start signing in from the application you use.')).toBeInTheDocument()
  })

  it('shows sign-in links only after a signed-out status', async () => {
    await renderWithRouter(<StatusPage status="signed-out" />)

    expect(screen.getByText('You have signed out')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sign in to account' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sign in to admin console' })).toBeInTheDocument()
  })
})

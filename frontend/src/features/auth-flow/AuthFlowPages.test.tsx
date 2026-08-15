import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { ForgotPasswordPage } from './ForgotPasswordPage'
import { LoginPage } from './LoginPage'
import { ResetPasswordPage } from './ResetPasswordPage'
import { ConsentPage } from './ConsentPage'
import { DevicePage } from './DevicePage'
import { EmailVerifyPage } from './EmailVerifyPage'
import { TotpPage } from './TotpPage'
import { MfaEnrollmentPage } from './MfaEnrollmentPage'
import { loginPageDictionary } from './LoginPage.i18n'
import { passwordRecoveryDictionary } from './PasswordRecoveryPages.i18n'
import { consentPageDictionary } from './ConsentPage.i18n'
import { emailVerifyPageDictionary } from './EmailVerifyPage.i18n'
import { totpPageDictionary } from './TotpPage.i18n'
import { devicePageDictionary } from './DevicePage.i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { mfaEnrollmentPageDictionary } from './MfaEnrollmentPage.i18n'

const loginT = loginPageDictionary.en
const recoveryT = passwordRecoveryDictionary.en
const consentT = consentPageDictionary.en
const emailT = emailVerifyPageDictionary.en
const totpT = totpPageDictionary.en
const deviceT = devicePageDictionary.en
const commonT = commonDictionary.en
const mfaEnrollmentT = mfaEnrollmentPageDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
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
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { next: '/continue' })))
  })

  afterEach(() => restoreGlobals())

  it('submits login credentials and continues the browser flow', async () => {
    render(<LoginPage csrfToken="csrf" returnTo="/return" />)
    fireEvent.change(screen.getByLabelText(loginT.usernameLabel), { target: { value: 'alice' } })
    fireEvent.change(screen.getByLabelText(loginT.passwordLabel), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: loginT.submit }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/login'),
      expect.objectContaining({
        body: JSON.stringify({ username: 'alice', password: 'secret', return_to: '/return' }),
      }),
    )
  })

  it('renders active external identity providers as login choices', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/api/auth/federation/providers')) {
          return Promise.resolve(
            response(200, {
              providers: [{ id: 'contoso', display_name: 'Contoso', protocol: 'oidc' }],
            }),
          )
        }
        return Promise.resolve(response(200, {}))
      }),
    )

    render(<LoginPage csrfToken="csrf" returnTo="/admin" />)

    const link = await screen.findByRole('link', { name: 'Continue with Contoso' })
    expect(link).toHaveAttribute(
      'href',
      expect.stringContaining('/api/auth/federation/start?provider_id=contoso&return_to=%2Fadmin'),
    )
  })

  it('shows a returned login error and allows a retry', async () => {
    stubGlobal(
      'fetch',
      mock()
        // AuthShell が mount 時に取得する /api/branding を最初に消費する。
        .mockResolvedValueOnce(response(200, {}))
        // LoginPage が有効な外部 IdP 一覧を取得する。
        .mockResolvedValueOnce(response(200, { providers: [] }))
        .mockResolvedValueOnce(
          response(401, { error: 'invalid_credentials', message: 'The credentials are wrong.' }),
        )
        .mockResolvedValueOnce(response(200, { next: '/continue' })),
    )
    render(<LoginPage csrfToken="csrf" />)
    fireEvent.change(screen.getByLabelText(loginT.usernameLabel), { target: { value: 'alice' } })
    fireEvent.change(screen.getByLabelText(loginT.passwordLabel), { target: { value: 'wrong' } })
    fireEvent.click(screen.getByRole('button', { name: loginT.submit }))
    expect(await screen.findByText(commonT.invalidCredentials)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: loginT.submit }))
    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
  })

  it('renders configured footer link labels as text', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(
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
    stubGlobal('fetch', mock().mockResolvedValue(response(204)))
    render(<ForgotPasswordPage csrfToken="csrf" />)
    fireEvent.change(screen.getByLabelText(recoveryT.emailAddress), {
      target: { value: 'alice@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: recoveryT.sendResetLink }))

    expect(await screen.findByText(recoveryT.resetSent)).toBeInTheDocument()
    expect(screen.getByLabelText(recoveryT.emailAddress)).toBeDisabled()
  })

  it('shows an API error for a failed reset request', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(500, { message: 'Temporarily unavailable' })),
    )
    render(<ForgotPasswordPage csrfToken="csrf" />)
    fireEvent.change(screen.getByLabelText(recoveryT.emailAddress), {
      target: { value: 'alice@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: recoveryT.sendResetLink }))

    expect(await screen.findByText('Temporarily unavailable')).toBeInTheDocument()
  })

  it('completes a password reset and translates password-policy errors', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(200)))
    const { unmount } = render(<ResetPasswordPage csrfToken="csrf" token="reset-token" />)
    fireEvent.change(screen.getByLabelText(recoveryT.newPassword), {
      target: { value: 'a long new password' },
    })
    fireEvent.click(screen.getByRole('button', { name: recoveryT.updatePassword }))
    expect(await screen.findByText(recoveryT.passwordUpdated)).toBeInTheDocument()
    unmount()

    stubGlobal('fetch', mock().mockResolvedValue(response(400, { error: 'password_policy' })))
    render(<ResetPasswordPage csrfToken="csrf" token="reset-token" />)
    fireEvent.change(screen.getByLabelText(recoveryT.newPassword), {
      target: { value: 'a long new password' },
    })
    fireEvent.click(screen.getByRole('button', { name: recoveryT.updatePassword }))
    expect(await screen.findByText(recoveryT.passwordPolicy)).toBeInTheDocument()
  })

  it('continues an allowed consent request and exposes a denied request failure', async () => {
    const props = { csrfToken: 'csrf', clientName: 'Portal', scopes: ['openid'] }
    const { unmount } = render(<ConsentPage {...props} />)
    fireEvent.click(screen.getByRole('button', { name: consentT.allow }))
    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    unmount()

    stubGlobal('fetch', mock().mockResolvedValue(response(403, { message: 'Permission denied' })))
    render(<ConsentPage {...props} />)
    fireEvent.click(screen.getByRole('button', { name: consentT.deny }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Permission denied')
  })

  it('redirects to the client when denying consent succeeds', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(200, { redirect_to: '/callback?error=access_denied' })),
    )
    render(<ConsentPage csrfToken="csrf" clientName="Portal" scopes={['openid']} />)
    fireEvent.click(screen.getByRole('button', { name: consentT.deny }))
    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/callback?error=access_denied'),
    )
  })

  it('shows the consent retention note as neutral information instead of a warning', () => {
    render(<ConsentPage csrfToken="csrf" clientName="Portal" scopes={['openid']} />)

    const note = screen.getByText(consentT.retentionNote).parentElement
    expect(note).toHaveClass('bg-slate-50', 'text-slate-600')
    expect(note).not.toHaveClass('bg-amber-50/70', 'text-amber-950')
  })

  it('shows a deny-specific retry message when denying fails at the network level', async () => {
    stubGlobal('fetch', mock().mockRejectedValue(new TypeError('network down')))
    render(<ConsentPage csrfToken="csrf" clientName="Portal" scopes={['openid']} />)
    fireEvent.click(screen.getByRole('button', { name: consentT.deny }))
    expect(await screen.findByRole('alert')).toHaveTextContent(consentT.denyError)
    expect(screen.getByRole('button', { name: consentT.allow })).not.toBeDisabled()
  })

  it('also exposes a failure when allowing consent fails', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(403, { message: 'Could not save consent' })),
    )
    render(<ConsentPage csrfToken="csrf" clientName="Portal" scopes={['openid']} />)
    fireEvent.click(screen.getByRole('button', { name: consentT.allow }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not save consent')
  })

  it('confirms an email change and retains an actionable error on failure', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(204)))
    const { unmount } = render(<EmailVerifyPage csrfToken="csrf" token="verification-token" />)
    fireEvent.click(screen.getByRole('button', { name: emailT.confirmEmail }))
    expect(await screen.findByText(emailT.confirmed)).toBeInTheDocument()
    unmount()

    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(400, { message: 'The link has expired' })),
    )
    render(<EmailVerifyPage csrfToken="csrf" token="verification-token" />)
    fireEvent.click(screen.getByRole('button', { name: emailT.confirmEmail }))
    expect(await screen.findByText('The link has expired')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: emailT.confirmEmail })).toBeEnabled()
  })

  it('submits a selected recovery code and keeps methods separate', async () => {
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp', 'recovery_code']} />)
    fireEvent.click(screen.getByRole('button', { name: totpT.methodRecoveryCode }))
    fireEvent.change(screen.getByLabelText(totpT.recoveryLabel), {
      target: { value: 'recovery-code' },
    })
    fireEvent.click(screen.getByRole('button', { name: totpT.verifyRecoveryCode }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/recovery-code'),
      expect.objectContaining({ body: expect.stringContaining('recovery-code') }),
    )
  })

  it('submits a TOTP code and continues the browser flow', async () => {
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp']} />)
    fireEvent.change(screen.getByLabelText(totpT.codeLabel), { target: { value: '123456' } })
    fireEvent.click(screen.getByRole('button', { name: totpT.verifyCode }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/totp'),
      expect.objectContaining({ body: expect.stringContaining('123456') }),
    )
  })

  it('enrolls TOTP in the dedicated pending flow and continues the transaction', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/mfa/enrollment/totp/start')) {
          return Promise.resolve(
            response(200, { secret: 'SECRET', account_name: 'alice', issuer: 'idmagic' }),
          )
        }
        if (url.includes('/mfa/enrollment/totp/confirm')) {
          return Promise.resolve(response(200, { next: '/continue' }))
        }
        return Promise.resolve(response(200, {}))
      }),
    )
    render(<MfaEnrollmentPage csrfToken="csrf" />)
    expect(await screen.findByText('SECRET')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(mfaEnrollmentT.verificationCode), {
      target: { value: '123456' },
    })
    fireEvent.click(screen.getByRole('button', { name: mfaEnrollmentT.submit }))
    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/mfa/enrollment/totp/confirm'),
      expect.objectContaining({ body: expect.stringContaining('123456') }),
    )
  })

  // wi-91: 記憶の同意は既定 off で、明示的にチェックしたときだけ送る。
  it('sends remember_device only when the user ticks the checkbox', async () => {
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp']} canRememberDevice />)
    fireEvent.change(screen.getByLabelText(totpT.codeLabel), { target: { value: '123456' } })
    fireEvent.click(screen.getByRole('button', { name: totpT.verifyCode }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/totp'),
      expect.objectContaining({ body: expect.stringContaining('"remember_device":false') }),
    )
  })

  it('sends remember_device=true after the user consents', async () => {
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp']} canRememberDevice />)
    fireEvent.click(screen.getByLabelText(totpT.rememberDevice))
    fireEvent.change(screen.getByLabelText(totpT.codeLabel), { target: { value: '123456' } })
    fireEvent.click(screen.getByRole('button', { name: totpT.verifyCode }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/totp'),
      expect.objectContaining({ body: expect.stringContaining('"remember_device":true') }),
    )
  })

  // wi-91: テナントが無効にしていれば導線ごと出さない。復旧コードでも出さない。
  it('hides the remember-device consent when the tenant disables it', () => {
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp']} />)
    expect(screen.queryByLabelText(totpT.rememberDevice)).not.toBeInTheDocument()
  })

  it('hides the remember-device consent on the recovery-code method', () => {
    render(
      <TotpPage
        csrfToken="csrf"
        secondFactorMethods={['totp', 'recovery_code']}
        canRememberDevice
      />,
    )
    expect(screen.getByLabelText(totpT.rememberDevice)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: totpT.methodRecoveryCode }))
    expect(screen.queryByLabelText(totpT.rememberDevice)).not.toBeInTheDocument()
  })

  it('shows a returned error for an invalid TOTP code', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(400, { message: 'The code is invalid' })))
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp']} />)
    fireEvent.change(screen.getByLabelText(totpT.codeLabel), { target: { value: '000000' } })
    fireEvent.click(screen.getByRole('button', { name: totpT.verifyCode }))

    expect(await screen.findByText('The code is invalid')).toBeInTheDocument()
  })

  it('shows a returned error for an invalid recovery code', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(400, { message: 'The recovery code is invalid' })),
    )
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp', 'recovery_code']} />)
    fireEvent.click(screen.getByRole('button', { name: totpT.methodRecoveryCode }))
    fireEvent.change(screen.getByLabelText(totpT.recoveryLabel), {
      target: { value: 'wrong-code' },
    })
    fireEvent.click(screen.getByRole('button', { name: totpT.verifyRecoveryCode }))

    expect(await screen.findByText('The recovery code is invalid')).toBeInTheDocument()
  })

  it('authenticates with a passkey and continues the browser flow', async () => {
    stubGlobal('navigator', {
      credentials: { get: mock().mockResolvedValue(assertionCredential()) },
    })
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/webauthn/challenge')) {
          return Promise.resolve(response(200, { publicKey: { challenge: 'Y2hhbGxlbmdl' } }))
        }
        return Promise.resolve(response(200, { next: '/continue' }))
      }),
    )
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp', 'webauthn']} />)
    fireEvent.click(screen.getByRole('button', { name: totpT.methodWebauthn }))
    fireEvent.click(screen.getByRole('button', { name: totpT.authenticateWithPasskey }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
  })

  it('shows a cancellation message when the passkey prompt is dismissed', async () => {
    stubGlobal('navigator', {
      credentials: {
        get: mock().mockRejectedValue(new DOMException('cancelled', 'NotAllowedError')),
      },
    })
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(200, { publicKey: { challenge: 'Y2hhbGxlbmdl' } })),
    )
    render(<TotpPage csrfToken="csrf" secondFactorMethods={['totp', 'webauthn']} />)
    fireEvent.click(screen.getByRole('button', { name: totpT.methodWebauthn }))
    fireEvent.click(screen.getByRole('button', { name: totpT.authenticateWithPasskey }))

    expect(await screen.findByText(totpT.passkeyCancelled)).toBeInTheDocument()
  })
})

describe('DevicePage', () => {
  const originalLocation = window.location

  beforeEach(() => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
  })

  afterEach(() => restoreGlobals())

  it('allows a device connection and continues the browser flow', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { next: '/continue' })))
    render(<DevicePage csrfToken="csrf" userCode="ABCDEFGH" />)
    fireEvent.click(screen.getByRole('button', { name: deviceT.approve }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/device'),
      expect.objectContaining({ body: expect.stringContaining('"action":"allow"') }),
    )
  })

  it('denies a device connection and continues the browser flow', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { next: '/continue' })))
    render(<DevicePage csrfToken="csrf" userCode="ABCDEFGH" />)
    fireEvent.click(screen.getByRole('button', { name: deviceT.deny }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/continue'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/device'),
      expect.objectContaining({ body: expect.stringContaining('"action":"deny"') }),
    )
  })

  it('shows an error when the device request cannot be processed', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(400, { message: 'The code was not found' })),
    )
    render(<DevicePage csrfToken="csrf" userCode="ABCDEFGH" />)
    fireEvent.click(screen.getByRole('button', { name: deviceT.approve }))

    expect(await screen.findByRole('alert')).toHaveTextContent('The code was not found')
  })

  it('redirects to the status page when re-authentication is required', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(403, { error: 'authentication_required' })),
    )
    render(<DevicePage csrfToken="csrf" userCode="ABCDEFGH" />)
    fireEvent.click(screen.getByRole('button', { name: deviceT.approve }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/status?state=authentication-required'),
    )
  })
})

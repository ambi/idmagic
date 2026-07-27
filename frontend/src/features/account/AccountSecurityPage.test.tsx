import { describe, it, expect, beforeAll, afterAll, afterEach, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import {
  AccountSecurityPage,
  PasskeyList,
  PasskeyRegisterForm,
  RecoveryCodesPanel,
  TotpEnrollmentForm,
  TotpRemovalForm,
} from './AccountSecurityPage'
import { accountSecurityDictionary } from './AccountSecurityPage.i18n'
import { formatAccountSecurityDateTime } from './accountSecurityPresentation'
import { commonDictionary } from '../../lib/i18n/common.i18n'

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui)
const t = accountSecurityDictionary.en
const commonT = commonDictionary.en
import type {
  AccountSecurity,
  RecoveryCodeStatus,
  TotpEnrollmentStart,
  WebAuthnCredentialSummary,
} from '../../types'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

// isWebAuthnSupported() は window.PublicKeyCredential の有無で判定するため、
// jsdom (未対応) でも「対応ブラウザ」の分岐をテストできるよう一時的に定義する。
beforeAll(() => {
  Object.defineProperty(window, 'PublicKeyCredential', { value: class {}, configurable: true })
})

afterAll(() => {
  delete (window as { PublicKeyCredential?: unknown }).PublicKeyCredential
})

describe('formatAccountSecurityDateTime', () => {
  it('returns the English no-record label when no value is given', () => {
    expect(formatAccountSecurityDateTime(undefined)).toBe(t.noRecord)
  })

  it('formats a valid ISO date string', () => {
    expect(formatAccountSecurityDateTime('2026-01-15T10:30:00Z')).toContain('2026')
  })
})

describe('TotpEnrollmentForm', () => {
  const enrollment: TotpEnrollmentStart = {
    secret: 'SECRET123',
    otpauth_uri: 'otpauth://totp/test',
    account_name: 'taro',
    issuer: 'idmagic',
  }

  it('reports digit-only code changes', () => {
    const onEnrollCodeChange = mock()
    render(
      <TotpEnrollmentForm
        enrollment={enrollment}
        enrollCode=""
        busy={false}
        onConfirm={mock()}
        onCancel={mock()}
        onEnrollCodeChange={onEnrollCodeChange}
      />,
    )
    fireEvent.change(screen.getByLabelText(t.totpCode), {
      target: { value: 'ab12cd' },
    })
    expect(onEnrollCodeChange).toHaveBeenCalledWith('12')
  })

  it('disables submit until 6 digits are entered', () => {
    render(
      <TotpEnrollmentForm
        enrollment={enrollment}
        enrollCode="123"
        busy={false}
        onConfirm={mock()}
        onCancel={mock()}
        onEnrollCodeChange={mock()}
      />,
    )
    expect(screen.getByRole('button', { name: t.completeEnrollment })).toBeDisabled()
  })

  it('calls onCancel when cancel is clicked', () => {
    const onCancel = mock()
    render(
      <TotpEnrollmentForm
        enrollment={enrollment}
        enrollCode=""
        busy={false}
        onConfirm={mock()}
        onCancel={onCancel}
        onEnrollCodeChange={mock()}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.cancel }))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })
})

describe('TotpRemovalForm', () => {
  it('disables the remove button until 6 digits are entered', () => {
    render(
      <TotpRemovalForm
        removeCode="12"
        busy={false}
        onSubmit={mock()}
        onRemoveCodeChange={mock()}
      />,
    )
    expect(screen.getByRole('button', { name: t.removeTotp })).toBeDisabled()
  })

  it('enables the remove button once 6 digits are entered', () => {
    render(
      <TotpRemovalForm
        removeCode="123456"
        busy={false}
        onSubmit={mock()}
        onRemoveCodeChange={mock()}
      />,
    )
    expect(screen.getByRole('button', { name: t.removeTotp })).toBeEnabled()
  })
})

describe('PasskeyList', () => {
  const passkey: WebAuthnCredentialSummary = {
    credential_id: 'cred-1',
    label: 'MacBook',
    transports: ['internal'],
    created_at: '2026-01-01T00:00:00Z',
  }

  it('shows an empty state when there are no passkeys', () => {
    render(<PasskeyList passkeys={[]} busy={false} onRemove={mock()} />)
    expect(screen.getByText(t.noPasskeys)).toBeInTheDocument()
  })

  it('calls onRemove with the credential id', () => {
    const onRemove = mock()
    render(<PasskeyList passkeys={[passkey]} busy={false} onRemove={onRemove} />)
    fireEvent.click(screen.getByRole('button', { name: t.remove }))
    expect(onRemove).toHaveBeenCalledWith('cred-1')
  })
})

describe('PasskeyRegisterForm', () => {
  it('reports label changes', () => {
    const onLabelChange = mock()
    render(
      <PasskeyRegisterForm
        passkeyLabel=""
        busy={false}
        onLabelChange={onLabelChange}
        onRegister={mock()}
      />,
    )
    fireEvent.change(screen.getByLabelText(t.passkeyName), {
      target: { value: 'My Key' },
    })
    expect(onLabelChange).toHaveBeenCalledWith('My Key')
  })

  it('calls onRegister when the button is clicked', () => {
    const onRegister = mock()
    render(
      <PasskeyRegisterForm
        passkeyLabel="My Key"
        busy={false}
        onLabelChange={mock()}
        onRegister={onRegister}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.registerPasskey }))
    expect(onRegister).toHaveBeenCalledTimes(1)
  })
})

describe('RecoveryCodesPanel', () => {
  const emptyRecovery: RecoveryCodeStatus = { total: 0, remaining: 0 }
  const activeRecovery: RecoveryCodeStatus = { total: 8, remaining: 5 }

  it('shows the Generate label when there are no codes yet', () => {
    render(
      <RecoveryCodesPanel
        recovery={emptyRecovery}
        generatedCodes={null}
        busy={false}
        onGenerate={mock()}
        onRevoke={mock()}
      />,
    )
    expect(screen.getByRole('button', { name: t.generate })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.revokeAll })).not.toBeInTheDocument()
  })

  it('shows the Regenerate label and revoke button once codes exist', () => {
    render(
      <RecoveryCodesPanel
        recovery={activeRecovery}
        generatedCodes={null}
        busy={false}
        onGenerate={mock()}
        onRevoke={mock()}
      />,
    )
    expect(screen.getByRole('button', { name: t.regenerate })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.revokeAll })).toBeInTheDocument()
  })

  it('renders generated codes when present', () => {
    render(
      <RecoveryCodesPanel
        recovery={activeRecovery}
        generatedCodes={['aaaa-bbbb', 'cccc-dddd']}
        busy={false}
        onGenerate={mock()}
        onRevoke={mock()}
      />,
    )
    expect(screen.getByText('aaaa-bbbb')).toBeInTheDocument()
    expect(screen.getByText('cccc-dddd')).toBeInTheDocument()
  })

  it('calls onRevoke when the revoke button is clicked', () => {
    const onRevoke = mock()
    render(
      <RecoveryCodesPanel
        recovery={activeRecovery}
        generatedCodes={null}
        busy={false}
        onGenerate={mock()}
        onRevoke={onRevoke}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.revokeAll }))
    expect(onRevoke).toHaveBeenCalledTimes(1)
  })
})

describe('AccountSecurityPage', () => {
  const security: AccountSecurity = {
    totp_enrolled: false,
    factors: [],
    webauthn_credentials: [],
    recovery_codes: { total: 0, remaining: 0 },
  }
  const enrollment: TotpEnrollmentStart = {
    secret: 'SECRET123',
    otpauth_uri: 'otpauth://totp/test',
    account_name: 'taro',
    issuer: 'idmagic',
  }

  afterEach(() => restoreGlobals())

  it('enrolls a TOTP factor and shows a success notice', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/mfa/totp/enroll/start'))
          return Promise.resolve(response(200, enrollment))
        if (url.includes('/mfa/totp/enroll/confirm')) return Promise.resolve(response(204))
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouterBase(
      <AccountSecurityPage csrfToken="csrf" username="taro" isAdmin={false} security={security} />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.setUpTotp }))
    fireEvent.change(await screen.findByLabelText(t.totpCode), {
      target: { value: '123456' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.completeEnrollment }))

    expect(await screen.findByText(t.totpEnrolled)).toBeInTheDocument()
    expect(screen.getByText(t.configured)).toBeInTheDocument()
  })

  it('shows an error when confirming the TOTP code fails', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/mfa/totp/enroll/start'))
          return Promise.resolve(response(200, enrollment))
        if (url.includes('/mfa/totp/enroll/confirm')) {
          return Promise.resolve(response(400, { message: 'The code is invalid' }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(
      <AccountSecurityPage csrfToken="csrf" username="taro" isAdmin={false} security={security} />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.setUpTotp }))
    fireEvent.change(await screen.findByLabelText(t.totpCode), {
      target: { value: '000000' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.completeEnrollment }))

    expect(await screen.findByText('The code is invalid')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.completeEnrollment })).toBeInTheDocument()
  })

  it('keeps existing recovery codes when step-up re-authentication is cancelled', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/step_up/start')) {
          return Promise.resolve(response(200, { methods: ['password'] }))
        }
        if (url.includes('/mfa/recovery-codes/generate')) {
          return Promise.resolve(
            response(403, { message: 'Reauthentication is required', error: 'step_up_required' }),
          )
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(
      <AccountSecurityPage csrfToken="csrf" username="taro" isAdmin={false} security={security} />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.generate }))

    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: commonT.cancel }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.generate })).toBeInTheDocument()
  })

  it('lists and removes a linked external identity', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) =>
        Promise.resolve(
          url.includes('/api/account/linked-identities') ? response(204) : response(200, {}),
        ),
      ),
    )
    await renderWithRouter(
      <AccountSecurityPage
        csrfToken="csrf"
        username="taro"
        isAdmin={false}
        security={security}
        linkedIdentities={[
          {
            provider_id: 'contoso',
            local_user_id: 'user-taro',
            linked_at: '2026-07-27T10:00:00Z',
          },
        ]}
      />,
    )

    expect(screen.getByText('contoso')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Unlink contoso' }))

    expect(await screen.findByText(t.noLinkedIdentities)).toBeInTheDocument()
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/account/linked-identities/contoso'),
      expect.objectContaining({ method: 'DELETE' }),
    )
  })
})

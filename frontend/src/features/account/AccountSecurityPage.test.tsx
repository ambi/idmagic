import { describe, it, expect, vi, beforeAll, afterAll } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import {
  PasskeyList,
  PasskeyRegisterForm,
  RecoveryCodesPanel,
  TotpEnrollmentForm,
  TotpRemovalForm,
  formatAccountSecurityDateTime,
} from './AccountSecurityPage'
import type {
  RecoveryCodeStatus,
  TotpEnrollmentStart,
  WebAuthnCredentialSummary,
} from '../../types'

// isWebAuthnSupported() は window.PublicKeyCredential の有無で判定するため、
// jsdom (未対応) でも「対応ブラウザ」の分岐をテストできるよう一時的に定義する。
beforeAll(() => {
  Object.defineProperty(window, 'PublicKeyCredential', { value: class {}, configurable: true })
})

afterAll(() => {
  delete (window as { PublicKeyCredential?: unknown }).PublicKeyCredential
})

describe('formatAccountSecurityDateTime', () => {
  it('returns 記録なし when no value is given', () => {
    expect(formatAccountSecurityDateTime(undefined)).toBe('記録なし')
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
    const onEnrollCodeChange = vi.fn()
    render(
      <TotpEnrollmentForm
        enrollment={enrollment}
        enrollCode=""
        busy={false}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
        onEnrollCodeChange={onEnrollCodeChange}
      />,
    )
    fireEvent.change(screen.getByLabelText('認証アプリに表示された 6 桁コード'), {
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
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
        onEnrollCodeChange={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: '登録を完了' })).toBeDisabled()
  })

  it('calls onCancel when cancel is clicked', () => {
    const onCancel = vi.fn()
    render(
      <TotpEnrollmentForm
        enrollment={enrollment}
        enrollCode=""
        busy={false}
        onConfirm={vi.fn()}
        onCancel={onCancel}
        onEnrollCodeChange={vi.fn()}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'キャンセル' }))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })
})

describe('TotpRemovalForm', () => {
  it('disables the remove button until 6 digits are entered', () => {
    render(
      <TotpRemovalForm
        removeCode="12"
        busy={false}
        onSubmit={vi.fn()}
        onRemoveCodeChange={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: '認証アプリを解除' })).toBeDisabled()
  })

  it('enables the remove button once 6 digits are entered', () => {
    render(
      <TotpRemovalForm
        removeCode="123456"
        busy={false}
        onSubmit={vi.fn()}
        onRemoveCodeChange={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: '認証アプリを解除' })).toBeEnabled()
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
    render(<PasskeyList passkeys={[]} busy={false} onRemove={vi.fn()} />)
    expect(screen.getByText('登録済みのパスキーはありません。')).toBeInTheDocument()
  })

  it('calls onRemove with the credential id', () => {
    const onRemove = vi.fn()
    render(<PasskeyList passkeys={[passkey]} busy={false} onRemove={onRemove} />)
    fireEvent.click(screen.getByRole('button', { name: /解除/ }))
    expect(onRemove).toHaveBeenCalledWith('cred-1')
  })
})

describe('PasskeyRegisterForm', () => {
  it('reports label changes', () => {
    const onLabelChange = vi.fn()
    render(
      <PasskeyRegisterForm
        passkeyLabel=""
        busy={false}
        onLabelChange={onLabelChange}
        onRegister={vi.fn()}
      />,
    )
    fireEvent.change(screen.getByLabelText('パスキーの名前 (任意)'), {
      target: { value: 'My Key' },
    })
    expect(onLabelChange).toHaveBeenCalledWith('My Key')
  })

  it('calls onRegister when the button is clicked', () => {
    const onRegister = vi.fn()
    render(
      <PasskeyRegisterForm
        passkeyLabel="My Key"
        busy={false}
        onLabelChange={vi.fn()}
        onRegister={onRegister}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'パスキーを登録' }))
    expect(onRegister).toHaveBeenCalledTimes(1)
  })
})

describe('RecoveryCodesPanel', () => {
  const emptyRecovery: RecoveryCodeStatus = { total: 0, remaining: 0 }
  const activeRecovery: RecoveryCodeStatus = { total: 8, remaining: 5 }

  it('shows the 生成 label when there are no codes yet', () => {
    render(
      <RecoveryCodesPanel
        recovery={emptyRecovery}
        generatedCodes={null}
        busy={false}
        onGenerate={vi.fn()}
        onRevoke={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: 'リカバリコードを生成' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'すべて失効' })).not.toBeInTheDocument()
  })

  it('shows the 再生成 label and revoke button once codes exist', () => {
    render(
      <RecoveryCodesPanel
        recovery={activeRecovery}
        generatedCodes={null}
        busy={false}
        onGenerate={vi.fn()}
        onRevoke={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: 'リカバリコードを再生成' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'すべて失効' })).toBeInTheDocument()
  })

  it('renders generated codes when present', () => {
    render(
      <RecoveryCodesPanel
        recovery={activeRecovery}
        generatedCodes={['aaaa-bbbb', 'cccc-dddd']}
        busy={false}
        onGenerate={vi.fn()}
        onRevoke={vi.fn()}
      />,
    )
    expect(screen.getByText('aaaa-bbbb')).toBeInTheDocument()
    expect(screen.getByText('cccc-dddd')).toBeInTheDocument()
  })

  it('calls onRevoke when the revoke button is clicked', () => {
    const onRevoke = vi.fn()
    render(
      <RecoveryCodesPanel
        recovery={activeRecovery}
        generatedCodes={null}
        busy={false}
        onGenerate={vi.fn()}
        onRevoke={onRevoke}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'すべて失効' }))
    expect(onRevoke).toHaveBeenCalledTimes(1)
  })
})

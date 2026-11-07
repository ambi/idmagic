import { describe, it, expect, mock } from 'bun:test'
import { render, screen, fireEvent } from '@testing-library/react'
import { LocaleProvider } from '../../lib/i18n'
import { DefaultPolicyFormPresentation } from './AdminSignInPolicyPage'
import { adminSignInPolicyDictionary } from './AdminSignInPolicyPage.i18n'
import type { SignInRule } from '../../types'

const t = adminSignInPolicyDictionary.en

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

describe('DefaultPolicyFormPresentation', () => {
  const mockRule: SignInRule = {
    rule_id: 'rule-1',
    name: 'tenant-default',
    enabled: true,
    required_authn: { strength: 'Mfa' },
    condition: {
      reauth_max_age_seconds: 3600,
      network_allow_cidrs: ['192.168.1.0/24'],
    },
    mfa_enrollment: {
      enforcement_start_at: '2030-01-01T00:00:00.000Z',
      grace_period_seconds: 900,
      allow_admin_bypass: true,
    },
  }

  it('renders initial form values correctly', () => {
    const handleCancel = mock()
    const handleSubmit = mock()

    renderEn(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    expect(screen.getByText(t.strengthMfaLabel)).toBeInTheDocument()

    const reauthInput = screen.getByLabelText(t.reauthSecondsFieldLabel)
    expect(reauthInput).toHaveValue(3600)

    const cidrsTextarea = screen.getByLabelText(t.allowedNetworksCidrFieldLabel)
    expect(cidrsTextarea).toHaveValue('192.168.1.0/24')
  })

  it('submits form with valid inputs', async () => {
    const handleCancel = mock()
    const handleSubmit = mock().mockResolvedValue(undefined)

    renderEn(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    const reauthInput = screen.getByLabelText(t.reauthSecondsFieldLabel)
    fireEvent.change(reauthInput, { target: { value: '1800' } })

    const cidrsTextarea = screen.getByLabelText(t.allowedNetworksCidrFieldLabel)
    fireEvent.change(cidrsTextarea, { target: { value: '10.0.0.0/8\n172.16.0.0/12' } })

    const saveButton = screen.getByRole('button', { name: t.save })
    const form = saveButton.closest('form')
    expect(form).not.toBeNull()
    if (form) {
      fireEvent.submit(form)
    }

    expect(handleSubmit).toHaveBeenCalledTimes(1)
    expect(handleSubmit).toHaveBeenCalledWith([
      {
        rule_id: 'rule-1',
        name: 'tenant-default',
        enabled: true,
        required_authn: { strength: 'Mfa' },
        condition: {
          reauth_max_age_seconds: 1800,
          network_allow_cidrs: ['10.0.0.0/8', '172.16.0.0/12'],
        },
        mfa_enrollment: {
          enforcement_start_at: expect.any(String),
          grace_period_seconds: 900,
          allow_admin_bypass: true,
        },
      },
    ])
  })

  it('shows error message and does not submit if reauth max age is invalid', () => {
    const handleCancel = mock()
    const handleSubmit = mock()

    renderEn(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    const reauthInput = screen.getByLabelText(t.reauthSecondsFieldLabel)
    fireEvent.change(reauthInput, { target: { value: '0' } })

    const saveButton = screen.getByRole('button', { name: t.save })
    const form = saveButton.closest('form')
    expect(form).not.toBeNull()
    if (form) {
      fireEvent.submit(form)
    }

    expect(handleSubmit).not.toHaveBeenCalled()
    expect(reauthInput.getAttribute('aria-invalid')).toBe('true')
    expect(reauthInput).toHaveAccessibleDescription('Enter a whole number of 1 or more.')
  })

  it('calls onCancel when cancel button is clicked', () => {
    const handleCancel = mock()
    const handleSubmit = mock()

    renderEn(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    const cancelButton = screen.getByRole('button', { name: t.cancel })
    fireEvent.click(cancelButton)

    expect(handleCancel).toHaveBeenCalledTimes(1)
  })

  it('renders external error if provided', () => {
    const handleCancel = mock()
    const handleSubmit = mock()

    renderEn(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
        error="An API error occurred"
      />,
    )

    expect(screen.getByText('An API error occurred')).toBeInTheDocument()
  })

  it('disables save button and shows loading state when saving is true', () => {
    const handleCancel = mock()
    const handleSubmit = mock()

    renderEn(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={true}
      />,
    )

    const saveButton = screen.getByRole('button', { name: t.saving })
    expect(saveButton).toBeDisabled()

    const cancelButton = screen.getByRole('button', { name: t.cancel })
    expect(cancelButton).toBeDisabled()
  })

  //spec:covers WCAG22-LABELS-ERRORS: CIDR でない行を値で名指すエラー文と aria-invalid で示し、保存ボタンを無効にして送信しないこと
  it('names an entry that is not a CIDR and keeps the policy unsaved', () => {
    const handleSubmit = mock()
    renderEn(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={mock()}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    const cidrs = screen.getByLabelText(t.allowedNetworksCidrFieldLabel)
    fireEvent.change(cidrs, { target: { value: '10.0.0.0/8\n10.0.0.300/8' } })

    expect(cidrs.getAttribute('aria-invalid')).toBe('true')
    expect(cidrs).toHaveAccessibleDescription(
      '"10.0.0.300/8" is not a CIDR. Enter an address and a prefix length, such as 192.0.2.0/24.',
    )
    const save = screen.getByRole('button', { name: t.save })
    expect(save).toBeDisabled()
    fireEvent.submit(save.closest('form') as HTMLFormElement)
    expect(handleSubmit).not.toHaveBeenCalled()
  })
})

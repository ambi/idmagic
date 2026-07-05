import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DefaultPolicyFormPresentation } from './AdminSignInPolicyPage'
import type { SignInRule } from '../../types'

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
  }

  it('renders initial form values correctly', () => {
    const handleCancel = vi.fn()
    const handleSubmit = vi.fn()

    render(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    // 要求する認証強度 (Selectの表示テキスト)
    expect(screen.getByText('MFA 必須')).toBeInTheDocument()

    // 再認証時間
    const reauthInput = screen.getByLabelText('再認証を求めるまでの時間（秒）')
    expect(reauthInput).toHaveValue(3600)

    // 許可するネットワーク
    const cidrsTextarea = screen.getByLabelText('許可するネットワーク (CIDR)')
    expect(cidrsTextarea).toHaveValue('192.168.1.0/24')
  })

  it('submits form with valid inputs', async () => {
    const handleCancel = vi.fn()
    const handleSubmit = vi.fn().mockResolvedValue(undefined)

    render(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    const reauthInput = screen.getByLabelText('再認証を求めるまでの時間（秒）')
    fireEvent.change(reauthInput, { target: { value: '1800' } })

    const cidrsTextarea = screen.getByLabelText('許可するネットワーク (CIDR)')
    fireEvent.change(cidrsTextarea, { target: { value: '10.0.0.0/8\n172.16.0.0/12' } })

    const saveButton = screen.getByRole('button', { name: '保存' })
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
      },
    ])
  })

  it('shows error message and does not submit if reauth max age is invalid', () => {
    const handleCancel = vi.fn()
    const handleSubmit = vi.fn()

    render(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    const reauthInput = screen.getByLabelText('再認証を求めるまでの時間（秒）')
    fireEvent.change(reauthInput, { target: { value: '0' } }) // 1 未満は不正

    const saveButton = screen.getByRole('button', { name: '保存' })
    const form = saveButton.closest('form')
    expect(form).not.toBeNull()
    if (form) {
      fireEvent.submit(form)
    }

    expect(handleSubmit).not.toHaveBeenCalled()
    expect(
      screen.getByText('再認証を求めるまでの時間には 1 以上の秒数を入力してください。'),
    ).toBeInTheDocument()
  })

  it('calls onCancel when cancel button is clicked', () => {
    const handleCancel = vi.fn()
    const handleSubmit = vi.fn()

    render(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
      />,
    )

    const cancelButton = screen.getByRole('button', { name: 'キャンセル' })
    fireEvent.click(cancelButton)

    expect(handleCancel).toHaveBeenCalledTimes(1)
  })

  it('renders external error if provided', () => {
    const handleCancel = vi.fn()
    const handleSubmit = vi.fn()

    render(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={false}
        error="API エラーが発生しました"
      />,
    )

    expect(screen.getByText('API エラーが発生しました')).toBeInTheDocument()
  })

  it('disables save button and shows loading state when saving is true', () => {
    const handleCancel = vi.fn()
    const handleSubmit = vi.fn()

    render(
      <DefaultPolicyFormPresentation
        rule={mockRule}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
        saving={true}
      />,
    )

    const saveButton = screen.getByRole('button', { name: '保存中…' })
    expect(saveButton).toBeDisabled()

    const cancelButton = screen.getByRole('button', { name: 'キャンセル' })
    expect(cancelButton).toBeDisabled()
  })
})

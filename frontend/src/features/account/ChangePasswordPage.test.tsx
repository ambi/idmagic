import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { ChangePasswordPresentation, passwordViolationMessage } from './ChangePasswordPage'

describe('passwordViolationMessage', () => {
  it('translates too_short', () => {
    expect(passwordViolationMessage('too_short')).toBe('パスワードが短すぎます。')
  })

  it('translates too_long', () => {
    expect(passwordViolationMessage('too_long')).toBe('パスワードが長すぎます。')
  })

  it('falls back for unknown violations', () => {
    expect(passwordViolationMessage('something_else')).toBe(
      'パスワードがセキュリティ要件を満たしていません。',
    )
  })
})

describe('ChangePasswordPresentation', () => {
  const baseProps = {
    backHref: '/account/profile',
    backLabel: 'プロフィールへ戻る',
    preferredUsername: 'taro',
    showCurrent: false,
    showNew: false,
    error: '',
    success: false,
    submitting: false,
    dialog: null,
    onToggleShowCurrent: vi.fn(),
    onToggleShowNew: vi.fn(),
    onSubmit: vi.fn((event: React.FormEvent<HTMLFormElement>) => event.preventDefault()),
  }

  it('shows the success message once changed', async () => {
    await renderWithRouter(<ChangePasswordPresentation {...baseProps} success />)
    expect(screen.getByText('パスワードを更新しました')).toBeInTheDocument()
  })

  it('shows the error message when present', async () => {
    await renderWithRouter(
      <ChangePasswordPresentation {...baseProps} error="現在のパスワードが一致しません。" />,
    )
    expect(screen.getByText('現在のパスワードが一致しません。')).toBeInTheDocument()
  })

  it('toggles the current password visibility', async () => {
    const onToggleShowCurrent = vi.fn()
    await renderWithRouter(
      <ChangePasswordPresentation {...baseProps} onToggleShowCurrent={onToggleShowCurrent} />,
    )
    const [currentToggle] = screen.getAllByRole('button', { name: 'パスワードを表示' })
    fireEvent.click(currentToggle)
    expect(onToggleShowCurrent).toHaveBeenCalledTimes(1)
  })

  it('renders password fields as masked by default', async () => {
    await renderWithRouter(<ChangePasswordPresentation {...baseProps} />)
    const current = document.getElementById('current_password') as HTMLInputElement
    expect(current.type).toBe('password')
  })

  it('shows password fields as text when show flags are set', async () => {
    await renderWithRouter(<ChangePasswordPresentation {...baseProps} showCurrent showNew />)
    const current = document.getElementById('current_password') as HTMLInputElement
    const next = document.getElementById('new_password') as HTMLInputElement
    expect(current.type).toBe('text')
    expect(next.type).toBe('text')
  })

  it('disables the submit button while submitting', async () => {
    await renderWithRouter(<ChangePasswordPresentation {...baseProps} submitting />)
    expect(screen.getByRole('button', { name: /変更しています/ })).toBeDisabled()
  })
})

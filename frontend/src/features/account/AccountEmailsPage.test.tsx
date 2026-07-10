import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AccountEmailsPresentation } from './AccountEmailsPage'

describe('AccountEmailsPresentation', () => {
  const baseProps = {
    email: 'taro@example.com',
    emailVerified: true,
    isAdmin: false,
    newEmail: '',
    editing: false,
    submitting: false,
    error: '',
    sentTo: '',
    dialog: null,
    onStartEdit: vi.fn(),
    onCancelEdit: vi.fn(),
    onNewEmailChange: vi.fn(),
    onSubmit: vi.fn(),
  }

  it('shows the current email and verified badge', async () => {
    await renderWithRouter(<AccountEmailsPresentation {...baseProps} />)
    expect(screen.getAllByText('taro@example.com').length).toBeGreaterThan(0)
    expect(screen.getByText('確認済み')).toBeInTheDocument()
  })

  it('shows unverified badge when not verified', async () => {
    await renderWithRouter(<AccountEmailsPresentation {...baseProps} emailVerified={false} />)
    expect(screen.getByText('未確認')).toBeInTheDocument()
  })

  it('calls onStartEdit when the change button is clicked', async () => {
    const onStartEdit = vi.fn()
    await renderWithRouter(<AccountEmailsPresentation {...baseProps} onStartEdit={onStartEdit} />)
    fireEvent.click(screen.getByRole('button', { name: '変更' }))
    expect(onStartEdit).toHaveBeenCalledTimes(1)
  })

  it('renders the edit form and reports input changes when editing', async () => {
    const onNewEmailChange = vi.fn()
    await renderWithRouter(
      <AccountEmailsPresentation {...baseProps} editing onNewEmailChange={onNewEmailChange} />,
    )
    const input = screen.getByLabelText('新しいメールアドレス')
    fireEvent.change(input, { target: { value: 'new@example.com' } })
    expect(onNewEmailChange).toHaveBeenCalledWith('new@example.com')
  })

  it('disables submit until a new email is entered', async () => {
    await renderWithRouter(<AccountEmailsPresentation {...baseProps} editing newEmail="" />)
    expect(screen.getByRole('button', { name: '確認メールを送信' })).toBeDisabled()
  })

  it('shows a success message once a confirmation email is sent', async () => {
    await renderWithRouter(<AccountEmailsPresentation {...baseProps} sentTo="new@example.com" />)
    expect(screen.getByText('new@example.com')).toBeInTheDocument()
    expect(screen.getByText(/確認メールを送信しました/)).toBeInTheDocument()
  })

  it('shows an error message when present', async () => {
    await renderWithRouter(<AccountEmailsPresentation {...baseProps} error="失敗しました" />)
    expect(screen.getByText('失敗しました')).toBeInTheDocument()
  })

  it('calls onCancelEdit when cancel is clicked', async () => {
    const onCancelEdit = vi.fn()
    await renderWithRouter(
      <AccountEmailsPresentation
        {...baseProps}
        editing
        newEmail="new@example.com"
        onCancelEdit={onCancelEdit}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'キャンセル' }))
    expect(onCancelEdit).toHaveBeenCalledTimes(1)
  })
})

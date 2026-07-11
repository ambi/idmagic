import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AccountEmailsPage, AccountEmailsPresentation } from './AccountEmailsPage'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

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

describe('AccountEmailsPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  async function startEditingAndSubmit(newEmail = 'new@example.com') {
    await renderWithRouter(
      <AccountEmailsPage csrfToken="csrf" email="taro@example.com" emailVerified isAdmin={false} />,
    )
    fireEvent.click(screen.getByRole('button', { name: '変更' }))
    fireEvent.change(screen.getByLabelText('新しいメールアドレス'), {
      target: { value: newEmail },
    })
    fireEvent.click(screen.getByRole('button', { name: '確認メールを送信' }))
  }

  it('requests an email change and shows the confirmation notice', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(204)))
    await startEditingAndSubmit()

    expect(await screen.findByText(/確認メールを送信しました/)).toBeInTheDocument()
    expect(screen.getByText('new@example.com')).toBeInTheDocument()
  })

  it('shows an error message when the request fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(400, { message: 'すでに使用されています' })),
    )
    await startEditingAndSubmit()

    expect(await screen.findByText('すでに使用されています')).toBeInTheDocument()
  })

  it('keeps the form open without an error when step-up re-authentication is cancelled', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => {
        if (url.includes('/step_up/start')) {
          return Promise.resolve(response(200, { methods: ['password'] }))
        }
        return Promise.resolve(
          response(403, { message: '再認証が必要です', error: 'step_up_required' }),
        )
      }),
    )
    await startEditingAndSubmit()

    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: 'キャンセル' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.queryByText(/確認メールを送信しました/)).not.toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})

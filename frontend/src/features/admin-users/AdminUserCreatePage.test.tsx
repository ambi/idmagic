import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUserCreatePage } from './AdminUserCreatePage'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import type { AdminUser } from '../../types'

const t = adminUsersDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const user: AdminUser = {
  id: 'user-1',
  preferred_username: 'taro',
  name: 'Taro Yamada',
  email: 'taro@example.com',
  email_verified: true,
  mfa_enrolled: false,
  roles: ['support'],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('AdminUserCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('creates a user and redirects to the detail page', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(201, { ...user, id: 'user-2' }))),
    )
    await renderWithRouter(<AdminUserCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.username), { target: { value: 'jiro' } })
    fireEvent.change(screen.getByLabelText(t.initialPasswordLabel), {
      target: { value: 'correct horse battery staple' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/admin/users/user-2'))
  })

  it('shows an error and keeps the form when creation fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'This username is already in use.' }))),
    )
    await renderWithRouter(<AdminUserCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.username), { target: { value: 'jiro' } })
    fireEvent.change(screen.getByLabelText(t.initialPasswordLabel), {
      target: { value: 'correct horse battery staple' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    expect(await screen.findByText('This username is already in use.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  // REQ-IDMANAGEMENT-042
  it('shows the localized password policy message for a problem details rejection', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() =>
        Promise.resolve(
          response(422, {
            type: 'urn:idmagic:error:password_policy',
            title: 'Password policy',
            status: 422,
            detail: 'The password does not meet the security requirements.',
            violations: ['too_short'],
          }),
        ),
      ),
    )
    await renderWithRouter(<AdminUserCreatePage csrfToken="csrf" />)

    // テナントの上書きはクライアントの最小長より長い長さを求められる。
    fireEvent.change(screen.getByLabelText(t.username), { target: { value: 'jiro' } })
    fireEvent.change(screen.getByLabelText(t.initialPasswordLabel), {
      target: { value: 'twelve chars' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    expect(await screen.findByText(commonDictionary.en.passwordPolicy)).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

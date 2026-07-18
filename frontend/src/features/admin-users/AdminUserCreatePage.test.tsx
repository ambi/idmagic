import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUserCreatePage } from './AdminUserCreatePage'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import type { AdminUser } from '../../types'

const t = adminUsersDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
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
  afterEach(() => vi.unstubAllGlobals())

  it('creates a user and redirects to the detail page', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(201, { ...user, id: 'user-2' }))),
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
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(409, { message: 'This username is already in use.' }))),
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
})

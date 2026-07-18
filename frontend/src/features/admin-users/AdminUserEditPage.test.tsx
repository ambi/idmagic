import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUserEditPage } from './AdminUserEditPage'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import type { AdminUser, TenantUserAttributeSchema } from '../../types'

const t = adminUsersDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

const emptySchema: TenantUserAttributeSchema = {
  tenant_id: 'tenant-1',
  builtin: [],
  attributes: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

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

describe('AdminUserEditPage', () => {
  const originalLocation = window.location
  afterEach(() => vi.unstubAllGlobals())

  it('shows an error and keeps the form when updating profile fields fails', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(400, { message: 'Could not update the name.' }))),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.change(screen.getByLabelText(t.displayName), { target: { value: 'Jiro Yamada' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the name.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  it('requires a confirmation step before submitting a role change', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, user))),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.change(screen.getByLabelText(t.rolesHeading), { target: { value: 'admin' } })
    fireEvent.click(screen.getByRole('button', { name: t.confirmChangesHeading }))

    expect(await screen.findByText(t.roleChangeWarningTitle)).toBeInTheDocument()
    expect(fetch).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: t.confirmChanges }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/admin/users/user-1'))
  })
})

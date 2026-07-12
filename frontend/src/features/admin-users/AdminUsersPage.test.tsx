import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUsersPage, AdminUserCreatePage, AdminUserEditPage } from './AdminUsersPage'
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

describe('locale', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('renders the user list in English by default', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => {
        if (url.includes('/groups')) {
          return Promise.resolve(
            response(200, { groups: [], group_roles: [], effective_roles: user.roles }),
          )
        }
        return Promise.resolve(response(200, { users: [] }))
      }),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.disableAccount })).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => {
        if (url.includes('/groups')) {
          return Promise.resolve(
            response(200, { groups: [], group_roles: [], effective_roles: user.roles }),
          )
        }
        return Promise.resolve(response(200, { users: [] }))
      }),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />, { locale: 'ja' })
    expect(
      screen.getByRole('heading', { name: adminUsersDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminUsersPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('deletes a user and refreshes the list on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/users') && init?.method === 'DELETE') {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)

    fireEvent.click(screen.getByRole('button', { name: t.deleteAccount }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText(t.userDeleteScheduledNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectUserPrompt)).toBeInTheDocument()
  })

  it('shows an error and keeps the dialog open when deletion fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/users') && init?.method === 'DELETE') {
          return Promise.resolve(response(409, { message: 'Could not delete the user.' }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)

    fireEvent.click(screen.getByRole('button', { name: t.deleteAccount }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText('Could not delete the user.')).toBeInTheDocument()
  })

  it('shows an error when disabling a user fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => {
        if (url.includes('/disable')) {
          return Promise.resolve(response(403, { message: 'You are not allowed to disable this.' }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)

    fireEvent.click(screen.getByRole('button', { name: t.disableAccount }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: t.disableConfirm }))

    expect(await screen.findByText('You are not allowed to disable this.')).toBeInTheDocument()
  })

  it('shows an error when reloading the list fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(500, { message: 'Could not fetch the list.' }))),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)

    fireEvent.click(screen.getByRole('button', { name: t.reloadAriaLabel }))

    expect(await screen.findByText('Could not fetch the list.')).toBeInTheDocument()
  })
})

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

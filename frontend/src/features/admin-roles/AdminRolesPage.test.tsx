import { describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { AdminRolesPage } from './AdminRolesPage'
import { adminRolesDictionary } from './AdminRolesPage.i18n'
import type { AdminRole, AdminUser } from '../../types'

const role: AdminRole = {
  name: 'admin',
  description: 'Tenant administrator',
  aliases: [],
  permissions: [
    {
      name: 'ListAdminUsers',
      action: 'read',
      description: 'List users',
      interfaces: [{ method: 'GET', path: '/api/admin/users', name: 'ListAdminUsers' }],
    },
  ],
}

const user: AdminUser = {
  id: 'user-1',
  preferred_username: 'taro',
  name: 'Taro Yamada',
  email: 'taro@example.com',
  email_verified: true,
  mfa_enrolled: false,
  roles: ['admin'],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

describe('AdminRolesPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouterBase(<AdminRolesPage roles={[role]} users={[user]} />)
    expect(
      screen.getByRole('heading', { name: adminRolesDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminRolesDictionary.en.allowedOperationsHeading)).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(<AdminRolesPage roles={[role]} users={[user]} />)
    expect(
      screen.getByRole('heading', { name: adminRolesDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminRolesDictionary.ja.allowedOperationsHeading)).toBeInTheDocument()
    expect(screen.getByText('@taro')).toBeInTheDocument()
  })
})

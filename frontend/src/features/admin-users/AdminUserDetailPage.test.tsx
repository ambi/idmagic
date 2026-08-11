import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUserDetailPage } from './AdminUserDetailPage'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import type { AdminUser, TenantUserAttributeSchema } from '../../types'

const t = adminUsersDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
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
  required_actions: ['verify_email'],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('AdminUserDetailPage', () => {
  afterEach(() => restoreGlobals())

  it('is fully read-only: no group/required-action/session edit controls (policy)', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/groups')) {
          return Promise.resolve(
            response(200, { groups: [], group_roles: [], effective_roles: user.roles }),
          )
        }
        if (url.includes('/sessions')) {
          return Promise.resolve(response(200, { sessions: [] }))
        }
        return Promise.resolve(response(200, user))
      }),
    )
    await renderWithRouter(
      <AdminUserDetailPage csrfToken="csrf" user={user} schema={emptySchema} />,
    )

    // Required action is shown as an informational badge, not a toggle button.
    expect(await screen.findByText('Verify email address')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Verify email address/i })).not.toBeInTheDocument()

    // No group-leave or session-revoke controls on the read-only detail screen.
    expect(screen.queryByRole('button', { name: t.leaveGroup })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.endSession })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.revokeAllSessions })).not.toBeInTheDocument()
  })
})

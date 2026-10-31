import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { fireEvent, screen } from '@testing-library/react'
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

  //spec:covers REQ-AUTHENTICATION-022: リセットで既存のトークンが失効しても成功結果を表示すること
  it('リセット成功後は失効したトークンで再取得せず応答から状態を更新する', async () => {
    const fetcher = mock((url: string) => {
      if (url.endsWith('/authenticator-reset')) {
        return Promise.resolve(response(200, { mfa_enrolled: false, reenrollment_required: true }))
      }
      if (url.includes('/groups')) {
        return Promise.resolve(
          response(200, { groups: [], group_roles: [], effective_roles: user.roles }),
        )
      }
      if (url.includes('/sessions')) return Promise.resolve(response(200, { sessions: [] }))
      return Promise.resolve(response(401, { error: 'invalid_token' }))
    })
    stubGlobal('fetch', fetcher)
    await renderWithRouter(
      <AdminUserDetailPage
        csrfToken="csrf"
        user={{ ...user, mfa_enrolled: true }}
        schema={emptySchema}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.userActionsAriaLabel }))
    fireEvent.click(await screen.findByRole('menuitem', { name: t.resetAuthenticators }))
    fireEvent.click(screen.getByRole('checkbox', { name: t.resetTargetTotp }))
    fireEvent.click(screen.getByRole('button', { name: t.resetConfirm }))
    expect(await screen.findByText(t.authenticatorResetReenrollmentNotice)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.userActionsAriaLabel }))
    expect(
      await screen.findByRole('menuitem', { name: t.issueMfaEnrollmentBypass }),
    ).toBeInTheDocument()
    expect(fetcher.mock.calls.some(([url]) => url.endsWith('/users/user-1'))).toBe(false)
  })

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

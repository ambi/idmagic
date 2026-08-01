import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUserEditPage } from './AdminUserEditPage'
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
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

// The edit page also renders the group / required-action / session management
// sections (T010), so any fetch mock here must route those side-effect GETs
// separately from the profile PATCH under test.
function routeFetch(patchResponse: () => ReturnType<typeof response>) {
  return mock((url: string, init?: RequestInit) => {
    if (url.includes('/groups')) {
      return Promise.resolve(
        response(200, { groups: [], group_roles: [], effective_roles: user.roles }),
      )
    }
    if (url.includes('/sessions')) {
      return Promise.resolve(response(200, { sessions: [] }))
    }
    if (init?.method === 'PATCH') {
      return Promise.resolve(patchResponse())
    }
    return Promise.resolve(response(200, user))
  })
}

describe('AdminUserEditPage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('shows an error and keeps the form when updating profile fields fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      routeFetch(() => response(400, { message: 'Could not update the name.' })),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.change(screen.getByLabelText(t.displayName), { target: { value: 'Jiro Yamada' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the name.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  it('requires a confirmation step before submitting a role change', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      routeFetch(() => response(200, user)),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.change(screen.getByLabelText(t.rolesHeading), { target: { value: 'admin' } })
    fireEvent.click(screen.getByRole('button', { name: t.confirmChangesHeading }))

    expect(await screen.findByText(t.roleChangeWarningTitle)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: t.confirmChanges }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/admin/users/user-1'))
  })

  it('lets an admin toggle a required action without leaving the edit screen', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      routeFetch(() => response(200, user)),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.click(await screen.findByRole('button', { name: /Verify email address/i }))

    expect(await screen.findByText(t.requiredActionSetNotice)).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

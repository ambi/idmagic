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

  //spec:covers REQ-SYSTEM-010: API が英語で返す組み込み属性を日本語で表示し、カスタム属性のラベルと送信キーを維持する。
  it('localizes built-in attribute labels while preserving custom labels', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    const fetcher = routeFetch(() => response(200, user))
    stubGlobal('fetch', fetcher)
    const def = {
      type: 'string' as const,
      multi_valued: false,
      required: false,
      editable_by_user: true,
      visibility: 'self_readable' as const,
      pii: false,
    }
    await renderWithRouter(
      <AdminUserEditPage
        csrfToken="csrf"
        user={user}
        schema={{
          ...emptySchema,
          builtin: [
            { ...def, key: 'middle_name', label: 'Middle name', oidc_scope: 'profile' },
            { ...def, key: 'department', label: 'Department' },
          ],
          attributes: [{ ...def, key: 'staff_code', label: 'Staff code' }],
        }}
      />,
      { locale: 'ja' },
    )
    expect(screen.getByLabelText('ミドルネーム (middle_name)')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('部署 (department)'), { target: { value: '開発部' } })
    expect(screen.getByLabelText('Staff code (staff_code)')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: adminUsersDictionary.ja.save }))
    await waitFor(() => {
      const patch = fetcher.mock.calls.find(([, init]) => init?.method === 'PATCH')
      expect(JSON.parse(String(patch?.[1]?.body)).attributes.department).toEqual({
        type: 'string',
        string: '開発部',
      })
    })
  })

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

  //spec:covers WCAG22-LABELS-ERRORS: 空白だけのユーザー名を欄のエラー文と aria-invalid で示し、保存ボタンを無効にすること
  it('marks a username of only spaces invalid with a field error', async () => {
    stubGlobal(
      'fetch',
      routeFetch(() => response(200, user)),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    const username = screen.getByLabelText(t.username)
    fireEvent.change(username, { target: { value: '  ' } })

    expect(username.getAttribute('aria-invalid')).toBe('true')
    expect(username).toHaveAccessibleDescription('Enter a value for this field.')
    expect(screen.getByRole('button', { name: t.save })).toBeDisabled()
  })

  it('sends a cleared display name and trims a changed one', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    const fetchMock = routeFetch(() => response(200, user))
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.change(screen.getByLabelText(t.displayName), { target: { value: '  Jiro  ' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalled())
    const patch = fetchMock.mock.calls.find(([, init]) => init?.method === 'PATCH')
    expect(JSON.parse(String(patch?.[1]?.body))).toEqual({ name: 'Jiro' })
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

  it('lets an admin leave a group from the edit screen (T010: new leave action)', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (init?.method === 'DELETE' && url.includes('/members/')) {
          return Promise.resolve(response(204))
        }
        if (url.includes('/groups')) {
          return Promise.resolve(
            response(200, {
              groups: [
                {
                  id: 'group-1',
                  tenant_id: 'tenant-1',
                  name: 'Engineering',
                  roles: [],
                  member_count: 1,
                  created_at: '2026-01-01T00:00:00Z',
                },
              ],
              group_roles: [],
              effective_roles: user.roles,
            }),
          )
        }
        if (url.includes('/sessions')) {
          return Promise.resolve(response(200, { sessions: [] }))
        }
        return Promise.resolve(response(200, user))
      }),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.click(await screen.findByRole('button', { name: t.leaveGroup }))

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/admin/v1/groups/group-1/members/user-1'),
        expect.objectContaining({ method: 'DELETE' }),
      ),
    )
  })
})

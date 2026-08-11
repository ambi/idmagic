import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupEditPage } from './AdminGroupEditPage'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import type {
  AdminGroup,
  AdminUser,
  TenantGroupAttributeSchema,
  TenantUserAttributeSchema,
} from '../../types'

const t = adminGroupsDictionary.en

const schema: TenantUserAttributeSchema = {
  tenant_id: 'tenant-1',
  attributes: [],
  builtin: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const groupAttributeSchema: TenantGroupAttributeSchema = {
  tenant_id: 'tenant-1',
  attributes: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const group: AdminGroup = {
  id: 'group-1',
  tenant_id: 'tenant-1',
  name: 'Engineering',
  description: 'Engineering team',
  roles: ['support'],
  member_count: 0,
  created_at: '2026-01-01T00:00:00Z',
}

const user: AdminUser = {
  id: 'user-1',
  preferred_username: 'taro',
  name: 'Taro Yamada',
  email: 'taro@example.com',
  email_verified: true,
  mfa_enrolled: false,
  roles: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('AdminGroupEditPage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('shows an error and keeps the form when updating fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (init?.method === 'PATCH') {
          return Promise.resolve(response(400, { message: 'Could not update the group.' }))
        }
        if (url.includes('/api/admin/v1/groups/')) {
          return Promise.resolve(response(200, { group, members: [] }))
        }
        if (url.includes('/api/admin/v1/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(
      <AdminGroupEditPage
        csrfToken="csrf"
        group={group}
        schema={schema}
        groupAttributeSchema={groupAttributeSchema}
      />,
    )

    fireEvent.change(screen.getByLabelText(t.groupNameLabel), { target: { value: 'Platform' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the group.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  it('shows a member add control on the edit screen (T011: moved from the detail screen)', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/api/admin/v1/groups/')) {
          return Promise.resolve(response(200, { group, members: [] }))
        }
        if (url.includes('/api/admin/v1/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(
      <AdminGroupEditPage
        csrfToken="csrf"
        group={group}
        schema={schema}
        groupAttributeSchema={groupAttributeSchema}
      />,
    )

    expect(await screen.findByLabelText(t.selectUserToAddAria)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.add })).toBeInTheDocument()
  })

  it('adds the selected user as a group member', async () => {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (init?.method === 'POST' && url.includes('/members/')) {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/v1/groups/')) {
          return Promise.resolve(response(200, { group, members: [] }))
        }
        if (url.includes('/api/admin/v1/users')) {
          return Promise.resolve(response(200, { users: [user] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(
      <AdminGroupEditPage
        csrfToken="csrf"
        group={group}
        schema={schema}
        groupAttributeSchema={groupAttributeSchema}
      />,
    )

    const input = await screen.findByRole('combobox', { name: t.selectUserToAddAria })
    fireEvent.mouseDown(input)
    const option = await screen.findByRole('option', { name: user.preferred_username })
    fireEvent.click(option)
    fireEvent.click(screen.getByRole('button', { name: t.add }))

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining(`/api/admin/v1/groups/group-1/members/${user.id}`),
        expect.objectContaining({ method: 'POST' }),
      ),
    )
  })
})

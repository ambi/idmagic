import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupsPage, AdminGroupCreatePage, AdminGroupEditPage } from './AdminGroupsPage'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import type { AdminGroup } from '../../types'

const t = adminGroupsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
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

describe('locale', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('renders the group list in English by default', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, { groups: [] }))),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[]} />)
    expect(
      screen.getByRole('heading', { name: adminGroupsDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminGroupsDictionary.en.selectGroupPrompt)).toBeInTheDocument()
  })

  it('renders the group list in Japanese when explicitly selected', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, { groups: [] }))),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[]} />, { locale: 'ja' })
    expect(
      screen.getByRole('heading', { name: adminGroupsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminGroupsPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('deletes a group and refreshes the list on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/groups/group-1/members')) {
          return Promise.resolve(response(200, { members: [] }))
        }
        if (url.includes('/api/admin/groups/group-1') && init?.method === 'DELETE') {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/groups/group-1')) {
          return Promise.resolve(response(200, { group, members: [] }))
        }
        if (url.includes('/api/admin/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        if (url.includes('/api/admin/groups')) {
          return Promise.resolve(response(200, { groups: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[group]} />)

    fireEvent.click(await screen.findByRole('button', { name: t.deleteGroup }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText(t.groupDeletedNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectGroupPrompt)).toBeInTheDocument()
  })

  it('shows an error when deleting a group fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/groups/group-1') && init?.method === 'DELETE') {
          return Promise.resolve(response(409, { message: 'Could not delete the group.' }))
        }
        if (url.includes('/api/admin/groups/group-1')) {
          return Promise.resolve(response(200, { group, members: [] }))
        }
        if (url.includes('/api/admin/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[group]} />)

    fireEvent.click(await screen.findByRole('button', { name: t.deleteGroup }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText('Could not delete the group.')).toBeInTheDocument()
  })
})

describe('AdminGroupCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => vi.unstubAllGlobals())

  it('creates a group and redirects to its detail page', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(201, { ...group, id: 'group-2' }))),
    )
    await renderWithRouter(<AdminGroupCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(new RegExp(t.groupNameLabel)), {
      target: { value: 'Support' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/groups/group-2'),
    )
  })

  it('shows an error and keeps the form when creation fails', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(409, { message: 'This group name is already in use.' })),
      ),
    )
    await renderWithRouter(<AdminGroupCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(new RegExp(t.groupNameLabel)), {
      target: { value: 'Support' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    expect(await screen.findByText('This group name is already in use.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

describe('AdminGroupEditPage', () => {
  const originalLocation = window.location
  afterEach(() => vi.unstubAllGlobals())

  it('shows an error and keeps the form when updating fails', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(400, { message: 'Could not update the group.' }))),
    )
    await renderWithRouter(<AdminGroupEditPage csrfToken="csrf" group={group} />)

    fireEvent.change(screen.getByLabelText(t.groupNameLabel), { target: { value: 'Platform' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the group.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupsPage, AdminGroupCreatePage, AdminGroupEditPage } from './AdminGroupsPage'
import type { AdminGroup } from '../../types'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

const group: AdminGroup = {
  id: 'group-1',
  tenant_id: 'tenant-1',
  name: 'Engineering',
  description: 'エンジニアリングチーム',
  roles: ['support'],
  member_count: 0,
  created_at: '2026-01-01T00:00:00Z',
}

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

    fireEvent.click(await screen.findByRole('button', { name: 'グループを削除' }))
    fireEvent.click(screen.getByRole('button', { name: '削除を確定' }))

    expect(await screen.findByText('グループを削除しました。')).toBeInTheDocument()
    expect(screen.getByText('グループを選択してください。')).toBeInTheDocument()
  })

  it('shows an error when deleting a group fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/groups/group-1') && init?.method === 'DELETE') {
          return Promise.resolve(response(409, { message: 'グループを削除できませんでした。' }))
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

    fireEvent.click(await screen.findByRole('button', { name: 'グループを削除' }))
    fireEvent.click(screen.getByRole('button', { name: '削除を確定' }))

    expect(await screen.findByText('グループを削除できませんでした。')).toBeInTheDocument()
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

    fireEvent.change(screen.getByLabelText(/グループ名/), { target: { value: 'Support' } })
    fireEvent.click(screen.getByRole('button', { name: '作成' }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/groups/group-2'),
    )
  })

  it('shows an error and keeps the form when creation fails', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(409, { message: 'このグループ名は既に使われています。' })),
      ),
    )
    await renderWithRouter(<AdminGroupCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(/グループ名/), { target: { value: 'Support' } })
    fireEvent.click(screen.getByRole('button', { name: '作成' }))

    expect(await screen.findByText('このグループ名は既に使われています。')).toBeInTheDocument()
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
      vi.fn(() => Promise.resolve(response(400, { message: 'グループを更新できませんでした。' }))),
    )
    await renderWithRouter(<AdminGroupEditPage csrfToken="csrf" group={group} />)

    fireEvent.change(screen.getByLabelText('グループ名'), { target: { value: 'Platform' } })
    fireEvent.click(screen.getByRole('button', { name: '保存' }))

    expect(await screen.findByText('グループを更新できませんでした。')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

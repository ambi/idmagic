import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupCreatePage } from './AdminGroupCreatePage'
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

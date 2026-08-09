import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupsPage } from './AdminGroupsListPage'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import type { AdminGroup } from '../../types'

const t = adminGroupsDictionary.en
const tCommon = commonDictionary.en

const response = (status: number, body: unknown = {}, headers: Record<string, string> = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
  headers: new Headers(headers),
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
  afterEach(() => restoreGlobals())

  it('renders the group list in English by default', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { groups: [] }))),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[]} nextCursor={null} />)
    expect(
      screen.getByRole('heading', { name: adminGroupsDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminGroupsDictionary.en.selectGroupPrompt)).toBeInTheDocument()
  })

  it('renders the group list in Japanese when explicitly selected', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { groups: [] }))),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[]} nextCursor={null} />, {
      locale: 'ja',
    })
    expect(
      screen.getByRole('heading', { name: adminGroupsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminGroupsPage', () => {
  afterEach(() => restoreGlobals())

  it('deletes a group and refreshes the list on success', async () => {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/v1/groups/group-1/members')) {
          return Promise.resolve(response(200, { members: [] }))
        }
        if (url.includes('/api/admin/v1/groups/group-1') && init?.method === 'DELETE') {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/v1/groups/group-1')) {
          return Promise.resolve(response(200, { group, members: [] }))
        }
        if (url.includes('/api/admin/v1/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        if (url.includes('/api/admin/v1/groups')) {
          return Promise.resolve(response(200, { groups: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[group]} nextCursor={null} />)

    fireEvent.click(await screen.findByRole('button', { name: t.deleteGroup }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText(t.groupDeletedNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectGroupPrompt)).toBeInTheDocument()
  })

  it('shows an error when deleting a group fails', async () => {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/v1/groups/group-1') && init?.method === 'DELETE') {
          return Promise.resolve(response(409, { message: 'Could not delete the group.' }))
        }
        if (url.includes('/api/admin/v1/groups/group-1')) {
          return Promise.resolve(response(200, { group, members: [] }))
        }
        if (url.includes('/api/admin/v1/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[group]} nextCursor={null} />)

    fireEvent.click(await screen.findByRole('button', { name: t.deleteGroup }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText('Could not delete the group.')).toBeInTheDocument()
  })

  it('loads and appends the next page when "load more" is clicked', async () => {
    const nextGroup: AdminGroup = { ...group, id: 'group-2', name: 'Sales' }
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('cursor=abc')) {
          return Promise.resolve(response(200, { groups: [nextGroup] }))
        }
        if (url.includes('/api/admin/v1/groups/group-1')) {
          return Promise.resolve(response(200, { group, members: [] }))
        }
        if (url.includes('/api/admin/v1/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminGroupsPage csrfToken="csrf" groups={[group]} nextCursor="abc" />)

    fireEvent.click(screen.getByRole('button', { name: tCommon.loadMore }))

    expect(await screen.findByText('Sales')).toBeInTheDocument()
    expect(screen.getAllByText('Engineering').length).toBeGreaterThan(0)
  })
})

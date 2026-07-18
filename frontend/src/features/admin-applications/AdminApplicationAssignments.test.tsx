import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AssignmentList, AssignmentManager } from './AdminApplicationAssignments'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import type { AdminGroup, AdminUser, ApplicationAssignment } from '../../types'

const t = adminApplicationsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

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

const group: AdminGroup = {
  id: 'group-1',
  tenant_id: 'tenant-1',
  name: 'Engineering',
  roles: [],
  member_count: 1,
  created_at: '2026-01-01T00:00:00Z',
}

const assignment: ApplicationAssignment = {
  subject_type: 'user',
  subject_id: 'user-1',
  visibility: 'visible',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function stubFetch(
  handler: (url: string, init?: RequestInit) => ReturnType<typeof response> | undefined,
) {
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, init?: RequestInit) => {
      const result = handler(url, init)
      if (result) return Promise.resolve(result)
      throw new Error(`unexpected fetch ${url}`)
    }),
  )
}

// openSelect は Select (Radix DropdownMenu ベース) をキーボードで開き、指定ラベルの
// menuitem を選ぶ。SystemShell.test.tsx と同じ操作パターン (fireEvent.click では
// Radix の open ハンドラが発火しないため keyDown を使う)。
function chooseOption(triggerName: string | RegExp, optionName: string) {
  fireEvent.keyDown(screen.getByRole('button', { name: triggerName }), { key: 'Enter' })
  fireEvent.click(screen.getByRole('menuitem', { name: optionName }))
}

describe('AssignmentList', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('shows the empty state when there are no assignments', async () => {
    stubFetch((url) => {
      if (url.includes('/assignments')) return response(200, { assignments: [] })
      if (url.includes('/api/admin/users')) return response(200, { users: [] })
      if (url.includes('/api/admin/groups')) return response(200, { groups: [] })
      return undefined
    })
    await renderWithRouter(<AssignmentList appID="app-1" onError={() => {}} />)
    expect(await screen.findByText(t.noAssignmentsNotice)).toBeInTheDocument()
  })

  it('resolves the assigned user to a display name', async () => {
    stubFetch((url) => {
      if (url.includes('/assignments')) return response(200, { assignments: [assignment] })
      if (url.includes('/api/admin/users')) return response(200, { users: [user] })
      if (url.includes('/api/admin/groups')) return response(200, { groups: [] })
      return undefined
    })
    await renderWithRouter(<AssignmentList appID="app-1" onError={() => {}} />)
    expect(await screen.findByText('taro')).toBeInTheDocument()
    expect(screen.getByText(t.userTypeLabel)).toBeInTheDocument()
  })

  it('reports a fetch failure through onError', async () => {
    stubFetch((url) => {
      if (url.includes('/assignments'))
        return response(500, { message: 'Could not load assignments.' })
      if (url.includes('/api/admin/users')) return response(200, { users: [] })
      if (url.includes('/api/admin/groups')) return response(200, { groups: [] })
      return undefined
    })
    const onError = vi.fn()
    await renderWithRouter(<AssignmentList appID="app-1" onError={onError} />)
    await waitFor(() => expect(onError).toHaveBeenCalledWith('Could not load assignments.'))
  })
})

describe('AssignmentManager', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('assigns the selected user and shows it in the list', async () => {
    stubFetch((url, init) => {
      if (url.includes('/assignments') && init?.method === 'POST') {
        return response(201, assignment)
      }
      if (url.includes('/assignments')) return response(200, { assignments: [] })
      if (url.includes('/api/admin/users')) return response(200, { users: [user] })
      if (url.includes('/api/admin/groups')) return response(200, { groups: [] })
      return undefined
    })
    await renderWithRouter(<AssignmentManager appID="app-1" csrfToken="csrf" onError={() => {}} />)

    await screen.findByText(t.noAssignmentsShortNotice)
    chooseOption(t.selectPlaceholder, 'taro')
    fireEvent.click(screen.getByRole('button', { name: t.assign }))

    expect(await screen.findByText('taro')).toBeInTheDocument()
  })

  it('unassigns a subject and removes it from the list', async () => {
    stubFetch((url, init) => {
      if (url.includes('/assignments') && init?.method === 'DELETE') {
        return response(204)
      }
      if (url.includes('/assignments')) return response(200, { assignments: [assignment] })
      if (url.includes('/api/admin/users')) return response(200, { users: [user] })
      if (url.includes('/api/admin/groups')) return response(200, { groups: [] })
      return undefined
    })
    await renderWithRouter(<AssignmentManager appID="app-1" csrfToken="csrf" onError={() => {}} />)

    const row = (await screen.findByText('taro')).closest('li') as HTMLElement
    fireEvent.click(within(row).getByRole('button', { name: t.unassign }))

    await waitFor(() => expect(screen.queryByText('taro')).not.toBeInTheDocument())
  })

  it('reports an assignment failure through onError', async () => {
    stubFetch((url, init) => {
      if (url.includes('/assignments') && init?.method === 'POST') {
        return response(409, { message: 'This user is already assigned.' })
      }
      if (url.includes('/assignments')) return response(200, { assignments: [] })
      if (url.includes('/api/admin/users')) return response(200, { users: [user] })
      if (url.includes('/api/admin/groups')) return response(200, { groups: [] })
      return undefined
    })
    const onError = vi.fn()
    await renderWithRouter(<AssignmentManager appID="app-1" csrfToken="csrf" onError={onError} />)

    await screen.findByText(t.noAssignmentsShortNotice)
    chooseOption(t.selectPlaceholder, 'taro')
    fireEvent.click(screen.getByRole('button', { name: t.assign }))

    await waitFor(() => expect(onError).toHaveBeenCalledWith('This user is already assigned.'))
  })

  it('offers the group target type and lists the group as an assignable target', async () => {
    stubFetch((url) => {
      if (url.includes('/assignments')) return response(200, { assignments: [] })
      if (url.includes('/api/admin/users')) return response(200, { users: [] })
      if (url.includes('/api/admin/groups')) return response(200, { groups: [group] })
      return undefined
    })
    await renderWithRouter(<AssignmentManager appID="app-1" csrfToken="csrf" onError={() => {}} />)

    await screen.findByText(t.noAssignmentsShortNotice)
    chooseOption(t.userTypeLabel, t.groupTypeLabel)
    fireEvent.keyDown(screen.getByRole('button', { name: t.selectPlaceholder }), { key: 'Enter' })
    expect(await screen.findByRole('menuitem', { name: 'Engineering' })).toBeInTheDocument()
  })
})

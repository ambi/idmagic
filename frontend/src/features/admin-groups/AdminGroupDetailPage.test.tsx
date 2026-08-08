import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupDetailPage } from './AdminGroupDetailPage'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import type { AdminGroup } from '../../types'

const t = adminGroupsDictionary.en

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
  member_count: 1,
  created_at: '2026-01-01T00:00:00Z',
}

describe('AdminGroupDetailPage', () => {
  afterEach(() => restoreGlobals())

  it('is fully read-only: no member add/remove controls (ADR-086 policy, wi-314 T011)', async () => {
    stubGlobal(
      'fetch',
      mock((url: string) => {
        if (url.includes('/api/admin/v1/groups/')) {
          return Promise.resolve(
            response(200, {
              group,
              members: [{ user_id: 'user-1', preferred_username: 'taro', source: 'manual' }],
            }),
          )
        }
        if (url.includes('/api/admin/v1/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminGroupDetailPage csrfToken="csrf" group={group} />)

    expect(await screen.findByText('taro')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.removeMember })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.add })).not.toBeInTheDocument()
    expect(screen.queryByLabelText(t.selectUserToAddAria)).not.toBeInTheDocument()
  })
})

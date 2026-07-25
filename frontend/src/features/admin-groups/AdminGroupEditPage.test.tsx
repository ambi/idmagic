import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminGroupEditPage } from './AdminGroupEditPage'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import type { AdminGroup, TenantUserAttributeSchema } from '../../types'

const t = adminGroupsDictionary.en

const schema: TenantUserAttributeSchema = {
  tenant_id: 'tenant-1',
  attributes: [],
  builtin: [],
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

describe('AdminGroupEditPage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('shows an error and keeps the form when updating fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { message: 'Could not update the group.' }))),
    )
    await renderWithRouter(<AdminGroupEditPage csrfToken="csrf" group={group} schema={schema} />)

    fireEvent.change(screen.getByLabelText(t.groupNameLabel), { target: { value: 'Platform' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the group.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

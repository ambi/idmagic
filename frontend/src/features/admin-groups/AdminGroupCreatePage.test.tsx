import { afterEach, describe, it, expect, mock } from 'bun:test'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { AdminGroupCreatePage } from './AdminGroupCreatePage'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import type { AdminGroup, TenantGroupAttributeSchema } from '../../types'

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
  member_count: 0,
  created_at: '2026-01-01T00:00:00Z',
}

const groupAttributeSchema: TenantGroupAttributeSchema = {
  tenant_id: 'tenant-1',
  attributes: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('AdminGroupCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('creates a group and redirects to its detail page', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(201, { ...group, id: 'group-2' }))),
    )
    await renderWithRouter(
      <AdminGroupCreatePage csrfToken="csrf" groupAttributeSchema={groupAttributeSchema} />,
    )

    fireEvent.change(screen.getByLabelText(new RegExp(t.groupNameLabel)), {
      target: { value: 'Support' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/groups/group-2'),
    )
  })

  it('shows an error and keeps the form when creation fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'This group name is already in use.' }))),
    )
    await renderWithRouter(
      <AdminGroupCreatePage csrfToken="csrf" groupAttributeSchema={groupAttributeSchema} />,
    )

    fireEvent.change(screen.getByLabelText(new RegExp(t.groupNameLabel)), {
      target: { value: 'Support' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    expect(await screen.findByText('This group name is already in use.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  //spec:covers WCAG22-LABELS-ERRORS: 空白だけの名前と、上限を超える説明を欄のエラー文と aria-invalid で示し、作成ボタンを無効にすること
  it('marks a blank name and a description over 500 code points invalid', async () => {
    await renderWithRouter(
      <AdminGroupCreatePage csrfToken="csrf" groupAttributeSchema={groupAttributeSchema} />,
    )

    const name = screen.getByLabelText(new RegExp(t.groupNameLabel))
    fireEvent.change(name, { target: { value: '   ' } })
    expect(name.getAttribute('aria-invalid')).toBe('true')
    expect(name).toHaveAccessibleDescription('Enter a value for this field.')

    fireEvent.change(name, { target: { value: 'engineering' } })
    const description = screen.getByLabelText(new RegExp(t.descriptionOptionalLabel))
    fireEvent.change(description, { target: { value: '漢'.repeat(501) } })
    expect(description.getAttribute('aria-invalid')).toBe('true')
    expect(description).toHaveAccessibleDescription('Enter 500 characters or fewer.')
    expect(screen.getByRole('button', { name: t.create })).toBeDisabled()

    fireEvent.change(description, { target: { value: '漢'.repeat(500) } })
    expect(screen.getByRole('button', { name: t.create })).toBeEnabled()
  })
})

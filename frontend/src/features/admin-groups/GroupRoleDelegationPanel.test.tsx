import { afterEach, expect, it, mock } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminGroup } from '../../types'
import { GroupRoleDelegationPanel } from './GroupRoleDelegationPanel'
import { groupRoleDelegationDictionary } from './GroupRoleDelegationPanel.i18n'

const t = groupRoleDelegationDictionary.en
const group: AdminGroup = {
  id: 'automatic',
  tenant_id: 'tenant',
  name: 'Automatic',
  roles: ['catalog:read'],
  member_count: 1,
  created_at: '2026-01-01T00:00:00Z',
  upstream_managed: true,
}
afterEach(restoreGlobals)

//spec:covers REQ-IDMANAGEMENT-093: 委任の範囲を表示し、承認と取消しの API 結果で状態を更新する。
it('approves the displayed roles and revokes delegation', async () => {
  const fetchMock = mock((_url: string, init: RequestInit) => {
    const input = JSON.parse(String(init.body))
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => ({
        ...group,
        role_delegation: input.roles.length
          ? { ...input, approved_by: 'admin', approved_at: group.created_at }
          : undefined,
      }),
    })
  })
  stubGlobal('fetch', fetchMock)
  await renderWithRouter(
    <GroupRoleDelegationPanel csrfToken="csrf" group={group} unsavedChanges={false} />,
  )
  expect(screen.getByText(t.explanation)).toBeInTheDocument()
  expect(screen.getByText(t.adminNotice)).toBeInTheDocument()
  expect(screen.getByText(t.pending)).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: t.approve }))
  await screen.findByText(t.approved)
  expect(fetchMock.mock.calls[0]?.[0]).toContain('/groups/automatic/role-delegation')
  expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1].body))).toEqual({
    roles: ['catalog:read'],
    expression: '',
  })
  fireEvent.click(screen.getByRole('button', { name: t.revoke }))
  await waitFor(() => expect(screen.getByText(t.pending)).toBeInTheDocument())
  expect(JSON.parse(String(fetchMock.mock.calls[1]?.[1].body))).toEqual({
    roles: [],
    expression: '',
  })
})

//spec:covers REQ-IDMANAGEMENT-093: 古い式への承認を現在の承認として表示せず、保存前の変更を承認させない。
it('shows stale approval and disables approval while editing roles', async () => {
  await renderWithRouter(
    <GroupRoleDelegationPanel
      csrfToken="csrf"
      group={{
        ...group,
        role_delegation: {
          roles: group.roles,
          expression: 'old expression',
          approved_by: 'admin',
          approved_at: group.created_at,
        },
      }}
      unsavedChanges={true}
    />,
  )
  expect(screen.getByText(t.stale)).toBeInTheDocument()
  expect(screen.getByRole('button', { name: t.approve })).toBeDisabled()
  expect(screen.getByRole('button', { name: t.revoke })).not.toBeDisabled()
})

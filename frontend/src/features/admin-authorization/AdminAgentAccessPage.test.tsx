import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminAgent, McpResourceServer, ResourceTypeDefinition } from '../../types'
import { AdminAgentAccessPage } from './AdminAgentAccessPage'

const originalFetch = globalThis.fetch
afterEach(() => {
  globalThis.fetch = originalFetch
})
const agent: AdminAgent = {
  id: 'a1',
  tenant_id: 't',
  name: 'Assistant',
  owner_user_id: 'u1',
  kind: 'autonomous',
  status: 'active',
  client_ids: [],
  created_at: '',
}
const server: McpResourceServer = {
  id: 's1',
  tenant_id: 't',
  name: 'Documents',
  resource: 'https://docs.example',
  scopes: ['docs:read'],
  state: 'Active',
  created_at: '',
  updated_at: '',
  access_rules: [
    { resource_type: 'document', relation: 'viewer', scopes: ['docs:read'] },
    { resource_type: 'document', relation: 'reader', scopes: ['docs:read'] },
  ],
}
const types: ResourceTypeDefinition[] = [
  {
    name: 'document',
    relations: [
      { name: 'viewer', rewrites: [{ kind: 'direct', direct_subject_types: ['agent'] }] },
      { name: 'reader', rewrites: [{ kind: 'computed_userset', computed_relation: 'viewer' }] },
    ],
  },
]
const props = {
  csrfToken: 'csrf',
  agents: [agent],
  resourceServers: [server],
  resourceTypes: types,
  tuples: [],
  mode: 'create' as const,
}

//spec:covers REQ-AUTHORIZATION-014: 対象と Agent の選択が一件の保存に反映され、計算された関係を直接付与しない。
it('grants only the selected resource and agent', async () => {
  const requests: unknown[] = []
  globalThis.fetch = mock(async (_input: RequestInfo | URL, init?: RequestInit) => {
    requests.push(JSON.parse(String(init?.body)))
    return Response.json({ written_count: 1, deleted_count: 0, consistency: 'c' })
  }) as unknown as typeof fetch
  const onGranted = mock(() => {})
  await renderWithRouter(
    <AdminAgentAccessPage {...props} selectedAgent="origin-agent" onGranted={onGranted} />,
  )
  fireEvent.change(screen.getByLabelText('MCP resource server'), { target: { value: 's1' } })
  const permission = screen.getByLabelText('Permission') as HTMLSelectElement
  expect([...permission.options].map((option) => option.value)).toEqual(['', 'document:viewer'])
  fireEvent.change(permission, { target: { value: 'document:viewer' } })
  fireEvent.change(screen.getByLabelText('Resource ID'), { target: { value: 'handbook' } })
  fireEvent.change(screen.getByLabelText('Agent'), { target: { value: 'a1' } })
  fireEvent.click(screen.getByRole('button', { name: 'Grant access' }))
  await waitFor(() => expect(onGranted).toHaveBeenCalled())
  expect(onGranted).toHaveBeenCalledWith({ server: 's1', agent: 'a1' })
  expect(requests).toEqual([
    {
      writes: [
        {
          resource_type: 'document',
          resource_id: 'handbook',
          relation: 'viewer',
          subject_type: 'agent',
          subject_id: 'a1',
        },
      ],
    },
  ])
})

//spec:covers REQ-AUTHORIZATION-014: 取消は選択した一件だけを削除する。
it('revokes only the selected assignment', async () => {
  const tuple = {
    resource_type: 'document',
    resource_id: 'handbook',
    relation: 'viewer',
    subject_type: 'agent',
    subject_id: 'a1',
  }
  const requests: unknown[] = []
  globalThis.fetch = mock(async (_input: RequestInfo | URL, init?: RequestInit) => {
    requests.push(JSON.parse(String(init?.body)))
    return Response.json({ written_count: 0, deleted_count: 1, consistency: 'c' })
  }) as unknown as typeof fetch
  await renderWithRouter(<AdminAgentAccessPage {...props} mode="list" tuples={[tuple]} />)
  fireEvent.click(screen.getByRole('button', { name: 'Revoke' }))
  fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Revoke' }))
  await waitFor(() =>
    expect(screen.getByText('No matching direct assignments.')).toBeInTheDocument(),
  )
  expect(requests).toEqual([{ deletes: [tuple] }])
})

//spec:covers REQ-AUTHORIZATION-014: 保存の拒否で入力を失わず、失敗を表示し、成功へ遷移しない。
it('preserves the selected assignment after a failed grant', async () => {
  globalThis.fetch = mock(async () =>
    Response.json({ message: 'Rejected' }, { status: 422 }),
  ) as unknown as typeof fetch
  const onGranted = mock(() => {})
  await renderWithRouter(<AdminAgentAccessPage {...props} onGranted={onGranted} />)
  fireEvent.change(screen.getByLabelText('MCP resource server'), { target: { value: 's1' } })
  fireEvent.change(screen.getByLabelText('Permission'), { target: { value: 'document:viewer' } })
  fireEvent.change(screen.getByLabelText('Resource ID'), { target: { value: 'handbook' } })
  fireEvent.change(screen.getByLabelText('Agent'), { target: { value: 'a1' } })
  fireEvent.click(screen.getByRole('button', { name: 'Grant access' }))
  await screen.findByRole('alert')
  expect((screen.getByLabelText('Resource ID') as HTMLInputElement).value).toBe('handbook')
  expect((screen.getByLabelText('Agent') as HTMLSelectElement).value).toBe('a1')
  expect(onGranted).not.toHaveBeenCalled()
})

//spec:covers REQ-AUTHORIZATION-014: 無効な候補とワイルドカードのみの直接規則は通常付与に使わない。
it('excludes disabled principals and wildcard-only permissions', async () => {
  await renderWithRouter(
    <AdminAgentAccessPage
      {...props}
      agents={[agent, { ...agent, id: 'disabled', name: 'Disabled', status: 'disabled' }]}
      resourceServers={[server, { ...server, id: 'disabled', name: 'Disabled', state: 'Disabled' }]}
      resourceTypes={[
        {
          name: 'document',
          relations: [
            { name: 'viewer', rewrites: [{ kind: 'direct', direct_subject_types: ['agent:*'] }] },
          ],
        },
      ]}
    />,
  )
  fireEvent.change(screen.getByLabelText('MCP resource server'), { target: { value: 's1' } })
  expect(
    [...(screen.getByLabelText('MCP resource server') as HTMLSelectElement).options].map(
      (option) => option.value,
    ),
  ).toEqual(['', 's1'])
  expect(
    [...(screen.getByLabelText('Agent') as HTMLSelectElement).options].map(
      (option) => option.value,
    ),
  ).toEqual(['', 'a1'])
  expect(
    [...(screen.getByLabelText('Permission') as HTMLSelectElement).options].map(
      (option) => option.value,
    ),
  ).toEqual([''])
  expect((screen.getByRole('button', { name: 'Grant access' }) as HTMLButtonElement).disabled).toBe(
    true,
  )
})

//spec:covers REQ-AUTHORIZATION-014: 取得上限を明示し、表示した付与以外が存在し得ることを伝える。
it('warns when the assignment retrieval reaches its limit', async () => {
  const tuples = Array.from({ length: 1000 }, (_, index) => ({
    resource_type: 'document',
    resource_id: `d${index}`,
    relation: 'viewer',
    subject_type: 'agent',
    subject_id: 'a1',
  }))
  await renderWithRouter(<AdminAgentAccessPage {...props} mode="create" tuples={tuples} />)
  expect(screen.getByRole('alert').textContent).toContain('retrieval limit')
})

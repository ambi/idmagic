import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminAgent, McpResourceServer } from '../../types'
import { AdminAgentResourcesPage } from './AdminAgentResourcesPage'

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
  client_ids: ['auth-client'],
  created_at: '',
}
const server: McpResourceServer = {
  id: 's1',
  tenant_id: 't',
  name: 'Documents',
  resource: 'https://docs.example',
  scopes: ['read', 'write'],
  state: 'Active',
  created_at: '',
  updated_at: '',
  access_rules: [
    {
      resource_type: 'idmagic_mcp_operation',
      relation: 'invoke',
      scopes: ['read'],
      tool: {
        name: 'read_document',
        display_name: 'Read documents',
        assignment_resource_id: 'operation-read-ref',
      },
    },
    {
      resource_type: 'idmagic_mcp_operation',
      relation: 'invoke',
      scopes: ['write'],
      tool: {
        name: 'delete_document',
        display_name: 'Delete documents',
        assignment_resource_id: 'operation-delete-ref',
      },
    },
  ],
}
const tuple = {
  resource_type: 'idmagic_mcp_operation',
  resource_id: 'operation-delete-ref',
  relation: 'invoke',
  subject_type: 'agent',
  subject_id: 'a1',
}

//spec:covers REQ-AUTHORIZATION-017: 接続先と操作の選択だけで、追加と取消を一度で保存し、認証クライアントを変更しない。
it('saves operation assignments in one request without changing clients', async () => {
  const requests: Array<{ url: string; body: unknown }> = []
  globalThis.fetch = mock(async (input: RequestInfo | URL, init?: RequestInit) => {
    requests.push({ url: String(input), body: JSON.parse(String(init?.body)) })
    return Response.json({ written_count: 1, deleted_count: 1, consistency: 'c' })
  }) as unknown as typeof fetch
  const onSaved = mock(() => {})
  await renderWithRouter(
    <AdminAgentResourcesPage
      csrfToken="csrf"
      agent={agent}
      resourceServers={[server]}
      tuples={[tuple]}
      edit
      onSaved={onSaved}
    />,
  )
  fireEvent.click(screen.getByRole('checkbox', { name: /Read documents/ }))
  fireEvent.click(screen.getByRole('checkbox', { name: /Delete documents/ }))
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1))
  expect(requests).toEqual([
    {
      url: '/api/admin/v1/authorization/relation-tuples',
      body: { writes: [{ ...tuple, resource_id: 'operation-read-ref' }], deletes: [tuple] },
    },
  ])
})

//spec:covers REQ-AUTHORIZATION-017: 保存失敗で入力を保持し、取得上限では不完全な取消を送らない。
it('keeps the selected operation when saving fails', async () => {
  globalThis.fetch = mock(async () =>
    Response.json({ code: 'failed' }, { status: 500 }),
  ) as unknown as typeof fetch
  await renderWithRouter(
    <AdminAgentResourcesPage
      csrfToken="csrf"
      agent={agent}
      resourceServers={[server]}
      tuples={[]}
      edit
    />,
  )
  fireEvent.click(screen.getByRole('checkbox', { name: /Read documents/ }))
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(screen.getByRole('alert').textContent).toContain('Could not save'))
  expect(
    (screen.getByRole('checkbox', { name: /Read documents/ }) as HTMLInputElement).checked,
  ).toBe(true)
})

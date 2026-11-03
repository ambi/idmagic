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

const scopedServer: McpResourceServer = {
  ...server,
  access_rules: [
    {
      resource_type: 'idmagic_mcp_operation',
      relation: 'invoke',
      scopes: ['read'],
      tool: {
        name: 'read_document',
        display_name: 'Read documents',
        resource_id_argument: 'document_id',
        assignment_resource_id: 's1/op-read',
      },
    },
  ],
}

function recordWrites() {
  const bodies: Array<{ writes: unknown[]; deletes: unknown[] }> = []
  globalThis.fetch = mock(async (_input: RequestInfo | URL, init?: RequestInit) => {
    bodies.push(JSON.parse(String(init?.body)))
    return Response.json({ written_count: 1, deleted_count: 0, consistency: 'c' })
  }) as unknown as typeof fetch
  return bodies
}

const range = (resourceID: string, relation = 'invoke') => ({
  resource_type: 'idmagic_mcp_operation',
  resource_id: resourceID,
  relation,
  subject_type: 'agent',
  subject_id: 'a1',
})

//spec:covers REQ-AUTHORIZATION-017: 対象ごとの権限を持つ操作は範囲を明示して選ばせ、操作と指定した対象の割当を一つの差分で保存する。範囲を選ぶまで対象は許可しない。
it('requires an explicit target range for target-scoped operations', async () => {
  const bodies = recordWrites()
  const onSaved = mock(() => {})
  await renderWithRouter(
    <AdminAgentResourcesPage
      csrfToken="csrf"
      agent={agent}
      resourceServers={[scopedServer]}
      tuples={[]}
      edit
      onSaved={onSaved}
    />,
  )
  fireEvent.click(screen.getByRole('checkbox', { name: /Read documents/ }))
  expect(screen.getByText('No target is allowed until you choose a range.')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('radio', { name: 'Only the listed targets' }))
  fireEvent.change(screen.getByLabelText('Target IDs (one per line)'), {
    target: { value: 'handbook\n\nhandbook\npolicy' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))

  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(bodies).toEqual([
    {
      writes: [range('s1/op-read'), range('s1/op-read/handbook'), range('s1/op-read/policy')],
      deletes: [],
    },
  ])
})

//spec:covers REQ-AUTHORIZATION-017: 指定した対象からすべての対象へ広げる保存は、古い対象を消して全対象の範囲を書き、関係の ID に使えない対象は保存しない。
it('replaces listed targets when widening to all targets and rejects unusable IDs', async () => {
  const bodies = recordWrites()
  const onSaved = mock(() => {})
  await renderWithRouter(
    <AdminAgentResourcesPage
      csrfToken="csrf"
      agent={agent}
      resourceServers={[scopedServer]}
      tuples={[range('s1/op-read'), range('s1/op-read/handbook')]}
      edit
      onSaved={onSaved}
    />,
  )
  expect(screen.getByLabelText('Target IDs (one per line)')).toHaveValue('handbook')
  fireEvent.change(screen.getByLabelText('Target IDs (one per line)'), {
    target: { value: 'handbook\nbad:id' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(
    await screen.findByText('These target IDs contain ":" or "#" and cannot be saved: bad:id'),
  ).toBeInTheDocument()
  expect(bodies).toEqual([])

  fireEvent.click(screen.getByRole('radio', { name: 'All targets' }))
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(bodies).toEqual([
    { writes: [range('s1/op-read', 'invoke_any_target')], deletes: [range('s1/op-read/handbook')] },
  ])
})

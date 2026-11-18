import { screen } from '@testing-library/react'
import { expect, it } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminAgent } from '../../types'
import { AdminAgentAccessPage } from './AdminAgentAccessPage'

const agent: AdminAgent = {
  id: 'agent-1',
  tenant_id: 't',
  name: 'Assistant',
  owner_user_id: 'u',
  kind: 'autonomous',
  status: 'active',
  client_ids: [],
  created_at: '',
}

//spec:covers REQ-IDMANAGEMENT-106: 期限切れと撤回、欠落、対象、条件と上限、二層のガードレールを許可判定と区別する。
it('shows access materials without claiming an execution decision', async () => {
  await renderWithRouter(
    <AdminAgentAccessPage
      agent={agent}
      summary={{
        as_of: '2026-11-13T00:00:00Z',
        clients: [{ client_id: 'missing-client', scopes: [], missing: true }],
        delegations: [
          {
            user_id: 'expired-user',
            scopes: ['read'],
            state: 'granted',
            granted_at: '2026-10-01T00:00:00Z',
            expires_at: '2026-11-13T00:00:00Z',
            active: false,
          },
          {
            user_id: 'revoked-user',
            scopes: ['write'],
            state: 'revoked',
            granted_at: '2026-10-01T00:00:00Z',
            expires_at: '2027-01-01T00:00:00Z',
            active: false,
          },
        ],
      }}
      guardrails={{
        agent: { version: 'v', updated_at: '', allowed_resources: [] },
        tenant_default: { version: 'v', updated_at: '', max_token_ttl_seconds: 60 },
      }}
      resourceServers={[
        {
          id: 's',
          tenant_id: 't',
          name: 'Documents',
          resource: 'https://docs.example',
          scopes: ['read'],
          state: 'Active',
          created_at: '',
          updated_at: '',
          access_rules: [
            {
              resource_type: 'idmagic_mcp_operation',
              relation: 'invoke',
              scopes: ['read'],
              tool: {
                name: 'read',
                display_name: 'Read documents',
                assignment_resource_id: 'op',
                resource_id_argument: 'document',
              },
              condition: {
                deny_when: 'request.hour > 20',
                require_approval_when: 'request.risky',
                approval_detail_type: 'approval',
              },
              limits: [{ by: ['agent'], max_requests: 10, window_seconds: 60 }],
            },
          ],
        },
      ]}
      tuples={[
        {
          resource_type: 'idmagic_mcp_operation',
          resource_id: 'op',
          relation: 'invoke',
          subject_type: 'agent',
          subject_id: agent.id,
        },
        {
          resource_type: 'idmagic_mcp_operation',
          resource_id: 'op/doc-7',
          relation: 'invoke',
          subject_type: 'agent',
          subject_id: agent.id,
        },
      ]}
      catalogs={{
        s: { tools: [], complete: true, received_at: '2026-11-13T00:00:00Z' },
      }}
    />,
  )
  for (const value of [
    'Bound client is missing',
    'Expired',
    'Revoked',
    'doc-7',
    'request.hour > 20',
    'request.risky',
    '10 requests per 60 seconds by Agent',
    'None allowed',
    '60',
  ]) {
    expect(screen.getByText(value)).toBeTruthy()
  }
  expect(screen.getByText(/does not guarantee permission/)).toBeTruthy()
  expect(document.querySelector('form')).toBeNull()
})

//spec:covers REQ-IDMANAGEMENT-106: 取得上限を不完全な材料として表示し、空の各材料を区別する。
it('warns when assignments may be incomplete and explains empty materials', async () => {
  await renderWithRouter(
    <AdminAgentAccessPage
      agent={agent}
      summary={{ as_of: '2026-11-13T00:00:00Z', clients: [], delegations: [] }}
      guardrails={{}}
      resourceServers={[]}
      tuples={Array.from({ length: 1000 }, (_, index) => ({
        resource_type: 'idmagic_mcp_operation',
        resource_id: `unrelated-${index}`,
        relation: 'invoke',
        subject_type: 'agent',
        subject_id: 'other-agent',
      }))}
      catalogs={{}}
    />,
  )
  expect(screen.getByRole('alert').textContent).toContain('may be incomplete')
  expect(screen.getByText('No authentication clients are bound.')).toBeTruthy()
  expect(screen.getByText('No delegation grants.')).toBeTruthy()
  expect(screen.getByText('No connection rules are registered.')).toBeTruthy()
  expect(screen.getAllByText('Not configured')).toHaveLength(2)
})

const toolRule = (name: string, assignmentID: string, scopes: string[]) => ({
  resource_type: 'idmagic_mcp_operation',
  relation: 'invoke',
  scopes,
  tool: { name, display_name: name, assignment_resource_id: assignmentID },
})

//spec:covers REQ-OAUTH2-094: 接続先ごとに四項目の状態と、不足している設定とその修正先を示す。
it('shows per-server check states with guidance linking to where each gap is fixed', async () => {
  await renderWithRouter(
    <AdminAgentAccessPage
      agent={agent}
      summary={{
        as_of: '2026-11-13T00:00:00Z',
        clients: [{ client_id: 'bound-client', scopes: ['read'], missing: false }],
        delegations: [],
      }}
      guardrails={{
        agent: {
          version: 'v',
          updated_at: '',
          allowed_resources: ['https://ok.example'],
        },
        tenant_default: {
          version: 'v',
          updated_at: '',
          allowed_resources: ['https://ok.example'],
        },
      }}
      resourceServers={[
        {
          id: 'srv-ok',
          tenant_id: 't',
          name: 'Ready connection',
          resource: 'https://ok.example',
          scopes: ['read'],
          client_id: 'eval-client',
          state: 'Active',
          created_at: '',
          updated_at: '',
          access_rules: [toolRule('read', 'op-ok', ['read'])],
        },
        {
          id: 'srv-gap',
          tenant_id: 't',
          name: 'Incomplete connection',
          resource: 'https://gap.example',
          scopes: ['admin'],
          state: 'Disabled',
          created_at: '',
          updated_at: '',
          access_rules: [toolRule('admin', 'op-gap', ['admin'])],
        },
      ]}
      tuples={[
        {
          resource_type: 'idmagic_mcp_operation',
          resource_id: 'op-ok',
          relation: 'invoke',
          subject_type: 'agent',
          subject_id: agent.id,
        },
      ]}
      catalogs={{
        'srv-ok': { tools: [], complete: true, received_at: '2026-11-13T00:00:00Z' },
        'srv-gap': { tools: [], complete: false },
      }}
    />,
  )
  expect(screen.getAllByText('No issues')).toHaveLength(4)
  expect(screen.getAllByText('Missing settings')).toHaveLength(4)
  const serverEdit = '/admin/mcp-resource-servers/srv-gap/edit'
  for (const [guidance, href] of [
    ['The registration is disabled.', serverEdit],
    ['No evaluation client is bound.', serverEdit],
    ['No operations of this connection are assigned.', '/admin/agents/agent-1/resources'],
    [
      'The Agent guardrail denies issuance for this connection.',
      '/admin/agents/agent-1/guardrail/edit',
    ],
    ['No catalog has arrived from the Gateway.', serverEdit],
  ] as const) {
    expect(screen.getByRole('link', { name: new RegExp(guidance) })).toHaveAttribute('href', href)
  }
  // テナントのデフォルトは管理コンソールに編集画面がないため、管理 API を案内してリンクにしない。
  const tenantGuidance = screen.getByText(/The tenant default guardrail denies issuance/)
  expect(tenantGuidance.textContent).toContain('PUT /api/admin/v1/agent-guardrail-default')
  expect(tenantGuidance.closest('a')).toBeNull()
  expect(screen.getByText(/does not connect to MCP servers or run business tools/)).toBeTruthy()
})

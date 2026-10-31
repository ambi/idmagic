import { screen } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { CrossAppAccessIssuePolicy, CrossAppAccessRedeemPolicy } from '../../types'
import { adminCrossAppAccessDictionary } from './AdminCrossAppAccessPage.i18n'
import { AdminCrossAppAccessPage } from './AdminCrossAppAccessPage'

const issuePolicy: CrossAppAccessIssuePolicy = {
  tenant_id: 'tenant-1',
  id: 'issue-1',
  client_id: 'agent-client',
  audience: 'https://auth.example.com/',
  relying_party_client_id: 'chat-rp',
  target_client_id: 'target',
  resources: ['https://api.example.com/'],
  scopes: ['chat.read'],
  authorization_detail_types: [],
  release_email: false,
  created_at: '2026-10-31T00:00:00Z',
}

const redeemPolicy: CrossAppAccessRedeemPolicy = {
  tenant_id: 'tenant-1',
  id: 'redeem-1',
  provider_id: 'corporate-idp',
  client_id: 'mcp-client',
  scopes: ['files.read'],
  created_at: '2026-10-31T00:00:00Z',
}

describe('AdminCrossAppAccessPage', () => {
  it('lists both directions of policy', async () => {
    await renderWithRouter(
      <AdminCrossAppAccessPage
        csrfToken="csrf"
        issuePolicies={[issuePolicy]}
        redeemPolicies={[redeemPolicy]}
      />,
    )
    expect(screen.getByText('agent-client → https://auth.example.com/')).toBeInTheDocument()
    expect(screen.getByText('corporate-idp → mcp-client')).toBeInTheDocument()
    expect(screen.getByText('files.read')).toBeInTheDocument()
  })

  it('shows the empty notices in Japanese', async () => {
    await renderWithRouter(
      <AdminCrossAppAccessPage csrfToken="csrf" issuePolicies={[]} redeemPolicies={[]} />,
      { locale: 'ja' },
    )
    expect(screen.getByText(adminCrossAppAccessDictionary.ja.emptyIssueNotice)).toBeInTheDocument()
    expect(screen.getByText(adminCrossAppAccessDictionary.ja.emptyRedeemNotice)).toBeInTheDocument()
  })
})

import { describe, expect, it } from 'bun:test'
import { screen } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAgentDelegationsSection } from './AdminAgentDelegationsSection'

describe('AdminAgentDelegationsSection', () => {
  it('lists the delegations without any action to grant or revoke them', async () => {
    await renderWithRouter(
      <AdminAgentDelegationsSection
        delegations={[
          {
            user_id: 'user-1',
            preferred_username: 'taro',
            agent_id: 'agent-1',
            agent_name: 'Order bot',
            scopes: ['read'],
            state: 'revoked',
            granted_at: '2026-01-01T00:00:00Z',
            expires_at: '2027-01-01T00:00:00Z',
          },
        ]}
      />,
      { locale: 'ja' },
    )
    expect(screen.getByText('taro')).toBeInTheDocument()
    expect(screen.getByText('Order bot')).toBeInTheDocument()
    expect(screen.queryByRole('button')).toBeNull()
  })
})

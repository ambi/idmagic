import { describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen } from '@testing-library/react'
import type { AccountApprovalRequest } from '../../api'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { AccountApprovalsPresentation } from './AccountApprovalsPage'

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

const approvalRequest: AccountApprovalRequest = {
  id: 'approval-1',
  client_id: 'client-1',
  client_name: 'Expense Agent',
  agent_name: 'Travel Assistant',
  scopes: ['openid', 'payments.write'],
  authorization_details: [{ type: 'payment_initiation', amount: '5000', currency: 'JPY' }],
  binding_message: 'Trip W-123',
  requested_at: '2026-08-14T00:00:00Z',
  expires_at: '2026-08-14T00:05:00Z',
}

const baseProps = {
  username: 'alice',
  isAdmin: false,
  approvalRequests: [approvalRequest],
  pending: '',
  error: '',
  notice: '',
  onDismissNotice: mock(),
  onDecision: mock(),
}

describe('AccountApprovalsPresentation', () => {
  it('shows all information needed to identify the requested action', async () => {
    await renderWithRouter(<AccountApprovalsPresentation {...baseProps} />)

    expect(screen.getByText('Expense Agent')).toBeInTheDocument()
    expect(screen.getByText(/Travel Assistant/)).toBeInTheDocument()
    expect(screen.getByText('payments.write')).toBeInTheDocument()
    expect(screen.getByText('Trip W-123')).toBeInTheDocument()
    expect(screen.getByText(/payment_initiation/)).toBeInTheDocument()
  })

  it('passes an explicit approve or deny decision', async () => {
    const onDecision = mock()
    await renderWithRouter(<AccountApprovalsPresentation {...baseProps} onDecision={onDecision} />)

    fireEvent.click(screen.getByRole('button', { name: '承認する' }))
    fireEvent.click(screen.getByRole('button', { name: '拒否する' }))

    expect(onDecision).toHaveBeenCalledWith(approvalRequest, 'approve')
    expect(onDecision).toHaveBeenCalledWith(approvalRequest, 'deny')
  })

  it('shows an empty state without exposing completed requests', async () => {
    await renderWithRouter(<AccountApprovalsPresentation {...baseProps} approvalRequests={[]} />)

    expect(screen.getByText('保留中の承認リクエストはありません。')).toBeInTheDocument()
  })
})

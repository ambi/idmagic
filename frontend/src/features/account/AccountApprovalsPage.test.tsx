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
  //spec:covers CIBA-CORE-BINDING-MESSAGE: binding_message を承認画面が実際に描き、クライアント、
  // 要求スコープ、authorization_details と併せて示すことを固定する。API が 4 つを
  // 返すことは backend/shared/http/server_http/non_interactive_grant_standards_test.go が
  // 同じ id で読む。運ぶことと見せることは別なので、両方を読まないと片方が落ちる。
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

  //spec:covers REQ-OAUTH2-043: 承認の URL の承認リクエストを先頭に強調し、起票したリソースサーバーを示すこと
  it('puts the request from the approval link first and names the resource server that started it', async () => {
    const started: AccountApprovalRequest = {
      ...approvalRequest,
      id: 'approval-2',
      client_name: 'Generic MCP client',
      agent_name: undefined,
      resource_server_name: 'Payments MCP',
      scopes: [],
      binding_message: undefined,
    }
    await renderWithRouter(
      <AccountApprovalsPresentation
        {...baseProps}
        approvalRequests={[approvalRequest, started]}
        focusRequestId="approval-2"
      />,
    )

    const cards = screen.getAllByRole('article')
    expect(cards[0]).toHaveAttribute('aria-current', 'true')
    expect(cards[0]).toHaveTextContent('Generic MCP client')
    expect(cards[0]).toHaveTextContent('Payments MCP')
    expect(cards[1]).not.toHaveAttribute('aria-current')
  })

  //spec:covers REQ-OAUTH2-043: 本人宛の保留中にない承認の URL では、判断できる承認リクエストがないことを示すこと
  it('says when the request from the approval link cannot be decided', async () => {
    await renderWithRouter(
      <AccountApprovalsPresentation {...baseProps} focusRequestId="someone-elses" />,
    )

    expect(
      screen.getByText(
        'このリンクの承認リクエストは、すでに判断されたか、期限が切れたか、あなた宛てではありません。',
      ),
    ).toBeInTheDocument()
    expect(screen.getByText('Expense Agent')).toBeInTheDocument()
  })
})

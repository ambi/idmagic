import { createFileRoute } from '@tanstack/react-router'
import { listMyApprovalRequests } from '../../api/account'
import { AccountApprovalsPage } from '../../features/account/AccountApprovalsPage'
import { hasAdminRole, requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/account/approvals')({
  // request は MCP ゲートウェイが示す承認の URL が名指す承認リクエストである (REQ-OAUTH2-043)。
  validateSearch: (search: Record<string, unknown>) => ({
    request:
      typeof search.request === 'string' && search.request !== '' ? search.request : undefined,
  }),
  loader: async ({ location }) => {
    const account = await requirePortalAccount('account', location.pathname, location.searchStr)
    const approvalRequests = await listMyApprovalRequests()
    return {
      csrfToken: account.csrf_token,
      username: account.preferred_username ?? 'account',
      approvalRequests,
      isAdmin: hasAdminRole(account.roles),
    }
  },
  component: AccountApprovalsRoute,
})

function AccountApprovalsRoute() {
  const data = Route.useLoaderData()
  const { request } = Route.useSearch()
  return (
    <PageMarker kind="account-approvals">
      <AccountApprovalsPage {...data} focusRequestId={request} />
    </PageMarker>
  )
}

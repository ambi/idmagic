import { createFileRoute } from '@tanstack/react-router'
import { listMyApprovalRequests } from '../../api/account'
import { AccountApprovalsPage } from '../../features/account/AccountApprovalsPage'
import { hasAdminRole, requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/account/approvals')({
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
  return (
    <PageMarker kind="account-approvals">
      <AccountApprovalsPage {...data} />
    </PageMarker>
  )
}

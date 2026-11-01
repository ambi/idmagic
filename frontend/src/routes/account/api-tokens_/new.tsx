import { createFileRoute } from '@tanstack/react-router'
import { AccountApiTokensPage } from '../../../features/account/AccountApiTokensPage'
import { hasAdminRole, requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/account/api-tokens_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('account', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      username: account.preferred_username ?? 'account',
      isAdmin: hasAdminRole(account.roles),
    }
  },
  component: () => (
    <PageMarker kind="account-api-token-create">
      <AccountApiTokensPage {...Route.useLoaderData()} create />
    </PageMarker>
  ),
})

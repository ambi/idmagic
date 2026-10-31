import { createFileRoute } from '@tanstack/react-router'
import { listMyTokenVaultConnections } from '../../api/account'
import { AccountConnectionsPage } from '../../features/account/AccountConnectionsPage'
import { hasAdminRole, requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/account/connections')({
  validateSearch: (search: Record<string, unknown>) => ({
    result: typeof search.result === 'string' ? search.result : undefined,
  }),
  loader: async ({ location }) => {
    const account = await requirePortalAccount('account', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      username: account.preferred_username ?? 'account',
      connections: await listMyTokenVaultConnections(),
      isAdmin: hasAdminRole(account.roles),
    }
  },
  component: AccountConnectionsRoute,
})

function AccountConnectionsRoute() {
  const data = Route.useLoaderData()
  const { result } = Route.useSearch()
  return (
    <PageMarker kind="account-connections">
      <AccountConnectionsPage {...data} result={result} />
    </PageMarker>
  )
}

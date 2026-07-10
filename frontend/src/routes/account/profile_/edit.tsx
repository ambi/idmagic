import { createFileRoute } from '@tanstack/react-router'
import { request } from '../../../api/core'
import { AccountProfileEditPage } from '../../../features/account/AccountProfilePage'
import type { AccountProfile } from '../../../types'
import { hasAdminRole, requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/account/profile_/edit')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('account', location.pathname, location.searchStr)
    const profile = await request<AccountProfile>('/api/account/profile')
    return {
      csrfToken: account.csrf_token,
      profile,
      isAdmin: hasAdminRole(account.roles),
    }
  },
  component: AccountProfileEditRoute,
})

function AccountProfileEditRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="account-profile-edit">
      <AccountProfileEditPage {...data} />
    </PageMarker>
  )
}

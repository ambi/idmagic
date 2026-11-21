import { createFileRoute } from '@tanstack/react-router'
import { listMyAccessReviews } from '../../api/accessReviews'
import { MyAccessReviewsPage } from '../../features/access-reviews/AccessReviewPages'
import { hasAdminRole, requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/account/access-reviews')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('account', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      username: account.preferred_username ?? 'account',
      isAdmin: hasAdminRole(account.roles),
      reviews: await listMyAccessReviews(),
    }
  },
  component: () => (
    <PageMarker kind="account-access-reviews">
      <MyAccessReviewsPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})

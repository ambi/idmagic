import { createFileRoute } from '@tanstack/react-router'
import { listAccessReviewPlans } from '../../api/accessReviews'
import { AccessReviewPlansPage } from '../../features/access-reviews/AccessReviewPages'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/access-reviews')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      plans: await listAccessReviewPlans(),
    }
  },
  component: () => (
    <PageMarker kind="admin-access-reviews">
      <AccessReviewPlansPage {...Route.useLoaderData()} />
    </PageMarker>
  ),
})

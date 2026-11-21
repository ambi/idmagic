import { createFileRoute } from '@tanstack/react-router'
import { listAccessReviews } from '../../../api/accessReviews'
import { AccessReviewPlanPage } from '../../../features/access-reviews/AccessReviewPages'
import { requirePortalAccount } from '../../-guards'
import { accessReviewPlan, accessReviewCatalogs } from '../../-access-reviews'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/access-reviews_/$planId/')({
  validateSearch: (search: Record<string, unknown>) => ({ saved: search.saved === '1' }),
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [plan, reviews, catalogs] = await Promise.all([
      accessReviewPlan(params.planId),
      listAccessReviews(params.planId),
      accessReviewCatalogs(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      plan,
      reviews,
      catalogs,
    }
  },
  component: () => (
    <PageMarker kind="admin-access-review-detail">
      <AccessReviewPlanPage {...Route.useLoaderData()} saved={Route.useSearch().saved} />
    </PageMarker>
  ),
})

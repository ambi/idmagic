import { createFileRoute } from '@tanstack/react-router'
import { AccessReviewPlanEditor } from '../../../features/access-reviews/AccessReviewPages'
import { requirePortalAccount } from '../../-guards'
import { accessReviewCatalogs, accessReviewPlan } from '../../-access-reviews'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/access-reviews_/$planId/edit')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [initial, catalogs] = await Promise.all([
      accessReviewPlan(params.planId),
      accessReviewCatalogs(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      initial,
      ...catalogs,
    }
  },
  component: () => (
    <PageMarker kind="admin-access-review-edit">
      <AccessReviewPlanEditor {...Route.useLoaderData()} />
    </PageMarker>
  ),
})

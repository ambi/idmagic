import { createFileRoute } from '@tanstack/react-router'
import { AccessReviewPlanEditor } from '../../../features/access-reviews/AccessReviewPages'
import { requirePortalAccount } from '../../-guards'
import { accessReviewCatalogs } from '../../-access-reviews'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/access-reviews_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      ...(await accessReviewCatalogs()),
    }
  },
  component: () => (
    <PageMarker kind="admin-access-review-create">
      <AccessReviewPlanEditor {...Route.useLoaderData()} />
    </PageMarker>
  ),
})

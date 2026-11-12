import { createFileRoute } from '@tanstack/react-router'
import {
  loadRolloverSnapshot,
  XmlFederationKeyPage,
} from '../../../features/admin-keys/XmlFederationKeyPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/keys_/xml-federation')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      snapshot: await loadRolloverSnapshot(),
    }
  },
  component: XmlFederationKeyRoute,
})

function XmlFederationKeyRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-xml-federation-key">
      <XmlFederationKeyPage {...data} />
    </PageMarker>
  )
}

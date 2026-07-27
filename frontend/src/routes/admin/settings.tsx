import { createFileRoute } from '@tanstack/react-router'
import { request } from '../../api/core'
import {
  getAdminIntegrationEndpoints,
  listIdentityProviderConnections,
  listSamlIDPProfiles,
} from '../../api/admin'
import { AdminSettingsPage } from '../../features/admin-settings/AdminSettingsPage'
import type { AdminSettings } from '../../types'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/settings')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [settings, integrationEndpoints, samlIDPProfiles, identityProviders] = await Promise.all([
      request<AdminSettings>('/api/admin/settings'),
      getAdminIntegrationEndpoints(),
      listSamlIDPProfiles(),
      listIdentityProviderConnections(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      actorRoles: account.roles ?? [],
      actorRealm: account.realm ?? '',
      settings,
      integrationEndpoints,
      samlIDPProfiles,
      identityProviders,
    }
  },
  component: AdminSettingsRoute,
})

function AdminSettingsRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-settings">
      <AdminSettingsPage {...data} />
    </PageMarker>
  )
}

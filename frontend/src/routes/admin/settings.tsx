import { createFileRoute, redirect } from '@tanstack/react-router'
import { request } from '../../api/core'
import { getAdminIntegrationEndpoints, listSamlIDPProfiles } from '../../api/admin'
import { AdminSettingsPage } from '../../features/admin-settings/AdminSettingsPage'
import type { AdminSettings } from '../../types'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/settings')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    if (new URLSearchParams(location.searchStr).get('tab') === 'api-tokens')
      throw redirect({ to: '/admin/api-tokens' })
    const [settings, integrationEndpoints, samlIDPProfiles] = await Promise.all([
      request<AdminSettings>('/api/admin/v1/settings'),
      getAdminIntegrationEndpoints(),
      listSamlIDPProfiles(),
    ])
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      actorRoles: account.roles ?? [],
      actorRealm: account.realm ?? '',
      settings,
      integrationEndpoints,
      samlIDPProfiles,
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

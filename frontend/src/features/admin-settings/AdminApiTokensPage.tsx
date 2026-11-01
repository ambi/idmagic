import { AdminShell } from '../../components/AdminShell'
import { useDictionary } from '../../lib/i18n'
import type { AdminIntegrationEndpointCatalog } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { ApiTokensTab } from './ApiTokensTab'

export function AdminApiTokensPage({
  csrfToken,
  actorUsername,
  integrationEndpoints,
  create = false,
}: {
  csrfToken: string
  actorUsername?: string
  integrationEndpoints: AdminIntegrationEndpointCatalog
  create?: boolean
}) {
  const t = useDictionary(adminSettingsDictionary)
  return (
    <AdminShell
      active="api-tokens"
      actorUsername={actorUsername}
      title={create ? t.newTokenHeading : t.apiTokensHeading}
    >
      <ApiTokensTab
        csrfToken={csrfToken}
        integrationEndpoints={integrationEndpoints}
        screen={create ? 'create' : 'list'}
      />
    </AdminShell>
  )
}

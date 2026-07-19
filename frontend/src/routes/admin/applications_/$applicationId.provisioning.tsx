import { createFileRoute } from '@tanstack/react-router'
import {
  AuthenticationAPIError,
  getAdminApplication,
  getAdminApplicationProvisioning,
} from '../../../api'
import { AdminApplicationProvisioningPage } from '../../../features/admin-applications/AdminApplicationProvisioningPage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'
import type { ProvisioningConnection } from '../../../types'

export const Route = createFileRoute('/admin/applications_/$applicationId/provisioning')({
  loader: async ({ location, params }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const detail = await getAdminApplication(params.applicationId)
    let connection: ProvisioningConnection | null = null
    try {
      connection = await getAdminApplicationProvisioning(params.applicationId)
    } catch (cause) {
      if (!(cause instanceof AuthenticationAPIError && cause.code === 'provisioning_not_found')) {
        throw cause
      }
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      applicationID: params.applicationId,
      applicationName: detail.application.name,
      initialConnection: connection,
    }
  },
  component: AdminApplicationProvisioningRoute,
})

function AdminApplicationProvisioningRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-application-provisioning">
      <AdminApplicationProvisioningPage {...data} />
    </PageMarker>
  )
}

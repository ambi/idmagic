import { createFileRoute } from '@tanstack/react-router'
import { getAppSignInPolicy, getTenantDefaultSignInPolicy, listAdminApplications } from '../../api'
import {
  AdminSignInPolicyPage,
  type SignInPolicyAppRow,
} from '../../features/admin-sign-in-policy/AdminSignInPolicyPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/admin/sign-in-policy')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const [policyView, applications] = await Promise.all([
      getTenantDefaultSignInPolicy(),
      listAdminApplications(),
    ])
    // service 種別はフェデレーション経路を持たずサインインポリシー対象外。
    const policyApps = applications.filter((app) => app.kind !== 'service')
    const views = await Promise.all(policyApps.map((app) => getAppSignInPolicy(app.application_id)))
    const apps: SignInPolicyAppRow[] = policyApps.map((app, index) => ({
      application: app,
      view: views[index],
    }))
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      policy: policyView.policy,
      unenrolledUserCount: policyView.unenrolled_user_count,
      apps,
    }
  },
  component: AdminSignInPolicyRoute,
})

function AdminSignInPolicyRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-sign-in-policy">
      <AdminSignInPolicyPage {...data} />
    </PageMarker>
  )
}

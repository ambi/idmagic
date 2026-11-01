import { createFileRoute, useRouter } from '@tanstack/react-router'
import { AdminTokenVaultCreatePage } from '../../../features/admin-token-vault/AdminTokenVaultCreatePage'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'

export const Route = createFileRoute('/admin/token-vault_/new')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    return { csrfToken: account.csrf_token, actorUsername: account.preferred_username }
  },
  component: CreateRoute,
})
function CreateRoute() {
  const router = useRouter()
  return (
    <PageMarker kind="admin-token-vault-create">
      <AdminTokenVaultCreatePage
        {...Route.useLoaderData()}
        onCreated={async () => {
          await router.invalidate()
          await router.navigate({ to: '/admin/token-vault', state: { tokenVaultCreated: true } })
        }}
      />
    </PageMarker>
  )
}

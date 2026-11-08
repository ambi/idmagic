import { updateAgentWorkloadBinding } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { AdminAgent, AgentWorkloadBinding, WorkloadTrustBundle } from '../../types'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { detailURL } from './AdminWorkloadIdentityShared'
import { BindingForm } from './BindingForm'
import { returnToBundle } from './navigation'

export function AdminWorkloadBindingEditPage({
  csrfToken,
  actorUsername,
  trustBundle,
  binding,
  agents,
}: {
  csrfToken: string
  actorUsername?: string
  trustBundle: WorkloadTrustBundle
  binding: AgentWorkloadBinding
  agents: AdminAgent[]
}) {
  const t = useDictionary(adminWorkloadIdentityDictionary)
  const back = detailURL(trustBundle.id)
  return (
    <AdminShell
      active="workload-identity"
      actorUsername={actorUsername}
      title={t.editBinding}
      description={trustBundle.name}
    >
      <Card className="admin-form">
        <BindingForm
          csrfToken={csrfToken}
          trustBundleID={trustBundle.id}
          agents={agents}
          binding={binding}
          saveLabel={t.update}
          onSave={async (input) => {
            await updateAgentWorkloadBinding(csrfToken, binding.id, input)
            returnToBundle(trustBundle.id, t.bindingUpdatedNotice)
          }}
        />
        <div className="px-4 pb-4">
          <Button variant="outline" nativeButton={false} render={<a href={back} />}>
            {t.cancel}
          </Button>
        </div>
      </Card>
    </AdminShell>
  )
}

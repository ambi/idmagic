import { IconArrowLeft } from '@tabler/icons-react'
import { useState } from 'react'
import { AdminShell } from '../../components/AdminShell'
import { useDictionary } from '../../lib/i18n'
import { detailURL } from './AdminApplicationsShared'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'
import { ConnectForm } from './AdminApplicationProvisioningConnect'
import { DangerZone } from './AdminApplicationProvisioningDanger'
import { DeliveriesPanel } from './AdminApplicationProvisioningDeliveries'
import { OnDemandAndResyncPanel } from './AdminApplicationProvisioningOnDemand'
import { ConnectionSettingsForm } from './AdminApplicationProvisioningSettings'
import { ConnectionStatusPanel, TestConnectionPanel } from './AdminApplicationProvisioningStatus'
import type { ProvisioningConnection } from '../../types'

export function AdminApplicationProvisioningPage({
  csrfToken,
  actorUsername,
  applicationID,
  applicationName,
  initialConnection,
}: {
  csrfToken: string
  actorUsername?: string
  applicationID: string
  applicationName: string
  initialConnection: ProvisioningConnection | null
}) {
  const t = useDictionary(provisioningDictionary)
  const [connection, setConnection] = useState(initialConnection)

  return (
    <AdminShell
      active="applications"
      actorUsername={actorUsername}
      title={`${applicationName} · ${t.provisioningNavLabel}`}
      description={t.provisioningDescription}
      actions={
        <a
          href={detailURL(applicationID)}
          className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToDetail}
        </a>
      }
    >
      <div className="grid max-w-3xl gap-6">
        {connection === null ? (
          <ConnectForm
            csrfToken={csrfToken}
            applicationID={applicationID}
            onConnected={setConnection}
          />
        ) : (
          <>
            <ConnectionStatusPanel
              csrfToken={csrfToken}
              applicationID={applicationID}
              connection={connection}
              onChanged={setConnection}
            />
            <ConnectionSettingsForm
              csrfToken={csrfToken}
              applicationID={applicationID}
              connection={connection}
              onSaved={setConnection}
            />
            <TestConnectionPanel
              csrfToken={csrfToken}
              applicationID={applicationID}
              connection={connection}
              onChanged={setConnection}
            />
            <OnDemandAndResyncPanel csrfToken={csrfToken} applicationID={applicationID} />
            <DeliveriesPanel csrfToken={csrfToken} applicationID={applicationID} />
            <DangerZone
              csrfToken={csrfToken}
              applicationID={applicationID}
              onDeleted={() => setConnection(null)}
            />
          </>
        )}
      </div>
    </AdminShell>
  )
}

import { IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { deleteAdminApplicationProvisioning } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import { messageOf, SectionTitle } from './AdminApplicationsShared'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'

export function DangerZone({
  csrfToken,
  applicationID,
  onDeleted,
}: {
  csrfToken: string
  applicationID: string
  onDeleted: () => void
}) {
  const t = useDictionary(provisioningDictionary)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function handleDelete() {
    setBusy(true)
    setError('')
    try {
      await deleteAdminApplicationProvisioning(csrfToken, applicationID)
      onDeleted()
    } catch (cause) {
      setError(messageOf(cause, t.deleteConnectionFailedError))
      setBusy(false)
    }
  }

  return (
    <Card className="grid gap-3 border-red-200 p-5">
      <SectionTitle>{t.dangerZoneHeading}</SectionTitle>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {confirmDelete ? (
        <Alert variant="destructive" className="flex flex-wrap items-center justify-between gap-2">
          <span>{t.confirmDeleteConnectionPrompt}</span>
          <div className="flex gap-2">
            <Button variant="outline" disabled={busy} onClick={() => setConfirmDelete(false)}>
              {t.dismissConfirm}
            </Button>
            <Button variant="destructive" disabled={busy} onClick={() => void handleDelete()}>
              <IconTrash size={14} aria-hidden="true" />
              {t.confirmDelete}
            </Button>
          </div>
        </Alert>
      ) : (
        <div>
          <Button type="button" variant="destructive" onClick={() => setConfirmDelete(true)}>
            <IconTrash size={16} aria-hidden="true" />
            {t.deleteConnectionButton}
          </Button>
        </div>
      )}
    </Card>
  )
}

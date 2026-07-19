import { type FormEvent, useState } from 'react'
import { registerAdminApplicationProvisioning } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { messageOf, SectionTitle } from './AdminApplicationsShared'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'
import {
  CredentialFieldsEditor,
  credentialInputFrom,
  emptyCredentialFields,
} from './AdminApplicationProvisioningShared'
import type { ProvisioningAuthMethod, ProvisioningConnection } from '../../types'

export function ConnectForm({
  csrfToken,
  applicationID,
  onConnected,
}: {
  csrfToken: string
  applicationID: string
  onConnected: (conn: ProvisioningConnection) => void
}) {
  const t = useDictionary(provisioningDictionary)
  const [baseURL, setBaseURL] = useState('')
  const [authMethod, setAuthMethod] = useState<ProvisioningAuthMethod>('bearer_token')
  const [fields, setFields] = useState(emptyCredentialFields())
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const conn = await registerAdminApplicationProvisioning(csrfToken, applicationID, {
        base_url: baseURL,
        credential: credentialInputFrom(authMethod, fields),
      })
      onConnected(conn)
    } catch (cause) {
      setError(messageOf(cause, t.connectFailedError))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card className="grid gap-4 p-5">
      <SectionTitle>{t.connectHeading}</SectionTitle>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <form className="grid gap-4" onSubmit={(e) => void submit(e)}>
        <div className="grid gap-1.5">
          <Label>{t.baseUrlFieldLabel}</Label>
          <Input
            required
            placeholder={t.baseUrlPlaceholder}
            value={baseURL}
            onChange={(e) => setBaseURL(e.target.value)}
          />
        </div>
        <CredentialFieldsEditor
          authMethod={authMethod}
          setAuthMethod={setAuthMethod}
          fields={fields}
          setFields={setFields}
        />
        <div>
          <Button type="submit" disabled={busy || baseURL.trim() === ''}>
            {busy ? t.connecting : t.connectButton}
          </Button>
        </div>
      </form>
    </Card>
  )
}

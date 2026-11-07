import { type FormEvent, useState } from 'react'
import { registerAdminApplicationProvisioning } from '../../api'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { formSchema, requiredUrl, useFormValidation } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import { messageOf, SectionTitle } from './AdminApplicationsShared'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'
import {
  CredentialFieldsEditor,
  credentialInputFrom,
  emptyCredentialFields,
} from './AdminApplicationProvisioningShared'
import type { ProvisioningAuthMethod, ProvisioningConnection } from '../../types'

// SCIM の接続先は provisioning/domain の ValidateOutboundBaseURL と同じく、userinfo とフラグメントの
// ない https の URL である。名前の解決と接続の時点の検査はサーバーが行う。
const connectSchema = formSchema({ baseURL: requiredUrl({ https: true }) })

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
  const validation = useFormValidation(connectSchema, { baseURL })

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!validation.data) {
      validation.revealAll()
      return
    }
    setBusy(true)
    setError('')
    try {
      const conn = await registerAdminApplicationProvisioning(csrfToken, applicationID, {
        base_url: validation.data.baseURL,
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
          <Label required>{t.baseUrlFieldLabel}</Label>
          <Input
            required
            placeholder={t.baseUrlPlaceholder}
            value={baseURL}
            onChange={(e) => setBaseURL(e.target.value)}
            {...validation.fieldProps('baseURL')}
          />
          <FieldError form={validation} name="baseURL" />
        </div>
        <CredentialFieldsEditor
          authMethod={authMethod}
          setAuthMethod={setAuthMethod}
          fields={fields}
          setFields={setFields}
        />
        <div>
          <Button type="submit" disabled={busy || !validation.data}>
            {busy ? t.connecting : t.connectButton}
          </Button>
        </div>
      </form>
    </Card>
  )
}

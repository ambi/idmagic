import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import {
  AuthenticationAPIError,
  createCrossAppAccessIssuePolicy,
  createCrossAppAccessRedeemPolicy,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { adminCrossAppAccessDictionary } from './AdminCrossAppAccessPage.i18n'

function listURL() {
  return tenantURL('/admin/cross-app-access')
}

function parseList(value: string): string[] {
  return value.split(/[\s,]+/).filter(Boolean)
}

function Field({
  id,
  label,
  help,
  required = true,
  placeholder,
  value,
  onChange,
}: {
  id: string
  label: string
  help?: string
  required?: boolean
  placeholder?: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        required={required}
        placeholder={placeholder}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
      {help ? <p className="text-xs leading-5 text-slate-500">{help}</p> : null}
    </div>
  )
}

function CreateForm({
  actorUsername,
  title,
  submit,
  children,
}: {
  actorUsername?: string
  title: string
  submit: () => Promise<unknown>
  children: ReactNode
}) {
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const t = useDictionary(adminCrossAppAccessDictionary)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await submit()
      window.location.assign(listURL())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="cross-app-access"
      actorUsername={actorUsername}
      title={title}
      description={t.pageDescription}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full max-w-2xl p-4">
        <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
          {children}
          <div className="flex gap-2.5">
            <Button type="submit" disabled={saving}>
              {t.register}
            </Button>
            <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
              {t.cancel}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

export function AdminCrossAppAccessNewIssuePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const t = useDictionary(adminCrossAppAccessDictionary)
  const [clientId, setClientId] = useState('')
  const [audience, setAudience] = useState('')
  const [relyingParty, setRelyingParty] = useState('')
  const [targetClient, setTargetClient] = useState('')
  const [resources, setResources] = useState('')
  const [scopes, setScopes] = useState('')
  const [detailTypes, setDetailTypes] = useState('')
  const [releaseEmail, setReleaseEmail] = useState(false)

  return (
    <CreateForm
      actorUsername={actorUsername}
      title={t.addIssuePolicy}
      submit={() =>
        createCrossAppAccessIssuePolicy(csrfToken, {
          client_id: clientId,
          audience,
          relying_party_client_id: relyingParty,
          target_client_id: targetClient,
          resources: parseList(resources),
          scopes: parseList(scopes),
          authorization_detail_types: parseList(detailTypes),
          release_email: releaseEmail,
        })
      }
    >
      <Field
        id="client-id"
        label={t.clientIdLabel}
        help={t.clientIdIssueHelp}
        value={clientId}
        onChange={setClientId}
      />
      <Field
        id="audience"
        label={t.audienceLabel}
        placeholder={t.audiencePlaceholder}
        value={audience}
        onChange={setAudience}
      />
      <Field
        id="relying-party"
        label={t.relyingPartyLabel}
        help={t.relyingPartyHelp}
        value={relyingParty}
        onChange={setRelyingParty}
      />
      <Field
        id="target-client"
        label={t.targetClientLabel}
        value={targetClient}
        onChange={setTargetClient}
      />
      <Field
        id="resources"
        label={t.resourcesLabel}
        help={t.listHelp}
        placeholder={t.resourcesPlaceholder}
        value={resources}
        onChange={setResources}
      />
      <Field
        id="scopes"
        label={t.scopesLabel}
        help={t.listHelp}
        value={scopes}
        onChange={setScopes}
      />
      <Field
        id="detail-types"
        label={t.detailTypesLabel}
        help={t.listHelp}
        required={false}
        value={detailTypes}
        onChange={setDetailTypes}
      />
      <label className="flex items-center gap-2 text-sm text-slate-700">
        <input
          type="checkbox"
          checked={releaseEmail}
          onChange={(event) => setReleaseEmail(event.target.checked)}
        />
        {t.releaseEmailLabel}
      </label>
    </CreateForm>
  )
}

export function AdminCrossAppAccessNewRedeemPage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const t = useDictionary(adminCrossAppAccessDictionary)
  const [providerId, setProviderId] = useState('')
  const [clientId, setClientId] = useState('')
  const [scopes, setScopes] = useState('')

  return (
    <CreateForm
      actorUsername={actorUsername}
      title={t.addRedeemPolicy}
      submit={() =>
        createCrossAppAccessRedeemPolicy(csrfToken, {
          provider_id: providerId,
          client_id: clientId,
          scopes: parseList(scopes),
        })
      }
    >
      <Field id="provider" label={t.providerLabel} value={providerId} onChange={setProviderId} />
      <Field
        id="client-id"
        label={t.clientIdLabel}
        help={t.clientIdRedeemHelp}
        value={clientId}
        onChange={setClientId}
      />
      <Field
        id="scopes"
        label={t.scopesLabel}
        help={t.listHelp}
        value={scopes}
        onChange={setScopes}
      />
    </CreateForm>
  )
}

import { type FormEvent, type ReactNode, useState } from 'react'
import {
  AuthenticationAPIError,
  createIdentityProviderConnection,
  deleteIdentityProviderConnection,
  previewIdentityProviderMapping,
  runIdentityProviderAction,
  updateIdentityProviderConnection,
  type IdentityProviderConnection,
  type IdentityProviderConnectionInput,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import { identityProvidersDictionary } from './IdentityProvidersTab.i18n'

type FormState = {
  displayName: string
  protocol: 'oidc' | 'saml'
  issuer: string
  clientId: string
  secretReference: string
  authorizationEndpoint: string
  tokenEndpoint: string
  jwksUri: string
  samlEntityId: string
  samlSsoUrl: string
  samlCertificates: string
  subjectClaim: string
  usernameClaim: string
  emailClaim: string
  emailVerifiedClaim: string
  nameClaim: string
  linkingPolicy: 'none' | 'verified_email'
  jitProvisioning: boolean
  allowedDomains: string
}

const emptyForm: FormState = {
  displayName: '',
  protocol: 'oidc',
  issuer: '',
  clientId: '',
  secretReference: '',
  authorizationEndpoint: '',
  tokenEndpoint: '',
  jwksUri: '',
  samlEntityId: '',
  samlSsoUrl: '',
  samlCertificates: '',
  subjectClaim: 'sub',
  usernameClaim: 'preferred_username',
  emailClaim: 'email',
  emailVerifiedClaim: 'email_verified',
  nameClaim: 'name',
  linkingPolicy: 'none',
  jitProvisioning: false,
  allowedDomains: '',
}

function inputFrom(form: FormState): IdentityProviderConnectionInput {
  return {
    display_name: form.displayName.trim(),
    protocol: form.protocol,
    issuer: form.issuer.trim(),
    client_id: form.protocol === 'oidc' ? form.clientId.trim() : undefined,
    secret_reference:
      form.protocol === 'oidc' && form.secretReference.trim()
        ? form.secretReference.trim()
        : undefined,
    authorization_endpoint:
      form.protocol === 'oidc' ? form.authorizationEndpoint.trim() : undefined,
    token_endpoint: form.protocol === 'oidc' ? form.tokenEndpoint.trim() : undefined,
    jwks_uri: form.protocol === 'oidc' ? form.jwksUri.trim() : undefined,
    saml_entity_id: form.protocol === 'saml' ? form.samlEntityId.trim() : undefined,
    saml_sso_url: form.protocol === 'saml' ? form.samlSsoUrl.trim() : undefined,
    saml_signing_certificates:
      form.protocol === 'saml'
        ? form.samlCertificates
            .split(/\n\s*\n/)
            .map((value) => value.trim())
            .filter(Boolean)
        : undefined,
    claim_mapping: {
      subject: form.subjectClaim.trim(),
      username: form.usernameClaim.trim(),
      email: form.emailClaim.trim() || undefined,
      email_verified: form.emailVerifiedClaim.trim() || undefined,
      name: form.nameClaim.trim() || undefined,
    },
    linking_policy: form.linkingPolicy,
    jit_provisioning: form.jitProvisioning,
    allowed_email_domains: form.allowedDomains
      .split(',')
      .map((value) => value.trim())
      .filter(Boolean),
  }
}

function formFrom(connection: IdentityProviderConnection): FormState {
  return {
    ...emptyForm,
    displayName: connection.display_name,
    protocol: connection.protocol,
    issuer: connection.issuer,
    clientId: connection.client_id ?? '',
    authorizationEndpoint: connection.authorization_endpoint ?? '',
    tokenEndpoint: connection.token_endpoint ?? '',
    jwksUri: connection.jwks_uri ?? '',
    samlEntityId: connection.saml_entity_id ?? '',
    samlSsoUrl: connection.saml_sso_url ?? '',
    samlCertificates: (connection.saml_signing_certificates ?? []).join('\n\n'),
    subjectClaim: connection.claim_mapping.subject,
    usernameClaim: connection.claim_mapping.username,
    emailClaim: connection.claim_mapping.email ?? '',
    emailVerifiedClaim: connection.claim_mapping.email_verified ?? '',
    nameClaim: connection.claim_mapping.name ?? '',
    linkingPolicy: connection.linking_policy,
    jitProvisioning: connection.jit_provisioning,
    allowedDomains: (connection.allowed_email_domains ?? []).join(', '),
  }
}

export function IdentityProvidersTab({
  csrfToken,
  initialConnections,
}: {
  csrfToken: string
  initialConnections: IdentityProviderConnection[]
}) {
  const t = useDictionary(identityProvidersDictionary)
  const [connections, setConnections] = useState(initialConnections)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [editingID, setEditingID] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [previewClaims, setPreviewClaims] = useState(
    '{"sub":"subject-123","preferred_username":"alice","email":"alice@example.com","email_verified":true}',
  )
  const [preview, setPreview] = useState('')

  function field<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((current) => ({ ...current, [key]: value }))
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const saved = editingID
        ? await updateIdentityProviderConnection(csrfToken, editingID, inputFrom(form))
        : await createIdentityProviderConnection(csrfToken, inputFrom(form))
      setConnections((current) => [
        ...current.filter((connection) => connection.id !== saved.id),
        saved,
      ])
      setForm(emptyForm)
      setEditingID('')
      setNotice(t.saved)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
    } finally {
      setBusy(false)
    }
  }

  async function action(
    connection: IdentityProviderConnection,
    operation: 'activate' | 'disable' | 'refresh' | 'test',
  ) {
    setBusy(true)
    setError('')
    try {
      await runIdentityProviderAction(csrfToken, connection.id, operation)
      if (operation === 'activate' || operation === 'disable') {
        setConnections((current) =>
          current.map((item) =>
            item.id === connection.id
              ? { ...item, status: operation === 'activate' ? 'active' : 'disabled' }
              : item,
          ),
        )
      }
      setNotice(t.actionCompleted)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
    } finally {
      setBusy(false)
    }
  }

  async function remove(connection: IdentityProviderConnection) {
    setBusy(true)
    setError('')
    try {
      await deleteIdentityProviderConnection(csrfToken, connection.id)
      setConnections((current) => current.filter((item) => item.id !== connection.id))
      setNotice(t.deleted)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
    } finally {
      setBusy(false)
    }
  }

  async function previewCurrentMapping() {
    if (!editingID) return
    setBusy(true)
    setError('')
    try {
      const claims = JSON.parse(previewClaims) as Record<string, unknown>
      const result = await previewIdentityProviderMapping(csrfToken, editingID, claims)
      setPreview(JSON.stringify(result, null, 2))
    } catch (cause) {
      setError(
        cause instanceof SyntaxError
          ? t.invalidClaimsJson
          : cause instanceof AuthenticationAPIError
            ? cause.message
            : t.failed,
      )
    } finally {
      setBusy(false)
    }
  }

  const statusLabel = {
    draft: t.statusDraft,
    active: t.statusActive,
    disabled: t.statusDisabled,
  }

  return (
    <section className="grid gap-6">
      <header>
        <h2 className="text-xl font-semibold text-slate-950">{t.heading}</h2>
        <p className="mt-1 text-sm text-slate-600">{t.description}</p>
      </header>
      <Toast message={notice} onDismiss={() => setNotice('')} />
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {connections.length === 0 ? (
        <Card className="p-5 text-sm text-slate-500">{t.empty}</Card>
      ) : (
        <div className="grid gap-3">
          {connections.map((connection) => (
            <Card key={connection.id} className="flex flex-wrap items-center gap-3 p-4">
              <div className="min-w-0 flex-1">
                <p className="font-semibold text-slate-900">{connection.display_name}</p>
                <p className="text-xs text-slate-500">
                  {connection.protocol.toUpperCase()} · {statusLabel[connection.status]}
                </p>
              </div>
              <Button
                variant="outline"
                disabled={busy}
                onClick={() => {
                  setEditingID(connection.id)
                  setForm(formFrom(connection))
                }}
              >
                {t.edit}
              </Button>
              <Button variant="outline" disabled={busy} onClick={() => action(connection, 'test')}>
                {t.test}
              </Button>
              {connection.protocol === 'oidc' ? (
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => action(connection, 'refresh')}
                >
                  {t.refresh}
                </Button>
              ) : null}
              {connection.status === 'active' ? (
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => action(connection, 'disable')}
                >
                  {t.disable}
                </Button>
              ) : (
                <Button disabled={busy} onClick={() => action(connection, 'activate')}>
                  {t.activate}
                </Button>
              )}
              {connection.status === 'disabled' ? (
                <Button variant="destructive" disabled={busy} onClick={() => remove(connection)}>
                  {t.delete}
                </Button>
              ) : null}
            </Card>
          ))}
        </div>
      )}

      <Card className="p-5">
        <h3 className="mb-4 font-semibold text-slate-900">
          {editingID ? t.editConnection : t.newConnection}
        </h3>
        <form className="grid gap-4 md:grid-cols-2" onSubmit={submit}>
          <Field label={t.displayName}>
            <Input
              required
              value={form.displayName}
              onChange={(event) => field('displayName', event.target.value)}
            />
          </Field>
          <Field label={t.protocol}>
            <select
              className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm"
              value={form.protocol}
              onChange={(event) => field('protocol', event.target.value as 'oidc' | 'saml')}
            >
              <option value="oidc">OIDC</option>
              <option value="saml">SAML</option>
            </select>
          </Field>
          <Field label={t.issuer}>
            <Input
              required
              type="url"
              value={form.issuer}
              onChange={(event) => field('issuer', event.target.value)}
            />
          </Field>
          {form.protocol === 'oidc' ? (
            <>
              <Field label={t.clientId}>
                <Input
                  required
                  value={form.clientId}
                  onChange={(event) => field('clientId', event.target.value)}
                />
              </Field>
              <Field label={t.secretReference} help={t.secretReferenceHelp}>
                <Input
                  value={form.secretReference}
                  onChange={(event) => field('secretReference', event.target.value)}
                />
              </Field>
              <URLField
                label={t.authorizationEndpoint}
                value={form.authorizationEndpoint}
                onChange={(value) => field('authorizationEndpoint', value)}
              />
              <URLField
                label={t.tokenEndpoint}
                value={form.tokenEndpoint}
                onChange={(value) => field('tokenEndpoint', value)}
              />
              <URLField
                label={t.jwksUri}
                value={form.jwksUri}
                onChange={(value) => field('jwksUri', value)}
              />
            </>
          ) : (
            <>
              <Field label={t.samlEntityId}>
                <Input
                  required
                  value={form.samlEntityId}
                  onChange={(event) => field('samlEntityId', event.target.value)}
                />
              </Field>
              <URLField
                label={t.samlSsoUrl}
                value={form.samlSsoUrl}
                onChange={(value) => field('samlSsoUrl', value)}
              />
              <Field label={t.samlCertificates}>
                <textarea
                  required
                  className="min-h-28 rounded-md border border-slate-300 p-3 font-mono text-xs"
                  value={form.samlCertificates}
                  onChange={(event) => field('samlCertificates', event.target.value)}
                />
              </Field>
            </>
          )}
          {(
            [
              ['subjectClaim', t.subjectClaim],
              ['usernameClaim', t.usernameClaim],
              ['emailClaim', t.emailClaim],
              ['emailVerifiedClaim', t.emailVerifiedClaim],
              ['nameClaim', t.nameClaim],
            ] as const
          ).map(([key, label]) => (
            <Field key={key} label={label}>
              <Input
                required={key === 'subjectClaim' || key === 'usernameClaim'}
                value={form[key]}
                onChange={(event) => field(key, event.target.value)}
              />
            </Field>
          ))}
          <Field label={t.linkingPolicy}>
            <select
              className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm"
              value={form.linkingPolicy}
              onChange={(event) =>
                field('linkingPolicy', event.target.value as 'none' | 'verified_email')
              }
            >
              <option value="none">{t.linkingNone}</option>
              <option value="verified_email">{t.linkingVerifiedEmail}</option>
            </select>
          </Field>
          <Field label={t.allowedDomains}>
            <Input
              value={form.allowedDomains}
              onChange={(event) => field('allowedDomains', event.target.value)}
            />
          </Field>
          <label className="flex items-center gap-2 text-sm font-medium text-slate-700 md:col-span-2">
            <input
              type="checkbox"
              checked={form.jitProvisioning}
              onChange={(event) => field('jitProvisioning', event.target.checked)}
            />
            {t.jitProvisioning}
          </label>
          <div className="flex gap-2 md:col-span-2">
            <Button type="submit" disabled={busy}>
              {editingID ? t.update : t.create}
            </Button>
            {editingID ? (
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setEditingID('')
                  setForm(emptyForm)
                }}
              >
                {t.cancel}
              </Button>
            ) : null}
          </div>
          {editingID ? (
            <div className="grid gap-2 border-t border-slate-200 pt-4 md:col-span-2">
              <Label htmlFor="identity-provider-preview">{t.previewClaims}</Label>
              <textarea
                id="identity-provider-preview"
                className="min-h-28 rounded-md border border-slate-300 p-3 font-mono text-xs"
                value={previewClaims}
                onChange={(event) => setPreviewClaims(event.target.value)}
              />
              <div>
                <Button
                  type="button"
                  variant="outline"
                  disabled={busy}
                  onClick={previewCurrentMapping}
                >
                  {t.previewMapping}
                </Button>
              </div>
              {preview ? (
                <div>
                  <p className="mb-1 text-sm font-medium text-slate-700">{t.previewResult}</p>
                  <pre className="overflow-auto rounded-md bg-slate-950 p-3 text-xs text-white">
                    {preview}
                  </pre>
                </div>
              ) : null}
            </div>
          ) : null}
        </form>
      </Card>
    </section>
  )
}

function Field({ label, help, children }: { label: string; help?: string; children: ReactNode }) {
  return (
    <Label className="grid gap-1.5">
      <span>{label}</span>
      {children}
      {help ? <span className="text-xs font-normal text-slate-500">{help}</span> : null}
    </Label>
  )
}

function URLField({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <Field label={label}>
      <Input required type="url" value={value} onChange={(event) => onChange(event.target.value)} />
    </Field>
  )
}

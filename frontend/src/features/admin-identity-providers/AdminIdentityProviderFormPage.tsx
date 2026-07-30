import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  createIdentityProviderConnection,
  previewIdentityProviderMapping,
  tenantURL,
  updateIdentityProviderConnection,
  type IdentityProviderConnection,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { TrustSourceFields } from './AdminIdentityProviderFormFields'
import {
  emptyForm,
  Field,
  type FormState,
  formFrom,
  inputFrom,
} from './AdminIdentityProviderFormShared'
import { identityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'

export function AdminIdentityProviderCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  return (
    <ConnectionFormShell csrfToken={csrfToken} actorUsername={actorUsername} connection={null} />
  )
}

export function AdminIdentityProviderEditPage({
  csrfToken,
  actorUsername,
  connection,
}: {
  csrfToken: string
  actorUsername?: string
  connection: IdentityProviderConnection
}) {
  return (
    <ConnectionFormShell
      csrfToken={csrfToken}
      actorUsername={actorUsername}
      connection={connection}
    />
  )
}

function ConnectionFormShell({
  csrfToken,
  actorUsername,
  connection,
}: {
  csrfToken: string
  actorUsername?: string
  connection: IdentityProviderConnection | null
}) {
  const editing = connection !== null
  const [form, setForm] = useState<FormState>(connection ? formFrom(connection) : emptyForm)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [previewClaims, setPreviewClaims] = useState(
    '{"sub":"subject-123","preferred_username":"alice","email":"alice@example.com","email_verified":true}',
  )
  const [preview, setPreview] = useState('')
  const [previewError, setPreviewError] = useState('')
  const t = useDictionary(identityProvidersDictionary)

  function field<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((current) => ({ ...current, [key]: value }))
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const saved = editing
        ? await updateIdentityProviderConnection(csrfToken, connection.id, inputFrom(form))
        : await createIdentityProviderConnection(csrfToken, inputFrom(form))
      window.location.href = tenantURL(`/admin/identity-providers/${encodeURIComponent(saved.id)}`)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
      setBusy(false)
    }
  }

  async function previewCurrentMapping() {
    if (!connection) return
    setPreviewError('')
    try {
      const claims = JSON.parse(previewClaims) as Record<string, unknown>
      const result = await previewIdentityProviderMapping(csrfToken, connection.id, claims)
      setPreview(JSON.stringify(result, null, 2))
    } catch (cause) {
      setPreviewError(
        cause instanceof SyntaxError
          ? t.invalidClaimsJson
          : cause instanceof AuthenticationAPIError
            ? cause.message
            : t.failed,
      )
    }
  }

  return (
    <AdminShell
      active="identity-providers"
      actorUsername={actorUsername}
      title={editing ? connection.display_name : t.newConnectionTitle}
      actions={
        <a
          href={tenantURL(
            editing
              ? `/admin/identity-providers/${encodeURIComponent(connection.id)}`
              : '/admin/identity-providers',
          )}
          className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.cancel}
        </a>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      <Card className="p-5">
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

          <TrustSourceFields
            form={form}
            field={field}
            editing={editing}
            connection={connection}
            t={t}
          />

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
              {editing ? t.update : t.create}
            </Button>
          </div>

          {editing ? (
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
                  onClick={() => void previewCurrentMapping()}
                >
                  {t.previewMapping}
                </Button>
              </div>
              {previewError ? <Alert variant="destructive">{previewError}</Alert> : null}
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
    </AdminShell>
  )
}

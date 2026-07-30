import type { IdentityProviderConnection } from '../../api'
import { Input } from '../../components/ui/input'
import type { AdminIdentityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'
import { Field, type FormState, URLField } from './AdminIdentityProviderFormShared'

// TrustSourceFields はプロトコル固有 (OIDC/SAML) の trust source フィールドと
// claim mapping フィールドをまとめる。400 行の ui-page-lines 予算に収めるため
// AdminIdentityProviderFormPage.tsx から抽出した (architecture.yaml)。
export function TrustSourceFields({
  form,
  field,
  editing,
  connection,
  t,
}: {
  form: FormState
  field: <K extends keyof FormState>(key: K, value: FormState[K]) => void
  editing: boolean
  connection: IdentityProviderConnection | null
  t: AdminIdentityProvidersDictionary
}) {
  return (
    <>
      {form.protocol === 'oidc' ? (
        <>
          <Field label={t.clientId}>
            <Input
              required
              value={form.clientId}
              onChange={(event) => field('clientId', event.target.value)}
            />
          </Field>
          <Field
            label={t.secretValue}
            help={
              editing && connection
                ? `${connection.client_secret_configured ? t.secretConfigured : t.secretNotConfigured} — ${t.secretValueHelp}`
                : t.secretValueHelp
            }
          >
            <Input
              type="password"
              autoComplete="off"
              value={form.secretValue}
              onChange={(event) => field('secretValue', event.target.value)}
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
    </>
  )
}

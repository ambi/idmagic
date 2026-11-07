import type { IdentityProviderConnection } from '../../api'
import { FieldError } from '../../components/FieldError'
import { Input } from '../../components/ui/input'
import type { AdminIdentityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'
import {
  Field,
  type FormState,
  type IdentityProviderValidation,
  URLField,
} from './AdminIdentityProviderFormShared'

// TrustSourceFields はプロトコル固有 (OIDC/SAML) の trust source フィールドと
// claim mapping フィールドをまとめる。400 行の ui-page-lines 予算に収めるため
// AdminIdentityProviderFormPage.tsx から抽出した (architecture.yaml)。
export function TrustSourceFields({
  form,
  validation,
  field,
  editing,
  connection,
  t,
}: {
  form: FormState
  validation: IdentityProviderValidation
  field: <K extends keyof FormState>(key: K, value: FormState[K]) => void
  editing: boolean
  connection: IdentityProviderConnection | null
  t: AdminIdentityProvidersDictionary
}) {
  return (
    <>
      {form.protocol === 'oidc' ? (
        <>
          <Field label={t.clientId} error={<FieldError form={validation} name="clientId" />}>
            <Input
              required
              value={form.clientId}
              onChange={(event) => field('clientId', event.target.value)}
              {...validation.fieldProps('clientId')}
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
            validation={validation}
            name={'authorizationEndpoint'}
          />
          <URLField
            label={t.tokenEndpoint}
            value={form.tokenEndpoint}
            onChange={(value) => field('tokenEndpoint', value)}
            validation={validation}
            name={'tokenEndpoint'}
          />
          <URLField
            label={t.jwksUri}
            value={form.jwksUri}
            onChange={(value) => field('jwksUri', value)}
            validation={validation}
            name={'jwksUri'}
          />
        </>
      ) : (
        <>
          <Field
            label={t.samlEntityId}
            error={<FieldError form={validation} name="samlEntityId" />}
          >
            <Input
              required
              value={form.samlEntityId}
              onChange={(event) => field('samlEntityId', event.target.value)}
              {...validation.fieldProps('samlEntityId')}
            />
          </Field>
          <URLField
            label={t.samlSsoUrl}
            value={form.samlSsoUrl}
            onChange={(value) => field('samlSsoUrl', value)}
            validation={validation}
            name={'samlSsoUrl'}
          />
          <Field
            label={t.samlCertificates}
            error={<FieldError form={validation} name="samlCertificates" />}
          >
            <textarea
              required
              className="min-h-28 rounded-md border border-slate-300 p-3 font-mono text-xs"
              value={form.samlCertificates}
              onChange={(event) => field('samlCertificates', event.target.value)}
              {...validation.fieldProps('samlCertificates')}
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
      ).map(([key, label]) => {
        const required = key === 'subjectClaim' || key === 'usernameClaim'
        return (
          <Field
            key={key}
            label={label}
            error={required ? <FieldError form={validation} name={key} /> : undefined}
          >
            <Input
              required={required}
              value={form[key]}
              onChange={(event) => field(key, event.target.value)}
              {...(required ? validation.fieldProps(key) : {})}
            />
          </Field>
        )
      })}
    </>
  )
}

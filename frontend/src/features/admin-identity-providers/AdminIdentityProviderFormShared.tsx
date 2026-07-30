import type { ReactNode } from 'react'
import type { IdentityProviderConnection, IdentityProviderConnectionInput } from '../../api'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'

export type FormState = {
  displayName: string
  protocol: 'oidc' | 'saml'
  issuer: string
  clientId: string
  secretValue: string
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

export const emptyForm: FormState = {
  displayName: '',
  protocol: 'oidc',
  issuer: '',
  clientId: '',
  secretValue: '',
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

export function inputFrom(form: FormState): IdentityProviderConnectionInput {
  return {
    display_name: form.displayName.trim(),
    protocol: form.protocol,
    issuer: form.issuer.trim(),
    client_id: form.protocol === 'oidc' ? form.clientId.trim() : undefined,
    secret_reference:
      form.protocol === 'oidc' && form.secretValue.trim() ? form.secretValue.trim() : undefined,
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

export function formFrom(connection: IdentityProviderConnection): FormState {
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

export function Field({
  label,
  help,
  children,
}: {
  label: string
  help?: string
  children: ReactNode
}) {
  return (
    <Label className="grid gap-1.5">
      <span>{label}</span>
      {children}
      {help ? <span className="text-xs font-normal text-slate-500">{help}</span> : null}
    </Label>
  )
}

export function URLField({
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

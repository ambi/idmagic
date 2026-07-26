import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import type { AdminApplicationDetail, AdminIntegrationEndpointCatalog } from '../../types'
import { CopyableField } from './AdminApplicationsShared'
import type { AdminApplicationsDictionary } from './AdminApplicationsPage.i18n'

export function IdMagicSetupCard({
  detail,
  integrationEndpoints,
  t,
}: {
  detail: AdminApplicationDetail
  integrationEndpoints: AdminIntegrationEndpointCatalog
  t: AdminApplicationsDictionary
}) {
  if (!detail.oidc && !detail.saml) return null

  return (
    <Card className="p-5">
      <header>
        <h2 className="text-base font-semibold text-slate-900">{t.idmagicSetupHeading}</h2>
        <p className="mt-1 text-sm text-slate-600">
          {detail.oidc ? t.oidcSetupDescription : t.samlSetupDescription}
        </p>
      </header>
      {detail.oidc ? (
        <div className="mt-5 grid gap-4">
          <CopyableField label={t.clientIdFieldLabel} value={detail.oidc.client_id} />
          <CopyableField label={t.idpIssuerFieldLabel} value={integrationEndpoints.issuer} />
          <CopyableField
            label={t.openidDiscoveryFieldLabel}
            value={integrationEndpoints.oauth.openid_configuration}
          />
          <CopyableField
            label={t.authorizationEndpointSetupLabel}
            value={integrationEndpoints.oauth.authorization_endpoint}
          />
          <CopyableField
            label={t.tokenEndpointSetupLabel}
            value={integrationEndpoints.oauth.token_endpoint}
          />
          <CopyableField
            label={t.idpJwksUriFieldLabel}
            value={integrationEndpoints.oauth.jwks_uri}
          />
        </div>
      ) : null}
      {detail.saml ? (
        <div className="mt-5 grid gap-4">
          <CopyableField
            label={t.idpMetadataUrlFieldLabel}
            value={integrationEndpoints.saml.metadata_url}
          />
          <CopyableField
            label={t.idpEntityIdFieldLabel}
            value={integrationEndpoints.saml.entity_id}
          />
          <CopyableField label={t.idpSsoUrlFieldLabel} value={integrationEndpoints.saml.sso_url} />
          <CopyableField label={t.idpSloUrlFieldLabel} value={integrationEndpoints.saml.slo_url} />
          <CopyableField
            label={t.signingCertificateFingerprintFieldLabel}
            value={integrationEndpoints.saml.signing_certificate.fingerprint_sha256}
          />
          <Button asChild variant="outline" className="justify-self-start">
            <a href={integrationEndpoints.saml.signing_certificate.download_url} download>
              {t.downloadSigningCertificate}
            </a>
          </Button>
        </div>
      ) : null}
    </Card>
  )
}

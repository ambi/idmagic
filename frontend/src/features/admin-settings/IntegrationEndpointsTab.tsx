import { IconCopy, IconDownload, IconSettings } from '@tabler/icons-react'
import { tenantURL } from '../../api'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AdminIntegrationEndpointCatalog, AdminSamlIDPProfile } from '../../types'
import { adminSettingsDictionary, type AdminSettingsDictionary } from './AdminSettingsPage.i18n'

function EndpointField({
  label,
  value,
  t,
}: {
  label: string
  value: string
  t: AdminSettingsDictionary
}) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      <div className="flex items-center gap-2">
        <code className="min-w-0 flex-1 break-all rounded-md bg-slate-50 px-3 py-2 font-mono text-xs text-slate-800">
          {value}
        </code>
        <Button
          type="button"
          variant="outline"
          className="shrink-0"
          aria-label={t.copy}
          onClick={() => void navigator.clipboard?.writeText(value)}
        >
          <IconCopy size={16} aria-hidden="true" />
          <span className="hidden sm:inline">{t.copy}</span>
        </Button>
      </div>
    </div>
  )
}

function EndpointSection({
  title,
  description,
  fields,
  t,
  children,
}: {
  title: string
  description: string
  fields: Array<{ label: string; value: string }>
  t: AdminSettingsDictionary
  children?: React.ReactNode
}) {
  return (
    <Card className="p-6">
      <header>
        <h3 className="text-base font-semibold text-slate-900">{title}</h3>
        <p className="mt-1 text-sm text-slate-600">{description}</p>
      </header>
      <div className="mt-5 grid gap-4">
        {fields.map((field) => (
          <EndpointField key={field.label} label={field.label} value={field.value} t={t} />
        ))}
        {children}
      </div>
    </Card>
  )
}

export function IntegrationEndpointsTab({
  catalog,
  initialSamlIDPProfiles = [],
}: {
  catalog: AdminIntegrationEndpointCatalog
  initialSamlIDPProfiles?: AdminSamlIDPProfile[]
}) {
  const t = useDictionary(adminSettingsDictionary)

  return (
    <div className="grid gap-5">
      <div>
        <h2 className="text-base font-semibold text-slate-900">{t.integrationEndpointsHeading}</h2>
        <p className="mt-1 text-sm text-slate-600">{t.integrationEndpointsDescription}</p>
      </div>

      <EndpointSection
        title={t.oidcEndpointsHeading}
        description={t.oidcEndpointsDescription}
        t={t}
        fields={[
          { label: t.issuerLabel, value: catalog.issuer },
          {
            label: t.openidConfigurationLabel,
            value: catalog.oauth.openid_configuration,
          },
          {
            label: t.oauthAuthorizationServerLabel,
            value: catalog.oauth.oauth_authorization_server,
          },
          { label: t.authorizationEndpointLabel, value: catalog.oauth.authorization_endpoint },
          { label: t.tokenEndpointLabel, value: catalog.oauth.token_endpoint },
          { label: t.userinfoEndpointLabel, value: catalog.oauth.userinfo_endpoint },
          { label: t.jwksUriLabel, value: catalog.oauth.jwks_uri },
          { label: t.revocationEndpointLabel, value: catalog.oauth.revocation_endpoint },
          {
            label: t.introspectionEndpointLabel,
            value: catalog.oauth.introspection_endpoint,
          },
          { label: t.endSessionEndpointLabel, value: catalog.oauth.end_session_endpoint },
          { label: t.registrationEndpointLabel, value: catalog.oauth.registration_endpoint },
          {
            label: t.parEndpointLabel,
            value: catalog.oauth.pushed_authorization_request_endpoint,
          },
          {
            label: t.deviceAuthorizationEndpointLabel,
            value: catalog.oauth.device_authorization_endpoint,
          },
        ]}
      />

      <EndpointSection
        title={t.samlEndpointsHeading}
        description={t.samlEndpointsDescription}
        t={t}
        fields={[]}
      >
        <section>
          <div className="flex justify-end">
            <Button asChild variant="outline">
              <a href={tenantURL('/admin/settings/saml-idp-profiles')}>
                <IconSettings size={16} aria-hidden="true" />
                {t.samlProfilesManage}
              </a>
            </Button>
          </div>
          <div className="mt-4 grid gap-3">
            {initialSamlIDPProfiles.length === 0 ? (
              <p className="rounded-lg border border-dashed border-slate-300 px-4 py-5 text-sm text-slate-500">
                {t.samlProfilesEmpty}
              </p>
            ) : (
              initialSamlIDPProfiles.map((entry) => (
                <div
                  key={entry.profile.profile_id}
                  className="grid gap-4 rounded-lg border border-slate-200 bg-slate-50/60 p-4"
                >
                  <div>
                    <p className="font-medium text-slate-900">{entry.profile.name}</p>
                    <p className="mt-1 text-xs text-slate-500">
                      {entry.profile.is_default ? `${t.samlProfileDefault}. ` : ''}
                      {entry.profile.mode === 'shared'
                        ? t.samlProfileSharedLabel
                        : t.samlProfileDedicatedLabel}
                      {'. '}
                      {t.samlProfileAssignments.replace(
                        '{count}',
                        String(entry.service_provider_count),
                      )}
                    </p>
                  </div>
                  <EndpointField label={t.samlEntityIdLabel} value={entry.entity_id} t={t} />
                  <EndpointField label={t.samlMetadataUrlLabel} value={entry.metadata_url} t={t} />
                  <EndpointField label={t.samlSsoUrlLabel} value={entry.sso_url} t={t} />
                  <EndpointField label={t.samlSloUrlLabel} value={entry.slo_url} t={t} />
                  <EndpointField
                    label={t.certificateFingerprintLabel}
                    value={entry.signing_certificate_fingerprint_sha256}
                    t={t}
                  />
                  <Button asChild variant="outline" className="w-fit">
                    <a href={entry.signing_certificate_url} download>
                      <IconDownload size={16} aria-hidden="true" />
                      {t.downloadCertificate}
                    </a>
                  </Button>
                </div>
              ))
            )}
          </div>
        </section>
      </EndpointSection>

      <EndpointSection
        title={t.wsFederationEndpointsHeading}
        description={t.wsFederationEndpointsDescription}
        t={t}
        fields={[
          { label: t.wsFederationRealmLabel, value: catalog.ws_federation.realm },
          { label: t.wsFederationMetadataLabel, value: catalog.ws_federation.metadata_url },
          { label: t.passiveLogonUrlLabel, value: catalog.ws_federation.passive_logon_url },
          { label: t.activeLogonUrlLabel, value: catalog.ws_federation.active_logon_url },
          {
            label: t.metadataExchangeUrlLabel,
            value: catalog.ws_federation.metadata_exchange_url,
          },
        ]}
      />

      <EndpointSection
        title={t.apiEndpointsHeading}
        description={t.apiEndpointsDescription}
        t={t}
        fields={[
          {
            label: t.managementApiBaseUrlLabel,
            value: catalog.apis.management_api_base_url,
          },
          { label: t.scimBaseUrlLabel, value: catalog.apis.scim_base_url },
          { label: t.accountApiBaseUrlLabel, value: catalog.apis.account_api_base_url },
        ]}
      />
    </div>
  )
}

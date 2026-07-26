import { IconCopy, IconDownload } from '@tabler/icons-react'
import { useState } from 'react'
import { createSamlIDPProfile, deleteSamlIDPProfile, updateSamlIDPProfile } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { useDictionary, useLocale } from '../../lib/i18n'
import type {
  AdminIntegrationEndpointCatalog,
  AdminSamlIDPProfile,
  SamlIDPProfileMode,
} from '../../types'
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
  csrfToken = '',
  initialSamlIDPProfiles = [],
}: {
  catalog: AdminIntegrationEndpointCatalog
  csrfToken?: string
  initialSamlIDPProfiles?: AdminSamlIDPProfile[]
}) {
  const t = useDictionary(adminSettingsDictionary)
  const { locale } = useLocale()
  const [profiles, setProfiles] = useState(initialSamlIDPProfiles)
  const [newProfileName, setNewProfileName] = useState('')
  const [newProfileMode, setNewProfileMode] = useState<SamlIDPProfileMode>('shared')
  const [profileError, setProfileError] = useState('')
  const [profileBusy, setProfileBusy] = useState(false)
  const certificate = catalog.saml.signing_certificate
  const dateLocale = locale === 'ja' ? 'ja-JP' : 'en-US'

  async function createProfile() {
    if (!newProfileName.trim()) return
    setProfileBusy(true)
    setProfileError('')
    try {
      const created = await createSamlIDPProfile(csrfToken, {
        name: newProfileName.trim(),
        mode: newProfileMode,
      })
      setProfiles((current) => [...current, created])
      setNewProfileName('')
    } catch {
      setProfileError(t.samlProfileOperationFailed)
    } finally {
      setProfileBusy(false)
    }
  }

  async function saveProfile(profile: AdminSamlIDPProfile) {
    setProfileBusy(true)
    setProfileError('')
    try {
      const updated = await updateSamlIDPProfile(csrfToken, profile.profile.profile_id, {
        name: profile.profile.name.trim(),
        mode: profile.profile.mode,
      })
      setProfiles((current) =>
        current.map((entry) =>
          entry.profile.profile_id === updated.profile.profile_id ? updated : entry,
        ),
      )
    } catch {
      setProfileError(t.samlProfileOperationFailed)
    } finally {
      setProfileBusy(false)
    }
  }

  async function removeProfile(profileID: string) {
    setProfileBusy(true)
    setProfileError('')
    try {
      await deleteSamlIDPProfile(csrfToken, profileID)
      setProfiles((current) => current.filter(({ profile }) => profile.profile_id !== profileID))
    } catch {
      setProfileError(t.samlProfileDeleteFailed)
    } finally {
      setProfileBusy(false)
    }
  }

  return (
    <div className="grid gap-5">
      <div>
        <h2 className="text-base font-semibold text-slate-900">{t.integrationEndpointsHeading}</h2>
        <p className="mt-1 text-sm text-slate-600">{t.integrationEndpointsDescription}</p>
      </div>

      <Card className="p-6">
        <header>
          <h3 className="text-base font-semibold text-slate-900">{t.samlProfilesHeading}</h3>
          <p className="mt-1 text-sm text-slate-600">{t.samlProfilesDescription}</p>
        </header>
        {profileError ? (
          <Alert variant="destructive" className="mt-4">
            {profileError}
          </Alert>
        ) : null}
        <div className="mt-5 grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-4 sm:grid-cols-[minmax(0,1fr)_12rem_auto]">
          <div className="grid gap-1.5">
            <Label htmlFor="new-saml-profile-name">{t.samlProfileNameLabel}</Label>
            <Input
              id="new-saml-profile-name"
              value={newProfileName}
              onChange={(event) => setNewProfileName(event.target.value)}
            />
          </div>
          <div className="grid gap-1.5">
            <Label>{t.samlProfileModeLabel}</Label>
            <Select
              value={newProfileMode}
              onValueChange={(value) => setNewProfileMode(value as SamlIDPProfileMode)}
              options={[
                { value: 'shared', label: t.samlProfileSharedLabel },
                { value: 'dedicated', label: t.samlProfileDedicatedLabel },
              ]}
            />
          </div>
          <Button
            type="button"
            className="self-end"
            disabled={profileBusy || !newProfileName.trim()}
            onClick={() => void createProfile()}
          >
            {t.samlProfileCreate}
          </Button>
        </div>
        <div className="mt-5 grid gap-4">
          {profiles.map((entry) => {
            const profile = entry.profile
            const immutable = profile.is_default
            const deleteDisabled = immutable || entry.service_provider_count > 0
            return (
              <div
                key={profile.profile_id}
                className="grid gap-4 rounded-lg border border-slate-200 p-4"
              >
                <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_12rem_auto_auto]">
                  <div className="grid gap-1.5">
                    <Label>{t.samlProfileNameLabel}</Label>
                    <Input
                      value={profile.name}
                      disabled={immutable}
                      onChange={(event) =>
                        setProfiles((current) =>
                          current.map((candidate) =>
                            candidate.profile.profile_id === profile.profile_id
                              ? {
                                  ...candidate,
                                  profile: { ...candidate.profile, name: event.target.value },
                                }
                              : candidate,
                          ),
                        )
                      }
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label>{t.samlProfileModeLabel}</Label>
                    <Select
                      value={profile.mode}
                      disabled={immutable}
                      onValueChange={(value) =>
                        setProfiles((current) =>
                          current.map((candidate) =>
                            candidate.profile.profile_id === profile.profile_id
                              ? {
                                  ...candidate,
                                  profile: {
                                    ...candidate.profile,
                                    mode: value as SamlIDPProfileMode,
                                  },
                                }
                              : candidate,
                          ),
                        )
                      }
                      options={[
                        { value: 'shared', label: t.samlProfileSharedLabel },
                        { value: 'dedicated', label: t.samlProfileDedicatedLabel },
                      ]}
                    />
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    className="self-end"
                    disabled={immutable || profileBusy || !profile.name.trim()}
                    onClick={() => void saveProfile(entry)}
                  >
                    {t.samlProfileSave}
                  </Button>
                  <Button
                    type="button"
                    variant="destructive"
                    className="self-end"
                    disabled={deleteDisabled || profileBusy}
                    onClick={() => void removeProfile(profile.profile_id)}
                  >
                    {t.samlProfileDelete}
                  </Button>
                </div>
                <p className="text-xs text-slate-500">
                  {profile.is_default ? `${t.samlProfileDefault}. ` : ''}
                  {t.samlProfileAssignments.replace(
                    '{count}',
                    String(entry.service_provider_count),
                  )}
                  {deleteDisabled && !immutable ? ` ${t.samlProfileInUse}` : ''}
                </p>
                <EndpointField label={t.samlEntityIdLabel} value={entry.entity_id} t={t} />
                <EndpointField label={t.samlMetadataUrlLabel} value={entry.metadata_url} t={t} />
              </div>
            )
          })}
        </div>
      </Card>

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
        fields={[
          { label: t.samlEntityIdLabel, value: catalog.saml.entity_id },
          { label: t.samlMetadataUrlLabel, value: catalog.saml.metadata_url },
          { label: t.samlSsoUrlLabel, value: catalog.saml.sso_url },
          { label: t.samlSloUrlLabel, value: catalog.saml.slo_url },
          { label: t.certificateFingerprintLabel, value: certificate.fingerprint_sha256 },
        ]}
      >
        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
          <p className="text-xs text-slate-600">
            {t.certificateValidity
              .replace('{from}', new Date(certificate.not_before).toLocaleString(dateLocale))
              .replace('{to}', new Date(certificate.not_after).toLocaleString(dateLocale))}
          </p>
          <Button asChild variant="outline" className="mt-3">
            <a href={certificate.download_url} download>
              <IconDownload size={16} aria-hidden="true" />
              {t.downloadCertificate}
            </a>
          </Button>
        </div>
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

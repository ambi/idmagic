import { IconWorldShare } from '@tabler/icons-react'
import { useState } from 'react'
import { createSamlIDPProfile } from '../../api'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import type { AdminSamlIDPProfile } from '../../types'
import {
  CopyableField,
  messageOf,
  nameIdFormatOptions,
  SectionTitle,
} from './AdminApplicationsShared'
import type { AdminApplicationsDictionary } from './AdminApplicationsPage.i18n'

type Props = {
  csrfToken: string
  applicationName: string
  entityID: string
  initialProfiles: AdminSamlIDPProfile[]
  initialProfileID: string
  disabled: boolean
  acs: string
  slo: string
  audience: string
  nameIDFormat: string
  nameIDSource: string
  signAssertion: boolean
  signResponse: boolean
  wantSignedRequests: boolean
  signingCertificate: string
  rulesJSON: string
  onProfileChange: (profileID: string) => void
  onACSChange: (value: string) => void
  onSLOChange: (value: string) => void
  onAudienceChange: (value: string) => void
  onNameIDFormatChange: (value: string) => void
  onNameIDSourceChange: (value: string) => void
  onSignAssertionChange: (value: boolean) => void
  onSignResponseChange: (value: boolean) => void
  onWantSignedRequestsChange: (value: boolean) => void
  onSigningCertificateChange: (value: string) => void
  onRulesJSONChange: (value: string) => void
  onError: (message: string) => void
  t: AdminApplicationsDictionary
}

export function AdminApplicationSamlFormSection(props: Props) {
  const { t } = props
  const [profiles, setProfiles] = useState(props.initialProfiles)
  const [profileID, setProfileID] = useState(props.initialProfileID)
  const [creatingProfile, setCreatingProfile] = useState(false)

  function selectProfile(value: string) {
    setProfileID(value)
    props.onProfileChange(value)
  }

  async function createDedicatedProfile() {
    setCreatingProfile(true)
    props.onError('')
    try {
      const created = await createSamlIDPProfile(props.csrfToken, {
        name: `${props.applicationName} — dedicated`,
        mode: 'dedicated',
      })
      setProfiles((current) => [...current, created])
      selectProfile(created.profile.profile_id)
    } catch (cause) {
      props.onError(messageOf(cause, t.idpProfileCreateFailedError))
    } finally {
      setCreatingProfile(false)
    }
  }

  return (
    <section className="grid gap-4 border-t border-slate-200 pt-5">
      <div className="flex items-center gap-2">
        <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
        <SectionTitle>{t.samlSectionHeading}</SectionTitle>
      </div>
      <CopyableField label={t.entityIdFieldLabel} value={props.entityID} />
      <div className="grid gap-2">
        <Label>{t.idpProfileFieldLabel}</Label>
        <div className="flex flex-wrap items-center gap-2">
          <Select
            value={profileID}
            onValueChange={selectProfile}
            options={profiles.map(({ profile }) => ({
              value: profile.profile_id,
              label: `${profile.name} (${profile.mode === 'shared' ? t.sharedProfileLabel : t.dedicatedProfileLabel})`,
            }))}
            className="min-w-64 flex-1"
          />
          <Button
            type="button"
            variant="outline"
            disabled={props.disabled || creatingProfile}
            onClick={() => void createDedicatedProfile()}
          >
            {t.createDedicatedProfile}
          </Button>
        </div>
        <p className="text-xs text-slate-500">{t.idpProfileHelp}</p>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="edit-saml-acs">{t.acsUrlFieldLabel}</Label>
        <textarea
          id="edit-saml-acs"
          value={props.acs}
          onChange={(event) => props.onACSChange(event.target.value)}
          rows={2}
          className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
          placeholder="https://app.example.com/saml/acs"
        />
        <p className="text-xs text-slate-500">{t.acsUrlHelp}</p>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="edit-saml-slo">{t.sloUrlOptionalFieldLabel}</Label>
        <Input
          id="edit-saml-slo"
          value={props.slo}
          onChange={(event) => props.onSLOChange(event.target.value)}
          className="font-mono text-xs"
          placeholder="https://app.example.com/saml/slo"
        />
      </div>
      <div className="grid gap-1.5">
        <Label>{t.nameIdFormatFieldLabel}</Label>
        <Select
          value={props.nameIDFormat}
          onValueChange={props.onNameIDFormatChange}
          options={nameIdFormatOptions(t)}
          className="w-full"
        />
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid gap-1.5">
          <Label htmlFor="edit-saml-nameid-source">{t.nameIdSourceFieldLabel}</Label>
          <Input
            id="edit-saml-nameid-source"
            value={props.nameIDSource}
            onChange={(event) => props.onNameIDSourceChange(event.target.value)}
            placeholder="sub"
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="edit-saml-audience">{t.audienceOptionalFieldLabel}</Label>
          <Input
            id="edit-saml-audience"
            value={props.audience}
            onChange={(event) => props.onAudienceChange(event.target.value)}
            className="font-mono text-xs"
            placeholder={t.audienceEntityDefault}
          />
        </div>
      </div>
      <div className="grid gap-2.5">
        <BooleanField
          checked={props.signAssertion}
          onChange={props.onSignAssertionChange}
          label={t.signAssertionLabel}
        />
        <BooleanField
          checked={props.signResponse}
          onChange={props.onSignResponseChange}
          label={t.signResponseLabel}
        />
        <BooleanField
          checked={props.wantSignedRequests}
          onChange={props.onWantSignedRequestsChange}
          label={t.wantSignedRequestsLabel}
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="edit-saml-request-signing-cert">{t.requestSigningCertFieldLabel}</Label>
        <textarea
          id="edit-saml-request-signing-cert"
          value={props.signingCertificate}
          onChange={(event) => props.onSigningCertificateChange(event.target.value)}
          rows={7}
          spellCheck={false}
          className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
          placeholder="-----BEGIN CERTIFICATE-----"
        />
        <p className="text-xs text-slate-500">{t.requestSigningCertHelp}</p>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="edit-saml-rules">{t.claimMappingRulesJsonFieldLabel}</Label>
        <textarea
          id="edit-saml-rules"
          value={props.rulesJSON}
          onChange={(event) => props.onRulesJSONChange(event.target.value)}
          rows={8}
          spellCheck={false}
          className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
          placeholder='[{"claim_type":"email","source":"user_attribute","source_key":"email","required":true}]'
        />
        <p className="text-xs text-slate-500">{t.claimMappingRulesHelp}</p>
      </div>
    </section>
  )
}

function BooleanField({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: (value: boolean) => void
  label: string
}) {
  return (
    <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="size-4"
      />
      {label}
    </label>
  )
}

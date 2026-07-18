import { IconArrowLeft, IconKey, IconTrash, IconWorldShare } from '@tabler/icons-react'
import { type FormEvent, useEffect, useRef, useState } from 'react'
import {
  deleteApplicationIcon,
  updateAdminApplication,
  updateApplicationOidcConfig,
  updateApplicationSamlConfig,
  updateApplicationWsFedConfig,
  updateAppSignInPolicy,
  uploadApplicationIcon,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import {
  MAX_APPLICATION_ICON_BYTES,
  safeApplicationIconURL,
  validateApplicationIconFile,
} from '../../lib/applicationIcon'
import { useDictionary } from '../../lib/i18n'
import { AssignmentManager } from './AdminApplicationAssignments'
import { CategoryManager } from './AdminApplicationCategories'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import { ClientSecretRotationPanel } from './ClientSecretRotationPanel'
import {
  appRuleFromInputs,
  CopyableField,
  DEFAULT_NAMEID_FORMAT,
  DEFAULT_NAMEID_SOURCE,
  detailURL,
  initials,
  kindLabel,
  messageOf,
  nameIdFormatOptions,
  parseList,
  ReadonlyMeta,
  SAML_DEFAULT_NAMEID_FORMAT,
  SectionTitle,
  signInRuleWeakerThanDefault,
  signInStrengthOptions,
  statusOptions,
  summarizeSignInRule,
  TOKEN_TYPE_SAML11,
  wsfedTokenTypeOptions,
} from './AdminApplicationsShared'
import type {
  AdminApplicationDetail,
  ApplicationStatus,
  RequiredAuthnStrength,
  SignInRule,
  WsFedClaimMappingRule,
  WsFedTokenType,
} from '../../types'

export function AdminApplicationEditPage({
  csrfToken,
  actorUsername,
  detail,
}: {
  csrfToken: string
  actorUsername?: string
  detail: AdminApplicationDetail
}) {
  const app = detail.application
  const [name, setName] = useState(app.name)
  const [iconFile, setIconFile] = useState<File | null>(null)
  const [iconPreviewURL, setIconPreviewURL] = useState('')
  const [removeIcon, setRemoveIcon] = useState(false)
  const iconSelectionToken = useRef(0)
  const [launchURL, setLaunchURL] = useState(app.launch_url ?? '')
  const [status, setStatus] = useState<ApplicationStatus>(app.status)
  const [redirects, setRedirects] = useState((detail.oidc?.redirect_uris ?? []).join('\n'))
  const [scope, setScope] = useState(detail.oidc?.scope ?? '')
  const [grantTypes, setGrantTypes] = useState((detail.oidc?.grant_types ?? []).join(', '))
  const [responseTypes, setResponseTypes] = useState((detail.oidc?.response_types ?? []).join(', '))
  const [requirePAR, setRequirePAR] = useState(
    detail.oidc?.require_pushed_authorization_requests ?? false,
  )
  const [dpopBound, setDpopBound] = useState(detail.oidc?.dpop_bound_access_tokens ?? false)
  const [replies, setReplies] = useState((detail.wsfed?.reply_urls ?? []).join('\n'))
  const [audience, setAudience] = useState(detail.wsfed?.audience ?? '')
  const [tokenType, setTokenType] = useState<WsFedTokenType>(
    detail.wsfed?.token_type || TOKEN_TYPE_SAML11,
  )
  const [nameIDFormat, setNameIDFormat] = useState(
    detail.wsfed?.name_id_format || DEFAULT_NAMEID_FORMAT,
  )
  const [nameIDSource, setNameIDSource] = useState(
    detail.wsfed?.name_id_source || DEFAULT_NAMEID_SOURCE,
  )
  const [rulesJSON, setRulesJSON] = useState(JSON.stringify(detail.wsfed?.rules ?? [], null, 2))
  const [samlACS, setSamlACS] = useState((detail.saml?.acs_urls ?? []).join('\n'))
  const [samlSLO, setSamlSLO] = useState(detail.saml?.slo_url ?? '')
  const [samlAudience, setSamlAudience] = useState(detail.saml?.audience ?? '')
  const [samlNameIDFormat, setSamlNameIDFormat] = useState(
    detail.saml?.name_id_format || SAML_DEFAULT_NAMEID_FORMAT,
  )
  const [samlNameIDSource, setSamlNameIDSource] = useState(
    detail.saml?.name_id_source || DEFAULT_NAMEID_SOURCE,
  )
  const [samlSignAssertion, setSamlSignAssertion] = useState(detail.saml?.sign_assertion ?? true)
  const [samlSignResponse, setSamlSignResponse] = useState(detail.saml?.sign_response ?? false)
  const [samlWantSignedRequests, setSamlWantSignedRequests] = useState(
    detail.saml?.want_authn_requests_signed ?? false,
  )
  const [samlSigningCert, setSamlSigningCert] = useState(
    detail.saml?.authn_request_signing_certificate_pem ?? '',
  )
  const [samlRulesJSON, setSamlRulesJSON] = useState(
    JSON.stringify(detail.saml?.rules ?? [], null, 2),
  )
  const signInView = detail.sign_in_policy
  const initialSignInRule = signInView?.policy?.rules?.[0]
  const [signInEnabled, setSignInEnabled] = useState(initialSignInRule?.enabled ?? false)
  const [signInStrength, setSignInStrength] = useState<RequiredAuthnStrength>(
    initialSignInRule?.required_authn.strength ?? 'Password',
  )
  const [signInReauthMaxAge, setSignInReauthMaxAge] = useState(
    initialSignInRule?.condition.reauth_max_age_seconds?.toString() ?? '',
  )
  const [signInNetworkCIDRs, setSignInNetworkCIDRs] = useState(
    (initialSignInRule?.condition.network_allow_cidrs ?? []).join('\n'),
  )
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminApplicationsDictionary)
  const nameInvalid = name.trim() === ''
  async function selectIconFile(file: File | null) {
    const token = ++iconSelectionToken.current
    if (!file) {
      setError('')
      setIconFile(null)
      setRemoveIcon(false)
      return
    }
    const validationError = await validateApplicationIconFile(file)
    if (token !== iconSelectionToken.current) return
    if (validationError) {
      setIconFile(null)
      setRemoveIcon(false)
      setError(validationError === 'too-large' ? t.iconTooLargeError : t.iconInvalidTypeError)
      return
    }
    setError('')
    setIconFile(file)
    setRemoveIcon(false)
  }
  useEffect(() => {
    if (!iconFile) {
      setIconPreviewURL('')
      return
    }
    const url = URL.createObjectURL(iconFile)
    setIconPreviewURL(url)
    return () => URL.revokeObjectURL(url)
  }, [iconFile])

  const iconPreview = iconPreviewURL || (removeIcon ? '' : safeApplicationIconURL(app.icon_url))

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (nameInvalid) return
    setSaving(true)
    setError('')
    try {
      const metaPatch: Record<string, unknown> = {}
      if (name.trim() !== app.name) metaPatch.name = name.trim()
      if (app.kind !== 'service' && launchURL.trim() !== (app.launch_url ?? '')) {
        metaPatch.launch_url = launchURL.trim()
      }
      if (status !== app.status) metaPatch.status = status
      if (Object.keys(metaPatch).length > 0) {
        await updateAdminApplication(csrfToken, app.application_id, metaPatch)
      }
      if (removeIcon && app.icon_object_key) {
        await deleteApplicationIcon(csrfToken, app.application_id)
      }
      if (iconFile) {
        if (iconFile.size > MAX_APPLICATION_ICON_BYTES) {
          setError(t.iconTooLargeError)
          setSaving(false)
          return
        }
        await uploadApplicationIcon(csrfToken, app.application_id, iconFile)
      }
      if (detail.oidc) {
        const nextRedirects = parseList(redirects)
        const nextGrants = parseList(grantTypes)
        const nextResponses = parseList(responseTypes)
        const redirectsChanged =
          app.kind !== 'service' && nextRedirects.join(',') !== detail.oidc.redirect_uris.join(',')
        const scopeChanged = scope.trim() !== detail.oidc.scope
        const grantsChanged = nextGrants.join(',') !== detail.oidc.grant_types.join(',')
        const responsesChanged = nextResponses.join(',') !== detail.oidc.response_types.join(',')
        const parChanged = requirePAR !== detail.oidc.require_pushed_authorization_requests
        const dpopChanged = dpopBound !== detail.oidc.dpop_bound_access_tokens
        if (
          redirectsChanged ||
          scopeChanged ||
          grantsChanged ||
          responsesChanged ||
          parChanged ||
          dpopChanged
        ) {
          await updateApplicationOidcConfig(csrfToken, app.application_id, {
            redirect_uris: redirectsChanged ? nextRedirects : undefined,
            scope: scopeChanged ? scope.trim() : undefined,
            grant_types: grantsChanged ? nextGrants : undefined,
            response_types: responsesChanged ? nextResponses : undefined,
            require_pushed_authorization_requests: parChanged ? requirePAR : undefined,
            dpop_bound_access_tokens: dpopChanged ? dpopBound : undefined,
          })
        }
      }
      if (detail.wsfed) {
        let nextRules: WsFedClaimMappingRule[]
        try {
          const parsed = JSON.parse(rulesJSON || '[]')
          if (!Array.isArray(parsed)) throw new Error('not an array')
          nextRules = parsed
        } catch {
          setError(t.invalidClaimRulesJsonError)
          setSaving(false)
          return
        }
        const nextReplies = parseList(replies)
        const changed =
          nextReplies.join(',') !== detail.wsfed.reply_urls.join(',') ||
          audience.trim() !== detail.wsfed.audience ||
          tokenType !== detail.wsfed.token_type ||
          nameIDFormat !== detail.wsfed.name_id_format ||
          nameIDSource.trim() !== detail.wsfed.name_id_source ||
          JSON.stringify(nextRules) !== JSON.stringify(detail.wsfed.rules ?? [])
        if (changed) {
          await updateApplicationWsFedConfig(csrfToken, app.application_id, {
            reply_urls: nextReplies,
            audience: audience.trim(),
            token_type: tokenType,
            name_id_format: nameIDFormat,
            name_id_source: nameIDSource.trim(),
            rules: nextRules,
          })
        }
      }
      if (detail.saml) {
        let nextRules: WsFedClaimMappingRule[]
        try {
          const parsed = JSON.parse(samlRulesJSON || '[]')
          if (!Array.isArray(parsed)) throw new Error('not an array')
          nextRules = parsed
        } catch {
          setError(t.invalidClaimRulesJsonError)
          setSaving(false)
          return
        }
        const nextACS = parseList(samlACS)
        if (samlWantSignedRequests && samlSigningCert.trim() === '') {
          setError(t.signingCertRequiredError)
          setSaving(false)
          return
        }
        const changed =
          nextACS.join(',') !== detail.saml.acs_urls.join(',') ||
          samlSLO.trim() !== detail.saml.slo_url ||
          samlAudience.trim() !== detail.saml.audience ||
          samlNameIDFormat !== detail.saml.name_id_format ||
          samlNameIDSource.trim() !== detail.saml.name_id_source ||
          samlSignAssertion !== detail.saml.sign_assertion ||
          samlSignResponse !== detail.saml.sign_response ||
          samlWantSignedRequests !== (detail.saml.want_authn_requests_signed ?? false) ||
          samlSigningCert.trim() !== (detail.saml.authn_request_signing_certificate_pem ?? '') ||
          JSON.stringify(nextRules) !== JSON.stringify(detail.saml.rules ?? [])
        if (changed) {
          if (nextACS.length === 0) {
            setError(t.acsUrlRequiredError)
            setSaving(false)
            return
          }
          await updateApplicationSamlConfig(csrfToken, app.application_id, {
            acs_urls: nextACS,
            slo_url: samlSLO.trim(),
            audience: samlAudience.trim(),
            name_id_format: samlNameIDFormat,
            name_id_source: samlNameIDSource.trim(),
            sign_assertion: samlSignAssertion,
            sign_response: samlSignResponse,
            want_authn_requests_signed: samlWantSignedRequests,
            authn_request_signing_certificate_pem: samlSigningCert.trim(),
            rules: nextRules,
          })
        }
      }
      const reauthText = signInReauthMaxAge.trim()
      const reauthMaxAge = reauthText === '' ? undefined : Number.parseInt(reauthText, 10)
      if (
        reauthText !== '' &&
        (reauthMaxAge === undefined || !Number.isFinite(reauthMaxAge) || reauthMaxAge <= 0)
      ) {
        setError(t.reauthPositiveIntegerError)
        setSaving(false)
        return
      }
      const networkCIDRs = signInNetworkCIDRs
        .split('\n')
        .map((entry) => entry.trim())
        .filter((entry) => entry !== '')
      const nextSignInRules: SignInRule[] = signInEnabled
        ? [
            {
              rule_id: initialSignInRule?.rule_id ?? '',
              name: 'app-override',
              enabled: true,
              required_authn: {
                strength: signInStrength,
              },
              condition: {
                reauth_max_age_seconds: reauthMaxAge,
                network_allow_cidrs: networkCIDRs.length > 0 ? networkCIDRs : undefined,
              },
            },
          ]
        : []
      const prevSignInRules = signInView?.policy?.rules ?? []
      if (JSON.stringify(nextSignInRules) !== JSON.stringify(prevSignInRules)) {
        await updateAppSignInPolicy(csrfToken, app.application_id, nextSignInRules)
      }
      window.location.assign(detailURL(app.application_id))
    } catch (cause) {
      setError(messageOf(cause, t.applicationUpdateFailedError))
      setSaving(false)
    }
  }
  return (
    <AdminShell
      active="applications"
      actorUsername={actorUsername}
      title={t.editTitle.replace('{name}', app.name)}
      description={kindLabel(app, t)}
      actions={
        <a
          href={detailURL(app.application_id)}
          className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToDetail}
        </a>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="grid max-w-3xl gap-6">
        <Card className="p-6">
          <form onSubmit={submit} className="grid gap-6">
            <section className="grid gap-4">
              <SectionTitle>{t.basicInfoHeading}</SectionTitle>
              <div className="grid gap-1.5">
                <Label htmlFor="edit-name">{t.nameFieldLabel}</Label>
                <Input
                  id="edit-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  aria-invalid={nameInvalid}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-icon-file">{t.iconImageFieldLabel}</Label>
                <fieldset
                  className="flex items-center gap-3 rounded-lg border border-dashed border-slate-300 p-3"
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={(event) => {
                    event.preventDefault()
                    void selectIconFile(event.dataTransfer.files?.[0] ?? null)
                  }}
                >
                  {iconPreview ? (
                    <img
                      src={iconPreview}
                      alt=""
                      className="size-14 rounded-lg border border-slate-200 object-cover"
                    />
                  ) : (
                    <span className="flex size-14 items-center justify-center rounded-lg border border-blue-100 bg-blue-50 text-sm font-bold text-blue-700">
                      {initials(name)}
                    </span>
                  )}
                  <div className="grid flex-1 gap-2 sm:flex sm:items-center">
                    <Input
                      id="edit-icon-file"
                      type="file"
                      accept="image/png,image/jpeg,image/webp,image/gif"
                      onChange={(e) => {
                        void selectIconFile(e.target.files?.[0] ?? null)
                      }}
                    />
                    {app.icon_object_key || iconFile ? (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => {
                          setIconFile(null)
                          setRemoveIcon(true)
                        }}
                      >
                        <IconTrash size={16} aria-hidden="true" />
                        {t.delete}
                      </Button>
                    ) : null}
                  </div>
                </fieldset>
                <p className="text-xs text-slate-500">{t.iconHelpText}</p>
              </div>
              {app.kind !== 'service' ? (
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-launch">{t.launchUrlFieldLabel}</Label>
                  <Input
                    id="edit-launch"
                    value={launchURL}
                    onChange={(e) => setLaunchURL(e.target.value)}
                    placeholder="https://app.example.com/launch"
                  />
                </div>
              ) : null}
              <div className="grid gap-1.5">
                <Label>{t.statusFieldLabel}</Label>
                <Select
                  value={status}
                  onValueChange={(v) => setStatus(v as ApplicationStatus)}
                  options={statusOptions(t)}
                  className="w-40"
                />
              </div>
            </section>
            {detail.oidc ? (
              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <div className="flex items-center gap-2">
                  <IconKey size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>
                    {app.kind === 'service' ? t.serviceKindSectionHeading : t.oidcSectionHeading}
                  </SectionTitle>
                </div>
                <CopyableField label={t.clientIdFieldLabel} value={detail.oidc.client_id} />
                {detail.oidc.client_secret_rotatable ? (
                  <ClientSecretRotationPanel
                    applicationID={app.application_id}
                    csrfToken={csrfToken}
                    onError={setError}
                  />
                ) : null}
                {app.kind !== 'service' ? (
                  <div className="grid gap-1.5">
                    <Label htmlFor="edit-redirects">{t.redirectUriFieldLabel}</Label>
                    <textarea
                      id="edit-redirects"
                      value={redirects}
                      onChange={(e) => setRedirects(e.target.value)}
                      rows={3}
                      className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                      placeholder="https://app.example.com/callback"
                    />
                    <p className="text-xs text-slate-500">{t.redirectUriHelp}</p>
                  </div>
                ) : null}
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-scope">{t.scopeFieldLabel}</Label>
                  <Input
                    id="edit-scope"
                    value={scope}
                    onChange={(e) => setScope(e.target.value)}
                    className="font-mono text-xs"
                    placeholder="openid profile email"
                  />
                </div>
                {app.kind !== 'service' ? (
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="grid gap-1.5">
                      <Label htmlFor="edit-grant-types">{t.grantTypesFieldLabel}</Label>
                      <Input
                        id="edit-grant-types"
                        value={grantTypes}
                        onChange={(e) => setGrantTypes(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="authorization_code, refresh_token"
                      />
                      <p className="text-xs text-slate-500">{t.grantTypesHelp}</p>
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="edit-response-types">{t.responseTypesFieldLabel}</Label>
                      <Input
                        id="edit-response-types"
                        value={responseTypes}
                        onChange={(e) => setResponseTypes(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="code"
                      />
                      <p className="text-xs text-slate-500">{t.responseTypesHelp}</p>
                    </div>
                  </div>
                ) : null}
                <div className="grid gap-2.5">
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={requirePAR}
                      onChange={(e) => setRequirePAR(e.target.checked)}
                      className="size-4"
                    />
                    {t.requirePARLabel}
                  </label>
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={dpopBound}
                      onChange={(e) => setDpopBound(e.target.checked)}
                      className="size-4"
                    />
                    {t.requireDpopLabel}
                  </label>
                </div>
                <div className="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs sm:grid-cols-3">
                  <ReadonlyMeta label={t.clientTypeMetaLabel} value={detail.oidc.client_type} />
                  <ReadonlyMeta
                    label={t.authMethodMetaLabel}
                    value={detail.oidc.token_endpoint_auth_method}
                  />
                  <ReadonlyMeta label={t.fapiProfileMetaLabel} value={detail.oidc.fapi_profile} />
                </div>
              </section>
            ) : null}
            {detail.wsfed ? (
              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <div className="flex items-center gap-2">
                  <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.wsFedSectionHeading}</SectionTitle>
                </div>
                <CopyableField label={t.wtrealmFieldLabel} value={detail.wsfed.wtrealm} />
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-replies">{t.replyUrlFieldLabel}</Label>
                  <textarea
                    id="edit-replies"
                    value={replies}
                    onChange={(e) => setReplies(e.target.value)}
                    rows={2}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                    placeholder="https://app.example.com/wsfed"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label>{t.nameIdFormatFieldLabel}</Label>
                  <Select
                    value={nameIDFormat}
                    onValueChange={setNameIDFormat}
                    options={nameIdFormatOptions(t)}
                    className="w-full"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-nameid-source">{t.nameIdSourceFieldLabel}</Label>
                  <Input
                    id="edit-nameid-source"
                    value={nameIDSource}
                    onChange={(e) => setNameIDSource(e.target.value)}
                    placeholder="sub"
                  />
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="grid gap-1.5">
                    <Label htmlFor="edit-audience">{t.audienceOptionalFieldLabel}</Label>
                    <Input
                      id="edit-audience"
                      value={audience}
                      onChange={(e) => setAudience(e.target.value)}
                      className="font-mono text-xs"
                      placeholder={t.audiencePlaceholderDefault}
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label>{t.tokenTypeSamlVersionFieldLabel}</Label>
                    <Select
                      value={tokenType}
                      onValueChange={(v) => setTokenType(v as WsFedTokenType)}
                      options={wsfedTokenTypeOptions(t)}
                      className="w-full"
                    />
                  </div>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-wsfed-rules">{t.claimMappingRulesJsonFieldLabel}</Label>
                  <textarea
                    id="edit-wsfed-rules"
                    value={rulesJSON}
                    onChange={(e) => setRulesJSON(e.target.value)}
                    rows={8}
                    spellCheck={false}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                    placeholder='[{"claim_type":"http://schemas.xmlsoap.org/claims/UPN","source":"user_attribute","source_key":"preferred_username","required":true}]'
                  />
                  <p className="text-xs text-slate-500">{t.claimMappingRulesHelp}</p>
                </div>
              </section>
            ) : null}
            {detail.saml ? (
              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <div className="flex items-center gap-2">
                  <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.samlSectionHeading}</SectionTitle>
                </div>
                <CopyableField label={t.entityIdFieldLabel} value={detail.saml.entity_id} />
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-saml-acs">{t.acsUrlFieldLabel}</Label>
                  <textarea
                    id="edit-saml-acs"
                    value={samlACS}
                    onChange={(e) => setSamlACS(e.target.value)}
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
                    value={samlSLO}
                    onChange={(e) => setSamlSLO(e.target.value)}
                    className="font-mono text-xs"
                    placeholder="https://app.example.com/saml/slo"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label>{t.nameIdFormatFieldLabel}</Label>
                  <Select
                    value={samlNameIDFormat}
                    onValueChange={setSamlNameIDFormat}
                    options={nameIdFormatOptions(t)}
                    className="w-full"
                  />
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="grid gap-1.5">
                    <Label htmlFor="edit-saml-nameid-source">{t.nameIdSourceFieldLabel}</Label>
                    <Input
                      id="edit-saml-nameid-source"
                      value={samlNameIDSource}
                      onChange={(e) => setSamlNameIDSource(e.target.value)}
                      placeholder="sub"
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="edit-saml-audience">{t.audienceOptionalFieldLabel}</Label>
                    <Input
                      id="edit-saml-audience"
                      value={samlAudience}
                      onChange={(e) => setSamlAudience(e.target.value)}
                      className="font-mono text-xs"
                      placeholder={t.audienceEntityDefault}
                    />
                  </div>
                </div>
                <div className="grid gap-2.5">
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={samlSignAssertion}
                      onChange={(e) => setSamlSignAssertion(e.target.checked)}
                      className="size-4"
                    />
                    {t.signAssertionLabel}
                  </label>
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={samlSignResponse}
                      onChange={(e) => setSamlSignResponse(e.target.checked)}
                      className="size-4"
                    />
                    {t.signResponseLabel}
                  </label>
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={samlWantSignedRequests}
                      onChange={(e) => setSamlWantSignedRequests(e.target.checked)}
                      className="size-4"
                    />
                    {t.wantSignedRequestsLabel}
                  </label>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="edit-saml-request-signing-cert">
                    {t.requestSigningCertFieldLabel}
                  </Label>
                  <textarea
                    id="edit-saml-request-signing-cert"
                    value={samlSigningCert}
                    onChange={(e) => setSamlSigningCert(e.target.value)}
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
                    value={samlRulesJSON}
                    onChange={(e) => setSamlRulesJSON(e.target.value)}
                    rows={8}
                    spellCheck={false}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                    placeholder='[{"claim_type":"email","source":"user_attribute","source_key":"email","required":true}]'
                  />
                  <p className="text-xs text-slate-500">{t.claimMappingRulesHelp}</p>
                </div>
              </section>
            ) : null}

            {app.kind !== 'service' ? (
              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <div className="flex items-center gap-2">
                  <IconKey size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.signInPolicySectionHeading}</SectionTitle>
                </div>
                <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                  <input
                    type="checkbox"
                    checked={signInEnabled}
                    onChange={(e) => setSignInEnabled(e.target.checked)}
                    className="size-4"
                  />
                  {t.overrideTenantDefaultLabel}
                </label>
                {signInEnabled ? (
                  <div className="grid gap-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
                    <div className="grid gap-1.5">
                      <Label>{t.requiredAuthnStrengthFieldLabel}</Label>
                      <Select
                        value={signInStrength}
                        onValueChange={(value) => setSignInStrength(value as RequiredAuthnStrength)}
                        options={signInStrengthOptions(t)}
                        className="w-full"
                      />
                      <p className="text-xs text-slate-500">{t.mfaStepUpHelp}</p>
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="edit-sign-in-reauth">{t.reauthSecondsFieldLabel}</Label>
                      <Input
                        id="edit-sign-in-reauth"
                        type="number"
                        min="1"
                        value={signInReauthMaxAge}
                        onChange={(e) => setSignInReauthMaxAge(e.target.value)}
                        placeholder={t.reauthSecondsPlaceholder}
                      />
                      <p className="text-xs text-slate-500">{t.reauthSecondsHelp}</p>
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="edit-sign-in-cidrs">{t.allowedNetworksFieldLabel}</Label>
                      <textarea
                        id="edit-sign-in-cidrs"
                        value={signInNetworkCIDRs}
                        onChange={(e) => setSignInNetworkCIDRs(e.target.value)}
                        rows={3}
                        spellCheck={false}
                        className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                        placeholder={'10.0.0.0/8\n192.168.1.0/24'}
                      />
                      <p className="text-xs text-slate-500">{t.allowedNetworksHelp}</p>
                    </div>
                  </div>
                ) : null}
                {(() => {
                  const defaultRules = signInView?.tenant_default?.rules ?? []
                  const appRule = appRuleFromInputs(
                    signInStrength,
                    signInReauthMaxAge,
                    signInNetworkCIDRs,
                  )
                  const effectiveSummary = signInEnabled
                    ? summarizeSignInRule(appRule, t)
                    : defaultRules
                        .filter((r) => r.enabled)
                        .map((rule) => summarizeSignInRule(rule, t))
                        .join('、') || t.noAdditionalRequirementsNotice
                  const weaker = signInEnabled && signInRuleWeakerThanDefault(appRule, defaultRules)
                  return (
                    <div className="grid gap-3 rounded-lg border border-slate-200 bg-white p-4">
                      <div className="grid gap-1">
                        <p className="text-xs font-semibold text-slate-500">
                          {t.tenantDefaultLabel}
                        </p>
                        {defaultRules.filter((r) => r.enabled).length > 0 ? (
                          <ul className="grid gap-1 text-xs text-slate-600">
                            {defaultRules
                              .filter((r) => r.enabled)
                              .map((rule) => (
                                <li key={rule.rule_id}>{summarizeSignInRule(rule, t)}</li>
                              ))}
                          </ul>
                        ) : (
                          <p className="text-xs text-slate-500">
                            {t.noAdditionalRequirementsNotice}
                          </p>
                        )}
                      </div>
                      <div className="grid gap-1">
                        <p className="text-xs font-semibold text-slate-500">
                          {t.effectivePolicyLabel}
                        </p>
                        <p className="text-xs text-slate-600">{effectiveSummary}</p>
                        <p className="text-xs text-slate-400">
                          {signInEnabled ? t.overrideAppliedNotice : t.tenantDefaultAppliedNotice}
                          {t.savedAfterConfirmSuffix}
                        </p>
                      </div>
                      {weaker ? (
                        <Alert variant="destructive">{t.weakerThanDefaultWarning}</Alert>
                      ) : null}
                    </div>
                  )
                })()}
              </section>
            ) : null}

            <div className="flex justify-end gap-2 border-t border-slate-200 pt-5">
              <Button asChild variant="outline">
                <a href={detailURL(app.application_id)}>{t.cancel}</a>
              </Button>
              <Button type="submit" disabled={saving || nameInvalid}>
                {saving ? t.saving : t.save}
              </Button>
            </div>
          </form>
        </Card>

        {app.kind !== 'service' ? (
          <Card className="p-6">
            <AssignmentManager
              appID={app.application_id}
              csrfToken={csrfToken}
              onError={setError}
            />
          </Card>
        ) : null}

        {app.kind !== 'service' ? (
          <Card className="p-6">
            <CategoryManager app={app} csrfToken={csrfToken} onError={setError} />
          </Card>
        ) : null}
      </div>
    </AdminShell>
  )
}

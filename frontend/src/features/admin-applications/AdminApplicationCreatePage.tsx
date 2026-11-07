import { IconArrowLeft, IconCheck } from '@tabler/icons-react'
import { type FormEvent, useMemo, useState } from 'react'
import { createAdminApplication } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import {
  formSchema,
  optionalUrl,
  pemText,
  requiredText,
  requiredUrl,
  scopeText,
  urlList,
  useFormValidation,
  when,
} from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import {
  AUTH_METHODS,
  type AppType,
  appTypeOptions,
  CopyableField,
  DEFAULT_NAMEID_FORMAT,
  DEFAULT_NAMEID_SOURCE,
  detailURL,
  listURL,
  messageOf,
  nameIdFormatOptions,
  parseList,
  SAML_DEFAULT_NAMEID_FORMAT,
  SectionTitle,
  SourceAttributeSelect,
} from './AdminApplicationsShared'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'

// AdminApplicationCreatePage はアプリケーション登録の専用画面 (wi-314 T013)。従来
// CreateApplicationDialog がモーダルとして担っていたものを専用ルートへ移す。
export function AdminApplicationCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = listURL()
  const [type, setType] = useState<AppType>('oidc')
  const [name, setName] = useState('')
  const [launchURL, setLaunchURL] = useState('')
  const [redirectURIs, setRedirectURIs] = useState('')
  const [scope, setScope] = useState('')
  const [clientType, setClientType] = useState<'confidential' | 'public'>('confidential')
  const [authMethod, setAuthMethod] = useState('client_secret_basic')
  const [jwksURI, setJwksURI] = useState('')
  const [tlsSubjectDN, setTlsSubjectDN] = useState('')
  const [wtrealm, setWtrealm] = useState('')
  const [replyURLs, setReplyURLs] = useState('')
  const [nameIDFormat, setNameIDFormat] = useState(DEFAULT_NAMEID_FORMAT)
  const [nameIDSource, setNameIDSource] = useState(DEFAULT_NAMEID_SOURCE)
  const [samlEntityID, setSamlEntityID] = useState('')
  const [samlACSURLs, setSamlACSURLs] = useState('')
  const [samlSLOURL, setSamlSLOURL] = useState('')
  const [samlNameIDFormat, setSamlNameIDFormat] = useState(SAML_DEFAULT_NAMEID_FORMAT)
  const [samlNameIDSource, setSamlNameIDSource] = useState(DEFAULT_NAMEID_SOURCE)
  const [samlSignResponse, setSamlSignResponse] = useState(false)
  const [samlWantSignedRequests, setSamlWantSignedRequests] = useState(false)
  const [samlSigningCert, setSamlSigningCert] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [secret, setSecret] = useState<{ clientID: string; clientSecret: string } | null>(null)
  const [createdID, setCreatedID] = useState('')
  const t = useDictionary(adminApplicationsDictionary)

  const schema = useMemo(() => {
    const oidc = type === 'oidc'
    const saml = type === 'saml'
    return formSchema({
      name: requiredText(),
      launch_url: type === 'weblink' ? requiredUrl() : optionalUrl(),
      redirect_uris: when(oidc, urlList(parseList, { required: true })),
      scope: when(oidc || type === 'service', scopeText()),
      jwks_uri: when(oidc && authMethod === 'private_key_jwt', requiredUrl({ https: true })),
      tls_client_auth_subject_dn: when(
        oidc && authMethod === 'tls_client_auth',
        requiredText(LENGTH.uri),
      ),
      wtrealm: when(type === 'wsfed', requiredText(LENGTH.uri)),
      reply_urls: when(type === 'wsfed', urlList(parseList, { required: true })),
      entity_id: when(saml, requiredText(LENGTH.uri)),
      acs_urls: when(saml, urlList(parseList, { required: true })),
      slo_url: when(saml, optionalUrl({}, LENGTH.uri)),
      authn_request_signing_certificate_pem: when(saml, pemText(samlWantSignedRequests)),
    })
  }, [type, authMethod, samlWantSignedRequests])
  const form = useFormValidation(schema, {
    name,
    launch_url: launchURL,
    redirect_uris: redirectURIs,
    scope,
    jwks_uri: jwksURI,
    tls_client_auth_subject_dn: tlsSubjectDN,
    wtrealm,
    reply_urls: replyURLs,
    entity_id: samlEntityID,
    acs_urls: samlACSURLs,
    slo_url: samlSLOURL,
    authn_request_signing_certificate_pem: samlSigningCert,
  })

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!form.data) {
      form.revealAll()
      return
    }
    setSaving(true)
    setError('')
    try {
      const result = await createAdminApplication(csrfToken, {
        ...form.data,
        type,
        client_type: type === 'oidc' ? clientType : undefined,
        token_endpoint_auth_method: type === 'oidc' ? authMethod : undefined,
        name_id_format:
          type === 'wsfed' ? nameIDFormat : type === 'saml' ? samlNameIDFormat : undefined,
        name_id_source:
          type === 'wsfed'
            ? nameIDSource.trim()
            : type === 'saml'
              ? samlNameIDSource.trim()
              : undefined,
        sign_response: type === 'saml' ? samlSignResponse : undefined,
        want_authn_requests_signed: type === 'saml' ? samlWantSignedRequests : undefined,
      })
      const id = result.application.id
      if (result.client_secret && result.client_id) {
        // OIDC / サービスは client_secret を一度だけ提示し、確認後に詳細へ遷移する。
        setCreatedID(id)
        setSecret({ clientID: result.client_id, clientSecret: result.client_secret })
        return
      }
      window.location.assign(detailURL(id))
    } catch (cause) {
      setError(messageOf(cause, t.applicationCreateFailedError))
    } finally {
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="applications"
      actorUsername={actorUsername}
      title={secret ? t.createdHeading : t.addApplicationHeading}
      description={t.applicationEyebrow}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listPath} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      <Card className="w-full admin-form overflow-hidden">
        {secret ? (
          <div className="grid gap-4 p-6">
            <Alert variant="success">
              {t.clientCreatedNotice}
              <strong>{t.clientSecretVisibleOnceStrong}</strong>
              {t.clientSecretVisibleOnceSuffix}
            </Alert>
            <CopyableField label={t.clientIdFieldLabel} value={secret.clientID} />
            <CopyableField label={t.clientSecretFieldLabel} value={secret.clientSecret} />
            <div className="flex justify-end">
              <Button type="button" onClick={() => window.location.assign(detailURL(createdID))}>
                <IconCheck size={16} aria-hidden="true" />
                {t.storedConfirm}
              </Button>
            </div>
          </div>
        ) : (
          <form onSubmit={submit} className="flex flex-col">
            <div className="grid gap-6 p-6">
              {error ? <Alert variant="destructive">{error}</Alert> : null}

              <section className="grid gap-2">
                <SectionTitle>{t.typeSectionHeading}</SectionTitle>
                <div className="grid gap-2 sm:grid-cols-2">
                  {appTypeOptions(t).map((option) => {
                    const Icon = option.icon
                    const active = type === option.type
                    return (
                      <button
                        key={option.type}
                        type="button"
                        onClick={() => setType(option.type)}
                        className={`grid gap-1.5 rounded-xl border p-3 text-left transition ${
                          active
                            ? 'border-blue-500 bg-blue-50/60 ring-2 ring-blue-500/20'
                            : 'border-slate-200 hover:border-slate-300'
                        }`}
                      >
                        <Icon
                          size={20}
                          className={active ? 'text-blue-600' : 'text-slate-400'}
                          aria-hidden="true"
                        />
                        <span className="text-sm font-semibold text-slate-900">{option.label}</span>
                        <span className="text-xs leading-snug text-slate-500">
                          {option.description}
                        </span>
                      </button>
                    )
                  })}
                </div>
              </section>

              <section className="grid gap-4 border-t border-slate-200 pt-5">
                <SectionTitle>{t.basicInfoHeading}</SectionTitle>
                <div className="grid gap-1.5">
                  <Label htmlFor="app-name" required>
                    {t.nameFieldLabel}
                  </Label>
                  <Input
                    id="app-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                    placeholder="Payroll"
                    {...form.fieldProps('name')}
                  />
                  <FieldError form={form} name="name" />
                </div>
                {type !== 'service' ? (
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-launch" optional>
                      {type === 'weblink' ? t.linkUrlFieldLabel : t.launchUrlOptionalFieldLabel}
                    </Label>
                    <Input
                      id="app-launch"
                      value={launchURL}
                      onChange={(e) => setLaunchURL(e.target.value)}
                      placeholder="https://app.example.com"
                      required={type === 'weblink'}
                      {...form.fieldProps('launch_url')}
                    />
                    <FieldError form={form} name="launch_url" />
                    {type !== 'weblink' ? (
                      <p className="text-xs text-slate-500">{t.launchUrlHelp}</p>
                    ) : null}
                  </div>
                ) : null}
              </section>

              {type === 'service' ? (
                <section className="grid gap-4 border-t border-slate-200 pt-5">
                  <SectionTitle>{t.serviceKindSectionHeading}</SectionTitle>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-scope" optional>
                      {t.scopeOptionalFieldLabel}
                    </Label>
                    <Input
                      id="app-scope"
                      value={scope}
                      onChange={(e) => setScope(e.target.value)}
                      className="font-mono text-xs"
                      placeholder="catalog:read invoice:read"
                      {...form.fieldProps('scope')}
                    />
                    <FieldError form={form} name="scope" />
                    <p className="text-xs text-slate-500">{t.serviceScopeHelp}</p>
                  </div>
                </section>
              ) : null}

              {type === 'oidc' ? (
                <section className="grid gap-4 border-t border-slate-200 pt-5">
                  <SectionTitle>{t.oidcSectionHeading}</SectionTitle>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-redirects" required>
                      {t.redirectUriFieldLabel}
                    </Label>
                    <textarea
                      id="app-redirects"
                      value={redirectURIs}
                      onChange={(e) => setRedirectURIs(e.target.value)}
                      rows={3}
                      required
                      className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                      placeholder="https://app.example.com/callback"
                      {...form.fieldProps('redirect_uris')}
                    />
                    <FieldError form={form} name="redirect_uris" />
                    <p className="text-xs text-slate-500">{t.oidcRedirectHelp}</p>
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-oidc-scope" optional>
                      {t.scopeOptionalFieldLabel}
                    </Label>
                    <Input
                      id="app-oidc-scope"
                      value={scope}
                      onChange={(e) => setScope(e.target.value)}
                      className="font-mono text-xs"
                      placeholder="openid profile email"
                      {...form.fieldProps('scope')}
                    />
                    <FieldError form={form} name="scope" />
                  </div>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="grid gap-1.5">
                      <Label>{t.clientTypeFieldLabel}</Label>
                      <Select
                        value={clientType}
                        onValueChange={(v) => {
                          const next = v as 'confidential' | 'public'
                          setClientType(next)
                          setAuthMethod(next === 'public' ? 'none' : 'client_secret_basic')
                        }}
                        options={[
                          { value: 'confidential', label: 'confidential' },
                          { value: 'public', label: 'public' },
                        ]}
                        className="w-full"
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label>{t.authMethodFieldLabel}</Label>
                      <Select
                        value={authMethod}
                        onValueChange={setAuthMethod}
                        options={AUTH_METHODS}
                        className="w-full"
                      />
                    </div>
                  </div>
                  {authMethod === 'private_key_jwt' ? (
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-jwks-uri" required>
                        {t.jwksUriFieldLabel}
                      </Label>
                      <Input
                        id="app-jwks-uri"
                        type="url"
                        value={jwksURI}
                        onChange={(e) => setJwksURI(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="https://app.example.com/jwks.json"
                        required
                        {...form.fieldProps('jwks_uri')}
                      />
                      <FieldError form={form} name="jwks_uri" />
                    </div>
                  ) : null}
                  {authMethod === 'tls_client_auth' ? (
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-tls-dn" required>
                        {t.tlsClientCertSubjectDnFieldLabel}
                      </Label>
                      <Input
                        id="app-tls-dn"
                        value={tlsSubjectDN}
                        onChange={(e) => setTlsSubjectDN(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="CN=app,OU=clients,O=example"
                        required
                        {...form.fieldProps('tls_client_auth_subject_dn')}
                      />
                      <FieldError form={form} name="tls_client_auth_subject_dn" />
                    </div>
                  ) : null}
                  <p className="text-xs text-slate-500">{t.authMethodFixedNotice}</p>
                </section>
              ) : null}

              {type === 'wsfed' ? (
                <section className="grid gap-4 border-t border-slate-200 pt-5">
                  <SectionTitle>{t.wsFedSectionHeading}</SectionTitle>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-wtrealm" required>
                      {t.wtrealmFieldLabel}
                    </Label>
                    <Input
                      id="app-wtrealm"
                      value={wtrealm}
                      onChange={(e) => setWtrealm(e.target.value)}
                      required
                      className="font-mono text-xs"
                      placeholder="urn:app:example"
                      {...form.fieldProps('wtrealm')}
                    />
                    <FieldError form={form} name="wtrealm" />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-replies" required>
                      {t.replyUrlFieldLabel}
                    </Label>
                    <textarea
                      id="app-replies"
                      value={replyURLs}
                      onChange={(e) => setReplyURLs(e.target.value)}
                      rows={2}
                      required
                      className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                      placeholder="https://app.example.com/wsfed"
                      {...form.fieldProps('reply_urls')}
                    />
                    <FieldError form={form} name="reply_urls" />
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
                    <Label htmlFor="app-nameid-source">{t.nameIdSourceFieldLabel}</Label>
                    <SourceAttributeSelect
                      id="app-nameid-source"
                      value={nameIDSource}
                      onChange={setNameIDSource}
                    />
                  </div>
                </section>
              ) : null}

              {type === 'saml' ? (
                <section className="grid gap-4 border-t border-slate-200 pt-5">
                  <SectionTitle>{t.samlSectionHeading}</SectionTitle>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-saml-entity" required>
                      {t.entityIdFieldLabel}
                    </Label>
                    <Input
                      id="app-saml-entity"
                      value={samlEntityID}
                      onChange={(e) => setSamlEntityID(e.target.value)}
                      required
                      className="font-mono text-xs"
                      placeholder="https://app.example.com/saml/metadata"
                      {...form.fieldProps('entity_id')}
                    />
                    <FieldError form={form} name="entity_id" />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-saml-acs" required>
                      {t.acsUrlFieldLabel}
                    </Label>
                    <textarea
                      id="app-saml-acs"
                      value={samlACSURLs}
                      onChange={(e) => setSamlACSURLs(e.target.value)}
                      rows={2}
                      required
                      className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                      placeholder="https://app.example.com/saml/acs"
                      {...form.fieldProps('acs_urls')}
                    />
                    <FieldError form={form} name="acs_urls" />
                    <p className="text-xs text-slate-500">{t.redirectUriHelp}</p>
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-saml-slo" optional>
                      {t.sloUrlOptionalFieldLabel}
                    </Label>
                    <Input
                      id="app-saml-slo"
                      value={samlSLOURL}
                      onChange={(e) => setSamlSLOURL(e.target.value)}
                      className="font-mono text-xs"
                      placeholder="https://app.example.com/saml/slo"
                      {...form.fieldProps('slo_url')}
                    />
                    <FieldError form={form} name="slo_url" />
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
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-saml-nameid-source">{t.nameIdSourceFieldLabel}</Label>
                    <SourceAttributeSelect
                      id="app-saml-nameid-source"
                      value={samlNameIDSource}
                      onChange={setSamlNameIDSource}
                    />
                  </div>
                  <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
                    <input
                      type="checkbox"
                      checked={samlSignResponse}
                      onChange={(e) => setSamlSignResponse(e.target.checked)}
                      className="size-4"
                    />
                    {t.signResponseDefaultLabel}
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
                  <div className="grid gap-1.5">
                    <Label
                      htmlFor="app-saml-request-signing-cert"
                      required={samlWantSignedRequests}
                    >
                      {t.requestSigningCertFieldLabel}
                    </Label>
                    <textarea
                      id="app-saml-request-signing-cert"
                      value={samlSigningCert}
                      onChange={(e) => setSamlSigningCert(e.target.value)}
                      rows={6}
                      spellCheck={false}
                      className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                      placeholder="-----BEGIN CERTIFICATE-----"
                      {...form.fieldProps('authn_request_signing_certificate_pem')}
                    />
                    <FieldError form={form} name="authn_request_signing_certificate_pem" />
                  </div>
                </section>
              ) : null}
            </div>
            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <Button variant="outline" nativeButton={false} render={<a href={listPath} />}>
                {t.cancel}
              </Button>
              <Button type="submit" disabled={saving || !form.data}>
                {saving ? t.creating : t.create}
              </Button>
            </div>
          </form>
        )}
      </Card>
    </AdminShell>
  )
}

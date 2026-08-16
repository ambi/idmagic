import { IconArrowLeft, IconCheck } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { createAdminApplication } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
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

  const nameInvalid = name.trim() === ''

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (nameInvalid) return
    setSaving(true)
    setError('')
    if (type === 'saml' && samlWantSignedRequests && samlSigningCert.trim() === '') {
      setError(t.signingCertRequiredError)
      setSaving(false)
      return
    }
    try {
      const result = await createAdminApplication(csrfToken, {
        name: name.trim(),
        type,
        launch_url: launchURL.trim() || undefined,
        redirect_uris: type === 'oidc' ? parseList(redirectURIs) : undefined,
        scope: type === 'service' || type === 'oidc' ? scope.trim() || undefined : undefined,
        client_type: type === 'oidc' ? clientType : undefined,
        token_endpoint_auth_method: type === 'oidc' ? authMethod : undefined,
        jwks_uri: type === 'oidc' && authMethod === 'private_key_jwt' ? jwksURI.trim() : undefined,
        tls_client_auth_subject_dn:
          type === 'oidc' && authMethod === 'tls_client_auth' ? tlsSubjectDN.trim() : undefined,
        wtrealm: type === 'wsfed' ? wtrealm.trim() : undefined,
        reply_urls: type === 'wsfed' ? parseList(replyURLs) : undefined,
        name_id_format:
          type === 'wsfed' ? nameIDFormat : type === 'saml' ? samlNameIDFormat : undefined,
        name_id_source:
          type === 'wsfed'
            ? nameIDSource.trim()
            : type === 'saml'
              ? samlNameIDSource.trim()
              : undefined,
        entity_id: type === 'saml' ? samlEntityID.trim() : undefined,
        acs_urls: type === 'saml' ? parseList(samlACSURLs) : undefined,
        slo_url: type === 'saml' ? samlSLOURL.trim() || undefined : undefined,
        sign_response: type === 'saml' ? samlSignResponse : undefined,
        want_authn_requests_signed: type === 'saml' ? samlWantSignedRequests : undefined,
        authn_request_signing_certificate_pem:
          type === 'saml' ? samlSigningCert.trim() || undefined : undefined,
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
      <Card className="w-full max-w-2xl overflow-hidden">
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
                  <Label htmlFor="app-name">{t.nameFieldLabel}</Label>
                  <Input
                    id="app-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                    placeholder="Payroll"
                  />
                </div>
                {type !== 'service' ? (
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-launch">
                      {type === 'weblink' ? t.linkUrlFieldLabel : t.launchUrlOptionalFieldLabel}
                    </Label>
                    <Input
                      id="app-launch"
                      value={launchURL}
                      onChange={(e) => setLaunchURL(e.target.value)}
                      placeholder="https://app.example.com"
                      required={type === 'weblink'}
                    />
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
                    <Label htmlFor="app-scope">{t.scopeOptionalFieldLabel}</Label>
                    <Input
                      id="app-scope"
                      value={scope}
                      onChange={(e) => setScope(e.target.value)}
                      className="font-mono text-xs"
                      placeholder="catalog:read invoice:read"
                    />
                    <p className="text-xs text-slate-500">{t.serviceScopeHelp}</p>
                  </div>
                </section>
              ) : null}

              {type === 'oidc' ? (
                <section className="grid gap-4 border-t border-slate-200 pt-5">
                  <SectionTitle>{t.oidcSectionHeading}</SectionTitle>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-redirects">{t.redirectUriFieldLabel}</Label>
                    <textarea
                      id="app-redirects"
                      value={redirectURIs}
                      onChange={(e) => setRedirectURIs(e.target.value)}
                      rows={3}
                      required
                      className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                      placeholder="https://app.example.com/callback"
                    />
                    <p className="text-xs text-slate-500">{t.oidcRedirectHelp}</p>
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-oidc-scope">{t.scopeOptionalFieldLabel}</Label>
                    <Input
                      id="app-oidc-scope"
                      value={scope}
                      onChange={(e) => setScope(e.target.value)}
                      className="font-mono text-xs"
                      placeholder="openid profile email"
                    />
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
                      <Label htmlFor="app-jwks-uri">{t.jwksUriFieldLabel}</Label>
                      <Input
                        id="app-jwks-uri"
                        type="url"
                        value={jwksURI}
                        onChange={(e) => setJwksURI(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="https://app.example.com/jwks.json"
                        required
                      />
                    </div>
                  ) : null}
                  {authMethod === 'tls_client_auth' ? (
                    <div className="grid gap-1.5">
                      <Label htmlFor="app-tls-dn">{t.tlsClientCertSubjectDnFieldLabel}</Label>
                      <Input
                        id="app-tls-dn"
                        value={tlsSubjectDN}
                        onChange={(e) => setTlsSubjectDN(e.target.value)}
                        className="font-mono text-xs"
                        placeholder="CN=app,OU=clients,O=example"
                        required
                      />
                    </div>
                  ) : null}
                  <p className="text-xs text-slate-500">{t.authMethodFixedNotice}</p>
                </section>
              ) : null}

              {type === 'wsfed' ? (
                <section className="grid gap-4 border-t border-slate-200 pt-5">
                  <SectionTitle>{t.wsFedSectionHeading}</SectionTitle>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-wtrealm">{t.wtrealmFieldLabel}</Label>
                    <Input
                      id="app-wtrealm"
                      value={wtrealm}
                      onChange={(e) => setWtrealm(e.target.value)}
                      required
                      maxLength={LENGTH.uri}
                      className="font-mono text-xs"
                      placeholder="urn:app:example"
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-replies">{t.replyUrlFieldLabel}</Label>
                    <textarea
                      id="app-replies"
                      value={replyURLs}
                      onChange={(e) => setReplyURLs(e.target.value)}
                      rows={2}
                      required
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
                    <Label htmlFor="app-saml-entity">{t.entityIdFieldLabel}</Label>
                    <Input
                      id="app-saml-entity"
                      value={samlEntityID}
                      onChange={(e) => setSamlEntityID(e.target.value)}
                      required
                      maxLength={LENGTH.uri}
                      className="font-mono text-xs"
                      placeholder="https://app.example.com/saml/metadata"
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-saml-acs">{t.acsUrlFieldLabel}</Label>
                    <textarea
                      id="app-saml-acs"
                      value={samlACSURLs}
                      onChange={(e) => setSamlACSURLs(e.target.value)}
                      rows={2}
                      required
                      className="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs focus:border-blue-600 focus:outline-none focus:ring-3 focus:ring-blue-600/10"
                      placeholder="https://app.example.com/saml/acs"
                    />
                    <p className="text-xs text-slate-500">{t.redirectUriHelp}</p>
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="app-saml-slo">{t.sloUrlOptionalFieldLabel}</Label>
                    <Input
                      id="app-saml-slo"
                      value={samlSLOURL}
                      onChange={(e) => setSamlSLOURL(e.target.value)}
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
                    <Label htmlFor="app-saml-request-signing-cert">
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
                    />
                  </div>
                </section>
              ) : null}
            </div>
            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <Button variant="outline" nativeButton={false} render={<a href={listPath} />}>
                {t.cancel}
              </Button>
              <Button type="submit" disabled={saving || nameInvalid}>
                {saving ? t.creating : t.create}
              </Button>
            </div>
          </form>
        )}
      </Card>
    </AdminShell>
  )
}

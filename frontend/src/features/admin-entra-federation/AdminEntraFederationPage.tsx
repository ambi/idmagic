import {
  IconArrowLeft,
  IconDownload,
  IconKey,
  IconPlus,
  IconServerBolt,
  IconWorldShare,
} from '@tabler/icons-react'
import { type FormEvent, useMemo, useState } from 'react'
import {
  AuthenticationAPIError,
  configureEntraFederation,
  type EntraFederationDomainSummary,
  listEntraFederationDomains,
  listWsFedRelyingParties,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { FieldError } from '../../components/FieldError'
import {
  formSchema,
  optionalText,
  optionalUrl,
  requiredText,
  rule,
  useFormValidation,
} from '../../lib/form'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary, useLocale } from '../../lib/i18n'
import { localizedErrorMessage } from '../../lib/i18n/errorMessage'
import type { Locale } from '../../lib/i18n/locale'
import type {
  AdminIntegrationEndpointCatalog,
  EntraDomainFederation,
  WsFedRelyingParty,
} from '../../types'
import {
  adminEntraFederationDictionary,
  type AdminEntraFederationDictionary,
} from './AdminEntraFederationPage.i18n'
import { EntraVerificationStatusBadge } from './EntraVerificationStatusBadge'
import { EntraImmutableIdGuide } from './EntraImmutableIdGuide'
import {
  ENTRA_ADD_PATH,
  ENTRA_LIST_PATH,
  entraDomainURL,
  returnToEntraDomain,
  takeEntraListNotice,
} from './entraNavigation'
import { xmlFederationKeyURL } from '../admin-keys/xmlFederationNavigation'

// Entra domain federation は検証済みドメイン単位の操作。個別アプリの設定ではなく、テナント配下の
// ドメインをまとめて外部 IdP へ向ける設定なので、Application 編集とは別のテナント設定画面で扱う。
//
// 一覧はドメインごとの反映の確認の状態を示し、詳細（読み取り専用）、追加、編集を別の画面に分ける。
// 保存しただけのドメインを反映済みとして見せないため、状態は管理者が取り込んだ照合の結果だけから示す。
function endpointLinks(
  t: AdminEntraFederationDictionary,
  endpoints: AdminIntegrationEndpointCatalog['ws_federation'],
) {
  return [
    {
      label: t.federationMetadataLabel,
      href: endpoints.metadata_url,
      value: endpoints.metadata_url,
      icon: IconDownload,
    },
    {
      label: t.wsTrustMexLabel,
      href: endpoints.metadata_exchange_url,
      value: endpoints.metadata_exchange_url,
      icon: IconServerBolt,
    },
    {
      label: t.wsTrustUsernamemixedLabel,
      href: endpoints.active_logon_url,
      value: endpoints.active_logon_url,
      icon: IconServerBolt,
    },
  ]
}

export function AdminEntraFederationPage({
  actorUsername,
  relyingParties,
  domains,
  integrationEndpoints,
}: {
  actorUsername?: string
  relyingParties: WsFedRelyingParty[]
  domains: EntraFederationDomainSummary[]
  integrationEndpoints: AdminIntegrationEndpointCatalog
}) {
  const [items, setItems] = useState(relyingParties.filter((rp) => rp.entra_profile))
  const [summaries, setSummaries] = useState(domains)
  // 解除はドメインの詳細で切り戻しを確かめてから行う。一覧は RP 全体を消す操作を持たない。
  const [notice, setNotice] = useState(takeEntraListNotice)
  const t = useDictionary(adminEntraFederationDictionary)

  return (
    <AdminShell
      active="entra-federation"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      list={{
        displayed: items.length,
        onRefresh: async () => {
          const [rps, refreshed] = await Promise.all([
            listWsFedRelyingParties(),
            listEntraFederationDomains(),
          ])
          setItems(rps.filter((rp) => rp.entra_profile))
          setSummaries(refreshed)
        },
      }}
      actions={
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            nativeButton={false}
            render={<a href={xmlFederationKeyURL()} />}
          >
            <IconKey size={16} aria-hidden="true" />
            {t.rolloverSigningKey}
          </Button>
          <Button nativeButton={false} render={<a href={tenantURL(ENTRA_ADD_PATH)} />}>
            <IconPlus size={16} aria-hidden="true" />
            {t.addDomainFederation}
          </Button>
        </div>
      }
    >
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid gap-2 rounded-md border border-slate-200 bg-slate-50 p-3 sm:grid-cols-3">
        {endpointLinks(t, integrationEndpoints.ws_federation).map(
          ({ label, href, value, icon: Icon }) => (
            <a
              key={value}
              className="flex min-w-0 items-start gap-2 rounded-md border border-slate-200 bg-white p-2.5 text-xs text-slate-600 hover:border-blue-300 hover:text-blue-700"
              href={href}
            >
              <Icon size={16} className="mt-0.5 shrink-0 text-blue-600" aria-hidden="true" />
              <span className="min-w-0">
                <span className="block font-semibold text-slate-800">{label}</span>
                <span className="block truncate font-mono">{value}</span>
              </span>
            </a>
          ),
        )}
      </div>

      <EntraFederationList items={items} summaries={summaries} />
    </AdminShell>
  )
}

export function EntraFederationList({
  items,
  summaries,
}: {
  items: WsFedRelyingParty[]
  summaries: EntraFederationDomainSummary[]
}) {
  const t = useDictionary(adminEntraFederationDictionary)
  if (items.length === 0) {
    return <Card className="p-8 text-center text-sm text-slate-500">{t.noFederationsNotice}</Card>
  }
  const statusOf = (domain: string) =>
    summaries.find((summary) => summary.domain === domain)?.verification.status ?? 'unverified'
  return (
    <div className="flex flex-col gap-3">
      {items.map((relyingParty) => (
        <Card key={relyingParty.wtrealm} className="flex items-start gap-3 p-4">
          <div className="flex min-w-0 items-start gap-2">
            <IconWorldShare
              size={16}
              className="mt-0.5 shrink-0 text-blue-600"
              aria-hidden="true"
            />
            <div className="min-w-0 text-xs leading-5 text-slate-600">
              <p className="truncate font-mono">{relyingParty.wtrealm}</p>
              <ul className="my-1 grid gap-1.5">
                {relyingParty.entra_profile?.domains.map((federated) => (
                  <li
                    key={federated.domain}
                    className="flex min-w-0 flex-wrap items-center gap-x-2"
                    data-domain={federated.domain}
                  >
                    <a
                      className="font-semibold text-blue-700 hover:underline"
                      href={entraDomainURL(federated.domain)}
                      aria-label={t.viewDomainDetail.replace('{domain}', federated.domain)}
                    >
                      {federated.domain}
                    </a>
                    <EntraVerificationStatusBadge status={statusOf(federated.domain)} />
                    <span className="block w-full truncate font-mono">
                      {t.issuerUriPrefix}
                      {federated.issuer_uri}
                    </span>
                  </li>
                ))}
              </ul>
              <p>
                {t.sourceAnchorPrefix}
                {relyingParty.entra_profile?.source_anchor_attribute}
              </p>
            </div>
          </div>
        </Card>
      ))}
    </div>
  )
}

const ENTRA_TENANT_ID =
  /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/

// 規則は admin_entra_handler.go の configureEntraRequest.validate に従う。domain は URL ではない
// DNS 名、source anchor の属性は必須、reply URL はスキームとホストを持つ絶対 URL、Entra のテナント ID は GUID である。
function entraSchema(domainNotUrl: string, tenantIdInvalid: string) {
  return formSchema({
    domain: rule(requiredText(), (value) =>
      value.includes('://') ? { kind: 'message', text: domainNotUrl } : undefined,
    ),
    issuer: optionalText(),
    sourceAnchor: requiredText(),
    replyURL: optionalUrl(),
    entraTenantID: rule(optionalText(), (value) =>
      value !== undefined && !ENTRA_TENANT_ID.test(value)
        ? { kind: 'message', text: tenantIdInvalid }
        : undefined,
    ),
  })
}

type EntraFormInitial = {
  domain: string
  issuer: string
  sourceAnchor: string
  replyURL: string
  entraTenantID: string
}

const ADD_INITIAL: EntraFormInitial = {
  domain: '',
  issuer: '',
  sourceAnchor: 'on_premises_immutable_id',
  replyURL: 'https://login.microsoftonline.com/login.srf',
  entraTenantID: '',
}

// EntraFederationForm は追加と編集の共通のフォームである。保存に成功したらドメインの詳細へ移り、
// 失敗したら入力を残して近くに Alert を出す。
function EntraFederationForm({
  csrfToken,
  initial,
  lockDomain,
  cancelHref,
}: {
  csrfToken: string
  initial: EntraFormInitial
  lockDomain: boolean
  cancelHref: string
}) {
  const [domain, setDomain] = useState(initial.domain)
  const [issuer, setIssuer] = useState(initial.issuer)
  const [sourceAnchor, setSourceAnchor] = useState(initial.sourceAnchor)
  const [editingSourceAnchor, setEditingSourceAnchor] = useState(false)
  const [replyURL, setReplyURL] = useState(initial.replyURL)
  const [entraTenantID, setEntraTenantID] = useState(initial.entraTenantID)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const t = useDictionary(adminEntraFederationDictionary)
  const { locale } = useLocale()
  const schema = useMemo(
    () => entraSchema(t.domainNotUrlError, t.entraTenantIdInvalidError),
    [t.domainNotUrlError, t.entraTenantIdInvalidError],
  )
  const validation = useFormValidation(schema, {
    domain,
    issuer,
    sourceAnchor,
    replyURL,
    entraTenantID,
  })

  async function handleConfigure(event: FormEvent) {
    event.preventDefault()
    setError('')
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    setSaving(true)
    try {
      const requestedDomain = data.domain.toLowerCase()
      await configureEntraFederation(csrfToken, {
        domain: requestedDomain,
        issuer_uri: data.issuer,
        source_anchor_attribute: data.sourceAnchor,
        reply_url: data.replyURL,
        entra_tenant_id: data.entraTenantID,
      })
      returnToEntraDomain(
        requestedDomain,
        t.federationSavedNotice.replace('{domain}', requestedDomain),
      )
    } catch (cause) {
      setSaving(false)
      setError(configureErrorMessage(t, locale, cause, data.sourceAnchor))
    }
  }

  return (
    <>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="grid gap-4 p-4">
        <form className="grid gap-3 sm:grid-cols-2" onSubmit={handleConfigure}>
          <div className="grid gap-1.5">
            <Label htmlFor="entra_domain" required>
              {t.verifiedDomainLabel}
            </Label>
            <Input
              id="entra_domain"
              value={domain}
              placeholder="contoso.com"
              onChange={(e) => setDomain(e.target.value)}
              readOnly={lockDomain}
              required
              {...validation.fieldProps('domain')}
            />
            <FieldError form={validation} name="domain" />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="entra_tenant_id">{t.entraTenantIdLabel}</Label>
            <Input
              id="entra_tenant_id"
              value={entraTenantID}
              placeholder="00000000-0000-0000-0000-000000000000"
              onChange={(e) => setEntraTenantID(e.target.value)}
              {...validation.fieldProps('entraTenantID')}
            />
            <p className="text-xs text-slate-500">{t.entraTenantIdHint}</p>
            <FieldError form={validation} name="entraTenantID" />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="entra_issuer">{t.issuerUriLabel}</Label>
            <Input
              id="entra_issuer"
              value={issuer}
              placeholder={t.issuerUriPlaceholder}
              onChange={(e) => setIssuer(e.target.value)}
            />
          </div>
          <div className="grid gap-1.5 sm:col-span-2">
            <Label htmlFor="entra_reply">{t.replyUrlLabel}</Label>
            <Input
              id="entra_reply"
              type="url"
              value={replyURL}
              onChange={(e) => setReplyURL(e.target.value)}
              {...validation.fieldProps('replyURL')}
            />
            <FieldError form={validation} name="replyURL" />
          </div>
          <div className="grid gap-3 sm:col-span-2">
            <EntraImmutableIdGuide />
            <dl className="text-sm">
              <dt>{t.immutableIdStorageLabel}</dt>
              <dd className="break-all font-mono">{sourceAnchor}</dd>
            </dl>
            <div>
              <Button
                type="button"
                variant="outline"
                aria-expanded={editingSourceAnchor}
                aria-controls="entra_source_anchor_settings"
                onClick={() => setEditingSourceAnchor((open) => !open)}
              >
                {t.sourceAnchorAdvancedSettings}
              </Button>
            </div>
            {editingSourceAnchor ? (
              <div id="entra_source_anchor_settings" className="grid gap-1.5">
                <Label htmlFor="entra_source_anchor" required>
                  {t.sourceAnchorAttributeLabel}
                </Label>
                <p className="text-xs text-slate-500">{t.sourceAnchorAdvancedHint}</p>
                <Input
                  id="entra_source_anchor"
                  value={sourceAnchor}
                  onChange={(e) => setSourceAnchor(e.target.value)}
                  required
                  {...validation.fieldProps('sourceAnchor')}
                />
                <FieldError form={validation} name="sourceAnchor" />
              </div>
            ) : null}
          </div>
          <div className="flex gap-2 sm:col-span-2">
            <Button type="submit" disabled={!validation.data || saving}>
              {t.save}
            </Button>
            <Button variant="outline" nativeButton={false} render={<a href={cancelHref} />}>
              {t.cancel}
            </Button>
          </div>
        </form>
      </Card>
    </>
  )
}

function configureErrorMessage(
  t: AdminEntraFederationDictionary,
  locale: Locale,
  cause: unknown,
  sourceAnchor: string,
): string {
  if (!(cause instanceof AuthenticationAPIError)) return t.federationSaveFailedError
  switch (cause.code) {
    case 'source_anchor_not_unique':
      return t.sourceAnchorNotUniqueError
    case 'source_anchor_editable':
      return t.sourceAnchorEditableError
    case 'source_anchor_missing':
    case 'source_anchor_invalid': {
      const count = cause.body?.user_count
      const users = cause.body?.sample_usernames
      if (
        typeof count !== 'number' ||
        !Array.isArray(users) ||
        !users.every((name) => typeof name === 'string')
      ) {
        return t.federationSaveFailedError
      }
      const message =
        cause.code === 'source_anchor_missing'
          ? t.sourceAnchorMissingError
          : t.sourceAnchorInvalidError
      return `${message.replace('{count}', count.toLocaleString(locale))} ${t.sourceAnchorSampleUsers.replace('{users}', users.join(', '))} ${t.sourceAnchorRepairHint.replace('{attribute}', sourceAnchor)}`
    }
    default:
      return localizedErrorMessage(locale, cause.code, cause.message)
  }
}

// AdminEntraFederationAddPage はドメインの追加の画面 (/admin/federation/entra/new)。保存すると
// 追加したドメインの詳細へ移り、そこで Graph PowerShell の手順と反映の確認を案内する。
export function AdminEntraFederationAddPage({
  csrfToken,
  actorUsername,
  sourceAnchorAttribute,
}: {
  csrfToken: string
  actorUsername?: string
  sourceAnchorAttribute?: string
}) {
  const t = useDictionary(adminEntraFederationDictionary)
  return (
    <AdminShell
      active="entra-federation"
      actorUsername={actorUsername}
      title={t.addPageTitle}
      description={t.addPageDescription}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={tenantURL(ENTRA_LIST_PATH)} />}
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.federationList}
        </Button>
      }
    >
      <EntraFederationForm
        csrfToken={csrfToken}
        initial={{
          ...ADD_INITIAL,
          sourceAnchor: sourceAnchorAttribute ?? ADD_INITIAL.sourceAnchor,
        }}
        lockDomain={false}
        cancelHref={tenantURL(ENTRA_LIST_PATH)}
      />
    </AdminShell>
  )
}

// AdminEntraFederationEditPage は構成済みのドメインの編集の画面。保存した値で始め、ドメインは変えない。
export function AdminEntraFederationEditPage({
  csrfToken,
  actorUsername,
  domain,
  sourceAnchorAttribute,
  replyURL,
}: {
  csrfToken: string
  actorUsername?: string
  domain: EntraDomainFederation
  sourceAnchorAttribute: string
  replyURL: string
}) {
  const t = useDictionary(adminEntraFederationDictionary)
  return (
    <AdminShell
      active="entra-federation"
      actorUsername={actorUsername}
      title={t.editPageTitle.replace('{domain}', domain.domain)}
      description={t.editPageDescription}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={entraDomainURL(domain.domain)} />}
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.viewDomainDetail.replace('{domain}', domain.domain)}
        </Button>
      }
    >
      <EntraFederationForm
        csrfToken={csrfToken}
        initial={{
          domain: domain.domain,
          issuer: domain.issuer_uri,
          sourceAnchor: sourceAnchorAttribute,
          replyURL,
          entraTenantID: domain.entra_tenant_id ?? '',
        }}
        lockDomain
        cancelHref={entraDomainURL(domain.domain)}
      />
    </AdminShell>
  )
}

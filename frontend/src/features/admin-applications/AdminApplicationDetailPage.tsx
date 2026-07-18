import {
  IconArrowLeft,
  IconExternalLink,
  IconKey,
  IconPencil,
  IconTrash,
  IconWorldShare,
} from '@tabler/icons-react'
import { useState } from 'react'
import { deleteAdminApplication } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useLocale } from '../../lib/i18n'
import { AssignmentList } from './AdminApplicationAssignments'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import {
  AppIcon,
  CopyableField,
  editURL,
  kindLabel,
  KindBadge,
  listURL,
  messageOf,
  nameIdFormatOptions,
  ReadOnlyField,
  ReadonlyMeta,
  SectionTitle,
  StatusBadge,
  summarizeSignInRule,
  UriList,
  wsfedTokenTypeOptions,
} from './AdminApplicationsShared'
import type { AdminApplicationDetail } from '../../types'

export function AdminApplicationDetailPage({
  csrfToken,
  actorUsername,
  detail,
}: {
  csrfToken: string
  actorUsername?: string
  detail: AdminApplicationDetail
}) {
  const app = detail.application
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminApplicationsDictionary)
  const { locale } = useLocale()

  async function handleDelete() {
    setBusy(true)
    setError('')
    try {
      await deleteAdminApplication(csrfToken, app.application_id)
      window.location.assign(listURL())
    } catch (cause) {
      setError(messageOf(cause, t.applicationDeleteFailedError))
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="applications"
      actorUsername={actorUsername}
      title={app.name}
      description={kindLabel(app, t)}
      actions={
        <div className="flex items-center gap-2">
          <a
            href={listURL()}
            className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
          >
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.backToList}
          </a>
          <Button asChild>
            <a href={editURL(app.application_id)}>
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </a>
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={busy}
            onClick={() => setConfirmDelete(true)}
          >
            <IconTrash size={16} aria-hidden="true" />
            {t.delete}
          </Button>
        </div>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {confirmDelete ? (
        <Alert variant="destructive" className="flex flex-wrap items-center justify-between gap-2">
          <span>{t.confirmDeleteAppPrompt}</span>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setConfirmDelete(false)} disabled={busy}>
              {t.dismissConfirm}
            </Button>
            <Button variant="destructive" disabled={busy} onClick={() => void handleDelete()}>
              <IconTrash size={14} aria-hidden="true" />
              {t.confirmDelete}
            </Button>
          </div>
        </Alert>
      ) : null}

      <div className="grid max-w-3xl gap-6">
        <Card className="overflow-hidden">
          <div className="flex items-start gap-3 border-b border-slate-200 p-5">
            <AppIcon app={app} />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h2 className="truncate text-lg font-semibold text-slate-950">{app.name}</h2>
                <StatusBadge status={app.status} />
              </div>
              <div className="mt-1">
                <KindBadge app={app} />
              </div>
            </div>
          </div>

          <div className="grid gap-6 p-5">
            {/* 基本情報セクション */}
            <section className="grid gap-4 sm:grid-cols-2">
              <ReadOnlyField label={t.kindFieldLabel}>
                <span>{kindLabel(app, t)}</span>
              </ReadOnlyField>
              <ReadOnlyField label={t.statusFieldLabel}>
                <StatusBadge status={app.status} />
              </ReadOnlyField>
              <ReadOnlyField label={t.categoryFieldLabel}>
                {app.category_names && app.category_names.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {app.category_names.map((name) => (
                      <span
                        key={name}
                        className="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700"
                      >
                        {name}
                      </span>
                    ))}
                  </div>
                ) : (
                  <span className="text-slate-400">{t.noCategoryNotice}</span>
                )}
              </ReadOnlyField>
              {app.kind !== 'service' ? (
                <ReadOnlyField label={t.launchUrlFieldLabel}>
                  {app.launch_url ? (
                    <a
                      href={app.launch_url}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 break-all font-mono text-xs text-blue-700 hover:underline"
                    >
                      {app.launch_url}
                      <IconExternalLink size={13} aria-hidden="true" />
                    </a>
                  ) : (
                    <span className="text-slate-400">{t.notSetLabel}</span>
                  )}
                </ReadOnlyField>
              ) : null}
            </section>

            <section className="grid gap-4 border-t border-slate-100 pt-5 sm:grid-cols-2 text-xs text-slate-500">
              <ReadOnlyField label={t.registeredAtFieldLabel}>
                <span>
                  {app.created_at
                    ? new Date(app.created_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
                    : t.unknownDate}
                </span>
              </ReadOnlyField>
              <ReadOnlyField label={t.lastUpdatedAtFieldLabel}>
                <span>
                  {app.updated_at
                    ? new Date(app.updated_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
                    : t.unknownDate}
                </span>
              </ReadOnlyField>
            </section>

            {detail.oidc ? (
              <section className="grid gap-3 border-t border-slate-100 pt-5 first:border-t-0 first:pt-0">
                <div className="flex items-center gap-2">
                  <IconKey size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>
                    {app.kind === 'service' ? t.serviceKindSectionHeading : t.oidcSectionHeading}
                  </SectionTitle>
                </div>
                <CopyableField label={t.clientIdFieldLabel} value={detail.oidc.client_id} />
                {app.kind !== 'service' ? (
                  <ReadOnlyField label={t.redirectUriFieldLabel}>
                    <UriList values={detail.oidc.redirect_uris} />
                  </ReadOnlyField>
                ) : null}
                <ReadOnlyField label={t.scopeFieldLabel}>
                  <span className="font-mono text-xs">{detail.oidc.scope || '—'}</span>
                </ReadOnlyField>
                <ReadOnlyField label={t.grantTypesFieldLabel}>
                  <span className="font-mono text-xs">
                    {detail.oidc.grant_types.join(', ') || '—'}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.responseTypesFieldLabel}>
                  <span className="font-mono text-xs">
                    {detail.oidc.response_types.join(', ') || '—'}
                  </span>
                </ReadOnlyField>
                <div className="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs sm:grid-cols-3">
                  <ReadonlyMeta label={t.clientTypeMetaLabel} value={detail.oidc.client_type} />
                  <ReadonlyMeta
                    label={t.authMethodMetaLabel}
                    value={detail.oidc.token_endpoint_auth_method}
                  />
                  <ReadonlyMeta label={t.fapiProfileMetaLabel} value={detail.oidc.fapi_profile} />
                </div>
                <ReadOnlyField label={t.securityFieldLabel}>
                  <span className="text-xs text-slate-700">
                    {[
                      detail.oidc.require_pushed_authorization_requests ? t.parRequired : '',
                      detail.oidc.dpop_bound_access_tokens ? t.dpopBound : '',
                    ]
                      .filter(Boolean)
                      .join(', ') || t.standardSecurity}
                  </span>
                </ReadOnlyField>
                {app.kind === 'service' ? (
                  <p className="text-xs text-slate-500">{t.m2mNoLoginNotice}</p>
                ) : null}
              </section>
            ) : null}

            {detail.wsfed ? (
              <section className="grid gap-3 border-t border-slate-100 pt-5">
                <div className="flex items-center gap-2">
                  <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.wsFedSectionHeading}</SectionTitle>
                </div>
                <CopyableField label={t.wtrealmFieldLabel} value={detail.wsfed.wtrealm} />
                <ReadOnlyField label={t.replyUrlFieldLabel}>
                  <UriList values={detail.wsfed.reply_urls} />
                </ReadOnlyField>
                <ReadOnlyField label={t.nameIdFormatFieldLabel}>
                  <span className="break-all font-mono text-xs">
                    {nameIdFormatOptions(t).find((f) => f.value === detail.wsfed?.name_id_format)
                      ?.label ?? detail.wsfed.name_id_format}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.nameIdSourceFieldLabel}>
                  <span className="font-mono text-xs">{detail.wsfed.name_id_source}</span>
                </ReadOnlyField>
                <ReadOnlyField label={t.audienceFieldLabel}>
                  <span className="font-mono text-xs">
                    {detail.wsfed.audience ||
                      t.audienceDefaultSuffix.replace('{value}', detail.wsfed.wtrealm)}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.tokenTypeFieldLabel}>
                  <span className="text-xs">
                    {wsfedTokenTypeOptions(t).find((opt) => opt.value === detail.wsfed?.token_type)
                      ?.label ?? detail.wsfed.token_type}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.claimMappingRulesFieldLabel}>
                  {detail.wsfed.rules.length === 0 ? (
                    <span className="text-xs text-slate-400">{t.nameIdOnlyNotice}</span>
                  ) : (
                    <ul className="flex flex-wrap gap-1.5">
                      {detail.wsfed.rules.map((rule) => (
                        <li
                          key={rule.claim_type}
                          className="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs text-slate-700"
                        >
                          {rule.claim_type.split('/').pop()}
                          {rule.required ? '*' : ''}
                        </li>
                      ))}
                    </ul>
                  )}
                </ReadOnlyField>
              </section>
            ) : null}

            {detail.saml ? (
              <section className="grid gap-3 border-t border-slate-100 pt-5">
                <div className="flex items-center gap-2">
                  <IconWorldShare size={16} className="text-slate-400" aria-hidden="true" />
                  <SectionTitle>{t.samlSectionHeading}</SectionTitle>
                </div>
                <CopyableField label={t.entityIdFieldLabel} value={detail.saml.entity_id} />
                <ReadOnlyField label={t.acsUrlFieldLabel}>
                  <UriList values={detail.saml.acs_urls} />
                </ReadOnlyField>
                <ReadOnlyField label={t.sloUrlFieldLabel}>
                  <span className="break-all font-mono text-xs">{detail.saml.slo_url || '—'}</span>
                </ReadOnlyField>
                <ReadOnlyField label={t.nameIdFormatFieldLabel}>
                  <span className="break-all font-mono text-xs">
                    {nameIdFormatOptions(t).find((f) => f.value === detail.saml?.name_id_format)
                      ?.label ?? detail.saml.name_id_format}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.nameIdSourceFieldLabel}>
                  <span className="font-mono text-xs">{detail.saml.name_id_source}</span>
                </ReadOnlyField>
                <ReadOnlyField label={t.audienceFieldLabel}>
                  <span className="font-mono text-xs">
                    {detail.saml.audience ||
                      t.audienceDefaultSuffix.replace('{value}', detail.saml.entity_id)}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.signatureFieldLabel}>
                  <span className="text-xs">
                    {[
                      detail.saml.sign_assertion ? t.assertionSigned : '',
                      detail.saml.sign_response ? t.responseSigned : '',
                    ]
                      .filter(Boolean)
                      .join(' / ') || t.noSignature}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.requestSignatureVerificationFieldLabel}>
                  <span className="text-xs">
                    {detail.saml.want_authn_requests_signed
                      ? t.authnRequestSignatureRequired
                      : t.optional}
                  </span>
                </ReadOnlyField>
                <ReadOnlyField label={t.claimMappingRulesFieldLabel}>
                  {detail.saml.rules.length === 0 ? (
                    <span className="text-xs text-slate-400">{t.nameIdOnlyNotice}</span>
                  ) : (
                    <ul className="flex flex-wrap gap-1.5">
                      {detail.saml.rules.map((rule) => (
                        <li
                          key={rule.claim_type}
                          className="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs text-slate-700"
                        >
                          {rule.claim_type.split('/').pop()}
                          {rule.required ? '*' : ''}
                        </li>
                      ))}
                    </ul>
                  )}
                </ReadOnlyField>
              </section>
            ) : null}

            {/* ログインポリシーセクション */}
            <section className="grid gap-3 border-t border-slate-100 pt-5">
              <div className="flex items-center gap-2">
                <IconKey size={16} className="text-slate-400" aria-hidden="true" />
                <SectionTitle>{t.signInPolicySectionHeading}</SectionTitle>
              </div>
              <ReadOnlyField label={t.applicationStatusFieldLabel}>
                <span className="text-slate-700 font-semibold">{app.sign_in_policy_summary}</span>
              </ReadOnlyField>
              {detail.sign_in_policy && detail.sign_in_policy.effective_rules.length > 0 ? (
                <ReadOnlyField label={t.appliedRulesFieldLabel}>
                  <ul className="mt-1 flex flex-col gap-1.5">
                    {detail.sign_in_policy.effective_rules.map((rule) => (
                      <li
                        key={rule.rule_id}
                        className="rounded-lg border border-slate-200 bg-slate-50 p-3 font-mono text-xs text-slate-700"
                      >
                        <div className="font-sans font-semibold mb-1">
                          {rule.name || t.ruleNameFallback.replace('{id}', rule.rule_id)}
                        </div>
                        {summarizeSignInRule(rule, t)}
                      </li>
                    ))}
                  </ul>
                </ReadOnlyField>
              ) : (
                <ReadOnlyField label={t.appliedRulesFieldLabel}>
                  <span className="text-xs text-slate-400">{t.noAppliedRulesNotice}</span>
                </ReadOnlyField>
              )}
            </section>

            {app.kind !== 'service' ? (
              <section className="grid gap-3 border-t border-slate-100 pt-5">
                <SectionTitle>{t.assignmentsHeading}</SectionTitle>
                <AssignmentList appID={app.application_id} onError={setError} />
              </section>
            ) : null}
          </div>
        </Card>
      </div>
    </AdminShell>
  )
}

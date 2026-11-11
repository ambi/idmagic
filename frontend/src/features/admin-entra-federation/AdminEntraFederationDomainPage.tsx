import { IconArrowLeft, IconCopy, IconPencil, IconUnlink } from '@tabler/icons-react'
import { type ChangeEvent, type FormEvent, type ReactNode, useState } from 'react'
import {
  AuthenticationAPIError,
  type EntraFederationDomainDetail,
  type EntraPowerShellProcedure,
  recordEntraFederationObservation,
  removeEntraFederationDomain,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { formSchema, requiredJson, useForm } from '../../lib/form'
import { useDictionary, useLocale } from '../../lib/i18n'
import { localizedErrorMessage } from '../../lib/i18n/errorMessage'
import type { Locale } from '../../lib/i18n/locale'
import type { EntraFederationVerification } from '../../types'
import {
  adminEntraFederationDictionary,
  type AdminEntraFederationDictionary,
} from './AdminEntraFederationPage.i18n'
import { EntraVerificationStatusBadge } from './EntraVerificationStatusBadge'
import {
  ENTRA_LIST_PATH,
  entraDomainEditURL,
  returnToEntraList,
  takeEntraDomainNotice,
} from './entraNavigation'

const observationSchema = formSchema({ observation: requiredJson() })

const PROCEDURE_STEPS: {
  key: keyof EntraPowerShellProcedure
  label: keyof AdminEntraFederationDictionary
}[] = [
  { key: 'connect', label: 'procedureConnect' },
  { key: 'backup', label: 'procedureBackup' },
  { key: 'compare', label: 'procedureCompare' },
  { key: 'apply', label: 'procedureApply' },
  { key: 'read_back', label: 'procedureReadBack' },
  { key: 'connect_read_only', label: 'procedureConnectReadOnly' },
]

// AdminEntraFederationDomainPage は一つのドメインの読み取り専用の詳細である。保存した設定、Entra に
// 登録する値、手順、反映の確認の状態を変更の操作なしに再取得して示し、編集は別の画面へ移る。
// 観測の取り込みは IdMagic の照合の記録だけを書く独立した操作なので、この画面に置く。
export function AdminEntraFederationDomainPage({
  csrfToken,
  actorUsername,
  detail,
}: {
  csrfToken: string
  actorUsername?: string
  detail: EntraFederationDomainDetail
}) {
  const t = useDictionary(adminEntraFederationDictionary)
  const { locale } = useLocale()
  const domain = detail.domain.domain
  const [verification, setVerification] = useState(detail.verification)
  const [notice, setNotice] = useState(() => takeEntraDomainNotice(domain))
  const [error, setError] = useState('')
  const [importError, setImportError] = useState('')
  const [importing, setImporting] = useState(false)
  const form = useForm(observationSchema, { observation: '' })

  async function copy(value: string) {
    setError('')
    try {
      await navigator.clipboard.writeText(value)
      setNotice(t.commandCopiedNotice)
    } catch {
      setError(t.commandCopyFailedError)
    }
  }

  async function loadFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return
    // Windows PowerShell 5.1 の出力は BOM を付けることがあり、JSON.parse は BOM を読まない。
    form.setValue('observation', (await file.text()).replace(/^﻿/, ''))
  }

  async function handleImport(event: FormEvent) {
    event.preventDefault()
    setImportError('')
    const data = form.data
    if (!data) {
      form.revealAll()
      return
    }
    setImporting(true)
    try {
      const recorded = await recordEntraFederationObservation(csrfToken, domain, data.observation)
      setVerification(recorded.verification)
      form.setValue('observation', '')
      setNotice(t.observationRecordedNotice)
    } catch (cause) {
      // 拒否された文書は入力に残し、管理者が直して再送できるようにする。
      setImportError(observationErrorMessage(t, locale, cause))
    } finally {
      setImporting(false)
    }
  }

  return (
    <AdminShell
      active="entra-federation"
      actorUsername={actorUsername}
      title={domain}
      description={t.domainDetailDescription}
      actions={
        <div className="flex gap-2">
          <Button
            variant="outline"
            nativeButton={false}
            render={<a href={tenantURL(ENTRA_LIST_PATH)} />}
          >
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.federationList}
          </Button>
          <Button nativeButton={false} render={<a href={entraDomainEditURL(domain)} />}>
            <IconPencil size={16} aria-hidden="true" />
            {t.edit}
          </Button>
        </div>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <Section title={t.verificationHeading} testId="entra-verification">
        <VerificationSummary verification={verification} />
        <form className="grid gap-2" onSubmit={handleImport}>
          <Label htmlFor="entra_observation" required>
            {t.observationLabel}
          </Label>
          <textarea
            id="entra_observation"
            className="min-h-32 w-full rounded-md border border-slate-300 bg-white p-2 font-mono text-xs"
            required
            {...form.input('observation')}
          />
          <FieldError form={form} name="observation" />
          <div className="flex flex-wrap items-center gap-3">
            <label className="text-xs text-slate-600">
              {t.observationFileLabel}{' '}
              <input type="file" accept="application/json,.json" onChange={loadFile} />
            </label>
            <Button type="submit" disabled={!form.data || importing}>
              {importing ? t.importingObservation : t.importObservation}
            </Button>
          </div>
          {importError ? <Alert variant="destructive">{importError}</Alert> : null}
        </form>
      </Section>

      <Section title={t.savedSettingsHeading}>
        <dl className="grid gap-2 text-xs sm:grid-cols-2">
          <Item label={t.verifiedDomainLabel} value={domain} />
          <Item label={t.issuerUriLabel} value={detail.domain.issuer_uri} />
          <Item
            label={t.entraTenantIdLabel}
            value={detail.domain.entra_tenant_id ?? t.notSpecified}
          />
          <Item
            label={t.sourceAnchorAttributeLabel}
            value={detail.profile.source_anchor_attribute}
          />
          <Item label={t.replyUrlLabel} value={detail.reply_url} />
        </dl>
      </Section>

      <Section title={t.expectedSettingsHeading}>
        <dl className="grid gap-2 text-xs sm:grid-cols-2">
          {Object.entries(detail.powershell)
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([key, value]) => (
              <Item key={key} label={key} value={value} />
            ))}
        </dl>
      </Section>

      <Section title={t.certificatesHeading}>
        <dl className="grid gap-2 text-xs sm:grid-cols-2">
          <Item
            label={`${t.currentCertificateLabel} ${t.thumbprintLabel}`}
            value={detail.certificates.signing.sha256_thumbprint}
          />
          <Item
            label={`${t.currentCertificateLabel} ${t.notAfterLabel}`}
            value={new Date(detail.certificates.signing.not_after).toLocaleString(locale)}
          />
          {detail.certificates.next ? (
            <>
              <Item
                label={`${t.nextCertificateLabel} ${t.thumbprintLabel}`}
                value={detail.certificates.next.sha256_thumbprint}
              />
              <Item
                label={`${t.nextCertificateLabel} ${t.notAfterLabel}`}
                value={new Date(detail.certificates.next.not_after).toLocaleString(locale)}
              />
            </>
          ) : null}
        </dl>
      </Section>

      <Section title={t.prerequisitesHeading}>
        <ul className="list-disc pl-5 text-xs leading-5 text-slate-700">
          <li>{t.prerequisiteAdDelegation}</li>
          <li>{t.prerequisiteUpnImmutableId}</li>
          <li>{t.prerequisiteConnectComputers}</li>
          <li>{t.prerequisiteScp}</li>
          <li>{t.prerequisiteSystemNetwork}</li>
        </ul>
      </Section>

      <Section title={t.procedureHeading} testId="entra-procedure">
        <p className="text-xs text-slate-600">{t.procedureNotice}</p>
        {PROCEDURE_STEPS.map(({ key, label }) => (
          <Script
            key={key}
            title={t[label]}
            value={detail.procedure[key]}
            copyLabel={`${t.copy}: ${t[label]}`}
            onCopy={copy}
          />
        ))}
        <h3 className="mt-2 text-sm font-semibold">{t.commandsHeading}</h3>
        <Script
          title={t.createCommandLabel}
          value={detail.commands.create}
          copyLabel={t.copyCreateCommand}
          onCopy={copy}
        />
        <Script
          title={t.updateCommandLabel}
          value={detail.commands.update}
          copyLabel={t.copyUpdateCommand}
          onCopy={copy}
        />
      </Section>

      <RemovalSection csrfToken={csrfToken} detail={detail} onCopy={copy} />

      <Section title={t.postChecksHeading}>
        <p className="text-xs text-slate-600">{t.postChecksNotice}</p>
        <ul className="list-disc pl-5 text-xs leading-5 text-slate-700">
          <li>{t.postCheckSyncJoin}</li>
          <li>{t.postCheckDeviceJoin}</li>
          <li>{t.postCheckPrt}</li>
          <li>{t.postCheckAppSso}</li>
        </ul>
        {detail.known_limitations.map((limitation) => (
          <Alert variant="destructive" key={limitation.code}>
            {t.hybridJoinNotProvidedNotice}
          </Alert>
        ))}
        {detail.unfederated_source_users ? (
          <Alert variant="destructive">{`${t.unfederatedSourceUsersNotice.replace('{count}', detail.unfederated_source_users.user_count.toLocaleString(locale))} ${t.sourceAnchorSampleUsers.replace('{users}', detail.unfederated_source_users.sample_usernames.join(', '))}`}</Alert>
        ) : null}
      </Section>
    </AdminShell>
  )
}

// MIGRATION_GUIDE_URL は、Microsoft の federation から cloud authentication への移行の手順である。
// Managed への切り替えの権限と cmdlet は、この手順の「Convert domains from federated to managed」に合わせる。
const MIGRATION_GUIDE_URL =
  'https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/migrate-from-federation-to-cloud-authentication'

// managedCommand は、Managed へ切り替える固定のコマンドである。保存値を PowerShell の単一引用符の
// 文字列として符号化し、構成時の検証に依存せず一つの引数として渡す。
function managedCommand(domain: string): string {
  const quotedDomain = domain.replaceAll("'", "''")
  return [
    `Connect-MgGraph -Scopes 'Domain.ReadWrite.All','Directory.AccessAsUser.All' -NoWelcome`,
    `Update-MgDomain -DomainId '${quotedDomain}' -AuthenticationType 'Managed'`,
  ].join('\n')
}

// RemovalSection は、Entra 側の切り戻しとローカルの解除を案内する。解除の可否はサーバーが Microsoft の
// レルム探索で確かめるので、画面は解除を要求し、拒否されたら理由を示して詳細に留まる。
function RemovalSection({
  csrfToken,
  detail,
  onCopy,
}: {
  csrfToken: string
  detail: EntraFederationDomainDetail
  onCopy: (value: string) => Promise<void>
}) {
  const t = useDictionary(adminEntraFederationDictionary)
  const { locale } = useLocale()
  const domain = detail.domain.domain
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const remaining = detail.profile.domains
    .map((federated) => federated.domain)
    .filter((name) => name !== domain)
    .sort()
  const impact =
    remaining.length > 0
      ? t.removalRemainingDomains.replace('{domains}', remaining.join(', '))
      : t.removalLastDomain

  async function handleRemove() {
    setBusy(true)
    setError('')
    try {
      await removeEntraFederationDomain(csrfToken, domain)
      returnToEntraList(t.removalRemovedNotice.replace('{domain}', domain))
    } catch (cause) {
      // 拒否されたら詳細に留まる。ドメインは構成されたままなので、切り戻しの後に再実行できる。
      setBusy(false)
      setConfirming(false)
      setError(removalErrorMessage(t, locale, cause))
    }
  }

  return (
    <Section title={t.removalHeading} testId="entra-removal">
      <p className="text-xs text-slate-600">{t.removalNotice}</p>
      <p className="text-xs font-semibold text-slate-800">{impact}</p>
      <h3 className="text-xs font-semibold text-slate-800">{t.removalTargetManaged}</h3>
      <p className="text-xs text-slate-600">{t.removalManagedNotice}</p>
      <Script
        title={t.procedureReleaseToManaged}
        value={managedCommand(domain)}
        copyLabel={`${t.copy}: ${t.procedureReleaseToManaged}`}
        onCopy={onCopy}
      />
      <h3 className="text-xs font-semibold text-slate-800">{t.removalTargetPreviousIdp}</h3>
      <p className="text-xs text-slate-600">{t.removalPreviousIdpNotice}</p>
      <a
        className="text-xs font-semibold text-blue-700 hover:underline"
        href={MIGRATION_GUIDE_URL}
        target="_blank"
        rel="noreferrer"
      >
        {t.removalMigrationGuide}
      </a>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div>
        <Button
          type="button"
          variant="destructive"
          disabled={busy}
          onClick={() => {
            setError('')
            setConfirming(true)
          }}
        >
          <IconUnlink size={16} aria-hidden="true" />
          {t.removeDomain}
        </Button>
      </div>
      {confirming ? (
        <ConfirmActionDialog
          title={t.removalConfirmTitle.replace('{domain}', domain)}
          target={impact}
          description={t.removalConfirmDescription}
          confirmLabel={t.removeDomain}
          busy={busy}
          onClose={() => setConfirming(false)}
          onConfirm={() => void handleRemove()}
        />
      ) : null}
    </Section>
  )
}

function removalErrorMessage(
  t: AdminEntraFederationDictionary,
  locale: Locale,
  cause: unknown,
): string {
  if (!(cause instanceof AuthenticationAPIError)) return t.removalFailedError
  switch (cause.code) {
    case 'entra_federation_still_active':
      return t.removalStillActiveError
    case 'entra_rollback_unconfirmed':
      return t.removalUnconfirmedError
    default:
      return localizedErrorMessage(locale, cause.code, cause.message)
  }
}

function observationErrorMessage(
  t: AdminEntraFederationDictionary,
  locale: Locale,
  cause: unknown,
): string {
  if (!(cause instanceof AuthenticationAPIError)) return t.federationSaveFailedError
  if (cause.code === 'observation_outdated') return t.observationOutdatedError
  if (cause.code === 'invalid_request') {
    return t.observationRejectedError.replace('{detail}', cause.message)
  }
  return localizedErrorMessage(locale, cause.code, cause.message)
}

function statusDescription(
  t: AdminEntraFederationDictionary,
  verification: EntraFederationVerification,
) {
  return {
    unverified: t.statusUnverifiedDescription,
    matched: t.statusMatchedDescription,
    mismatched: t.statusMismatchedDescription,
    stale: t.statusStaleDescription,
  }[verification.status]
}

function VerificationSummary({ verification }: { verification: EntraFederationVerification }) {
  const t = useDictionary(adminEntraFederationDictionary)
  const { locale } = useLocale()
  const time = (value?: string) => (value ? new Date(value).toLocaleString(locale) : '')
  return (
    <div className="grid gap-2 text-xs text-slate-700">
      <div className="flex items-center gap-2">
        <EntraVerificationStatusBadge status={verification.status} />
        <span>{statusDescription(t, verification)}</span>
      </div>
      <p className="text-slate-500">{t.verificationManualNotice}</p>
      {verification.checked_at ? (
        <dl className="grid gap-2 sm:grid-cols-3">
          <Item label={t.retrievedAtLabel} value={time(verification.retrieved_at)} />
          <Item label={t.checkedAtLabel} value={time(verification.checked_at)} />
          <Item label={t.observedTenantLabel} value={verification.observed_entra_tenant_id ?? ''} />
        </dl>
      ) : null}
      {verification.differences && verification.differences.length > 0 ? (
        <table className="w-full table-fixed border-collapse text-left">
          <thead>
            <tr className="border-b border-slate-200">
              <th className="p-1.5">{t.differenceFieldLabel}</th>
              <th className="p-1.5">{t.differenceExpectedLabel}</th>
              <th className="p-1.5">{t.differenceObservedLabel}</th>
            </tr>
          </thead>
          <tbody>
            {verification.differences.map((difference) => (
              <tr key={difference.field} className="border-b border-slate-100 align-top">
                <td className="p-1.5 font-mono">{difference.field}</td>
                <td className="break-all p-1.5 font-mono">{difference.expected}</td>
                <td className="break-all p-1.5 font-mono">{difference.observed}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </div>
  )
}

function Section({
  title,
  testId,
  children,
}: {
  title: string
  testId?: string
  children: ReactNode
}) {
  return (
    <Card className="grid gap-3 p-4" data-testid={testId}>
      <h2 className="text-sm font-semibold text-slate-900">{title}</h2>
      {children}
    </Card>
  )
}

function Item({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="font-semibold text-slate-600">{label}</dt>
      <dd className="break-all font-mono text-slate-800">{value}</dd>
    </div>
  )
}

function Script({
  title,
  value,
  copyLabel,
  onCopy,
}: {
  title: string
  value: string
  copyLabel: string
  onCopy: (value: string) => Promise<void>
}) {
  return (
    <div className="grid min-w-0 gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-xs font-semibold text-slate-800">{title}</h3>
        <Button
          type="button"
          variant="outline"
          aria-label={copyLabel}
          onClick={() => void onCopy(value)}
        >
          <IconCopy size={14} aria-hidden="true" />
        </Button>
      </div>
      <pre className="max-h-64 overflow-auto rounded-md bg-slate-50 p-3 text-xs">
        <code>{value}</code>
      </pre>
    </div>
  )
}

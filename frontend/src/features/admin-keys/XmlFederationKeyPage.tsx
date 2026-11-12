import { IconArrowLeft, IconCopy, IconKey, IconRotateClockwise, IconX } from '@tabler/icons-react'
import { type ReactNode, useState } from 'react'
import {
  activateXmlFederationKey,
  AuthenticationAPIError,
  cancelPreparedXmlFederationKey,
  getEntraFederationDomain,
  listEntraFederationDomains,
  listSamlServiceProviders,
  listWsFedRelyingParties,
  listXmlFederationKeys,
  prepareXmlFederationKey,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminKey } from '../../types'
import { EntraVerificationStatusBadge } from '../admin-entra-federation/EntraVerificationStatusBadge'
import { ENTRA_LIST_PATH, entraDomainURL } from '../admin-entra-federation/entraNavigation'
import {
  type ActivationBlocker,
  activationBlockers,
  activeKey,
  confirmationKey,
  entraConfirmsPrepared,
  preparedKey,
  type RolloverSnapshot,
  verifyingKeys,
} from './xmlFederationRollover'
import {
  type XmlFederationKeyDictionary,
  xmlFederationKeyDictionary,
} from './XmlFederationKeyPage.i18n'

// loadRolloverSnapshot は、共有 XML 鍵と、それを信頼する連携先を既存の管理 API から読む。Entra の
// 更新のコマンドはドメインごとの詳細にしかないので、ドメインの数だけ詳細を読む。
export async function loadRolloverSnapshot(): Promise<RolloverSnapshot> {
  const [keys, domains, relyingParties, serviceProviders] = await Promise.all([
    listXmlFederationKeys(),
    listEntraFederationDomains(),
    listWsFedRelyingParties(),
    listSamlServiceProviders(),
  ])
  const details = await Promise.all(
    domains.map((domain) => getEntraFederationDomain(domain.domain)),
  )
  return {
    keys,
    entraDomains: details
      .map((detail) => ({
        domain: detail.domain.domain,
        verification: detail.verification,
        updateCommand: detail.commands.update,
      }))
      .sort((a, b) => a.domain.localeCompare(b.domain)),
    trusts: [
      ...relyingParties
        .filter((rp) => !rp.entra_profile)
        .map((rp) => ({
          id: `wsfed:${rp.wtrealm}`,
          kind: 'wsfed' as const,
          label: rp.display_name || rp.wtrealm,
        })),
      ...serviceProviders
        .filter((sp) => (sp.idp_profile_id || 'default') === 'default')
        .map((sp) => ({
          id: `saml:${sp.entity_id}`,
          kind: 'saml' as const,
          label: sp.display_name || sp.entity_id,
        })),
    ],
  }
}

type Confirmations = { key: string; trusts: Set<string>; overall: boolean }

function emptyConfirmations(snapshot: RolloverSnapshot): Confirmations {
  return { key: confirmationKey(snapshot), trusts: new Set(), overall: false }
}

// XmlFederationKeyPage は、共有 XML 鍵の二段階の更新を案内する。状態は毎回サーバーから読み、管理者の
// 確認だけを画面に持つ。確認は読み込んだ対象に結び付け、対象が変われば外す。
export function XmlFederationKeyPage({
  csrfToken,
  actorUsername,
  snapshot: initial,
}: {
  csrfToken: string
  actorUsername?: string
  snapshot: RolloverSnapshot
}) {
  const t = useDictionary(xmlFederationKeyDictionary)
  const [snapshot, setSnapshot] = useState(initial)
  const [confirmations, setConfirmations] = useState(() => emptyConfirmations(initial))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [dialog, setDialog] = useState<'activate' | 'cancel' | null>(null)

  const prepared = preparedKey(snapshot.keys)
  const blockers = activationBlockers(snapshot, confirmations)

  function replaceSnapshot(next: RolloverSnapshot) {
    setSnapshot(next)
    if (confirmationKey(next) !== confirmations.key) setConfirmations(emptyConfirmations(next))
  }

  async function run(action: () => Promise<string>) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      setNotice(await action())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.actionFailed)
    } finally {
      setBusy(false)
      setDialog(null)
    }
  }

  async function handlePrepare() {
    await run(async () => {
      await prepareXmlFederationKey(csrfToken)
      replaceSnapshot(await loadRolloverSnapshot())
      return t.preparedNotice
    })
  }

  async function handleCancel() {
    if (!prepared) return
    await run(async () => {
      await cancelPreparedXmlFederationKey(csrfToken, prepared.kid)
      replaceSnapshot(await loadRolloverSnapshot())
      return t.cancelledNotice.replace('{kid}', prepared.kid)
    })
  }

  // handleActivate は、確認した時点の対象が今も同じであることを読み直して確かめてから要求する。
  async function handleActivate() {
    if (!prepared) return
    setBusy(true)
    setError('')
    setNotice('')
    try {
      const latest = await loadRolloverSnapshot()
      if (confirmationKey(latest) !== confirmations.key) {
        setSnapshot(latest)
        setConfirmations(emptyConfirmations(latest))
        setError(t.changedSinceConfirmation)
        return
      }
      const result = await activateXmlFederationKey(csrfToken, prepared.kid)
      const next = await loadRolloverSnapshot()
      setSnapshot(next)
      setConfirmations(emptyConfirmations(next))
      setNotice(t.activatedNotice.replace('{kid}', result.next.kid))
    } catch (cause) {
      if (cause instanceof AuthenticationAPIError && cause.code === 'invalid_request') {
        const latest = await loadRolloverSnapshot().catch(() => snapshot)
        setSnapshot(latest)
        setConfirmations(emptyConfirmations(latest))
        setError(t.activationRejected)
      } else {
        setError(cause instanceof AuthenticationAPIError ? cause.message : t.actionFailed)
      }
    } finally {
      setBusy(false)
      setDialog(null)
    }
  }

  async function copy(value: string) {
    try {
      await navigator.clipboard.writeText(value)
    } catch {
      setError(t.actionFailed)
    }
  }

  function toggleTrust(id: string, checked: boolean) {
    setConfirmations((current) => {
      const trusts = new Set(current.trusts)
      if (checked) trusts.add(id)
      else trusts.delete(id)
      return { ...current, trusts }
    })
  }

  return (
    <AdminShell
      active="keys"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            nativeButton={false}
            render={<a href={tenantURL('/admin/keys')} />}
          >
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.signingKeysLink}
          </Button>
          <Button
            variant="outline"
            nativeButton={false}
            render={<a href={tenantURL(ENTRA_LIST_PATH)} />}
          >
            {t.entraLink}
          </Button>
        </div>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <KeyTable keys={snapshot.keys} />
      <ImpactSection snapshot={snapshot} />

      <Section title={t.prepareHeading} testId="xml-key-prepare">
        <p className="text-xs text-slate-600">{t.prepareNotice}</p>
        {prepared ? (
          <div className="grid gap-2">
            <dl className="grid gap-2 text-xs sm:grid-cols-2">
              <Item label={t.preparedKeyLabel} value={prepared.kid} />
              <Item
                label={t.fingerprintLabel}
                value={prepared.certificate_fingerprint_sha256 ?? ''}
              />
            </dl>
            <div>
              <Button variant="destructive" disabled={busy} onClick={() => setDialog('cancel')}>
                <IconX size={16} aria-hidden="true" />
                {t.cancelPrepared}
              </Button>
            </div>
          </div>
        ) : (
          <div>
            <Button disabled={busy} onClick={() => void handlePrepare()}>
              <IconKey size={16} aria-hidden="true" />
              {busy ? t.preparing : t.prepare}
            </Button>
          </div>
        )}
      </Section>

      {prepared || snapshot.entraDomains.length > 0 ? (
        <Section title={t.registerHeading} testId="xml-key-register">
          <p className="text-xs text-slate-600">{t.registerEntraNotice}</p>
          {snapshot.entraDomains.map((domain) => (
            <div key={domain.domain} className="grid gap-2" data-domain={domain.domain}>
              <div className="flex flex-wrap items-center gap-2 text-xs">
                <span className="font-mono font-semibold">{domain.domain}</span>
                <EntraVerificationStatusBadge status={domain.verification.status} />
                {prepared ? (
                  <span>
                    {entraConfirmsPrepared(domain, prepared) ? t.entraReady : t.entraNotReady}
                  </span>
                ) : null}
                <a
                  className="font-semibold text-blue-700 hover:underline"
                  href={entraDomainURL(domain.domain)}
                >
                  {t.openDomain}
                </a>
              </div>
              <Script
                value={domain.updateCommand}
                copyLabel={t.copyUpdateCommand.replace('{domain}', domain.domain)}
                onCopy={copy}
              />
            </div>
          ))}
          {prepared && snapshot.trusts.length > 0 ? (
            <div className="grid gap-2">
              <p className="text-xs text-slate-600">{t.registerTrustNotice}</p>
              <Script
                title={t.nextCertificateLabel}
                value={prepared.signing_certificate ?? ''}
                copyLabel={t.copyNextCertificate}
                onCopy={copy}
              />
              {snapshot.trusts.map((trust) => (
                <Check
                  key={trust.id}
                  checked={confirmations.trusts.has(trust.id)}
                  onChange={(checked) => toggleTrust(trust.id, checked)}
                  label={t.trustRegistered.replace('{name}', trust.label)}
                />
              ))}
            </div>
          ) : null}
        </Section>
      ) : null}

      {prepared ? (
        <Section title={t.activateHeading} testId="xml-key-activate">
          <p className="text-xs text-slate-600">{t.activateNotice}</p>
          <Check
            checked={confirmations.overall}
            onChange={(overall) => setConfirmations((current) => ({ ...current, overall }))}
            label={t.overallConfirm}
          />
          {blockers.length > 0 ? (
            <div className="grid gap-1 text-xs text-slate-700" data-testid="xml-key-blockers">
              <p className="font-semibold">{t.blockersHeading}</p>
              <ul className="list-disc pl-5">
                {blockers.map((blocker) => (
                  <li key={blockerKey(blocker)}>{blockerText(t, blocker)}</li>
                ))}
              </ul>
            </div>
          ) : null}
          <div>
            <Button disabled={busy || blockers.length > 0} onClick={() => setDialog('activate')}>
              <IconRotateClockwise size={16} aria-hidden="true" />
              {busy ? t.activating : t.activate}
            </Button>
          </div>
        </Section>
      ) : null}

      <AfterSection keys={snapshot.keys} />

      {dialog === 'activate' && prepared ? (
        <ConfirmActionDialog
          title={t.activateConfirmTitle}
          target={prepared.kid}
          description={t.activateConfirmDescription}
          confirmLabel={t.activate}
          busy={busy}
          onClose={() => setDialog(null)}
          onConfirm={() => void handleActivate()}
        />
      ) : null}
      {dialog === 'cancel' && prepared ? (
        <ConfirmActionDialog
          title={t.cancelConfirmTitle}
          target={prepared.kid}
          description={t.cancelConfirmDescription}
          confirmLabel={t.cancelPrepared}
          busy={busy}
          onClose={() => setDialog(null)}
          onConfirm={() => void handleCancel()}
        />
      ) : null}
    </AdminShell>
  )
}

function blockerKey(blocker: ActivationBlocker): string {
  switch (blocker.kind) {
    case 'entra_not_matched':
      return `entra:${blocker.domain}`
    case 'trust_unconfirmed':
      return blocker.trust.id
    default:
      return blocker.kind
  }
}

function blockerText(t: XmlFederationKeyDictionary, blocker: ActivationBlocker): string {
  switch (blocker.kind) {
    case 'no_prepared_key':
      return t.blockerNoPrepared
    case 'entra_not_matched':
      return t.blockerEntra.replace('{domain}', blocker.domain)
    case 'trust_unconfirmed':
      return t.blockerTrust.replace('{name}', blocker.trust.label)
    case 'overall_unconfirmed':
      return t.blockerOverall
  }
}

function keyState(t: XmlFederationKeyDictionary, key: AdminKey): string {
  if (key.active) return t.stateActive
  if (key.prepared) return t.statePrepared
  return t.stateVerifying
}

function KeyTable({ keys }: { keys: AdminKey[] }) {
  const t = useDictionary(xmlFederationKeyDictionary)
  const { locale } = useLocale()
  const time = (value?: string) => (value ? new Date(value).toLocaleString(locale) : '')
  return (
    <Section title={t.certificatesHeading} testId="xml-key-certificates">
      {keys.length === 0 ? (
        <p className="text-sm text-slate-500">{t.noKeysNotice}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="text-slate-500">
              <tr>
                <th className="p-1.5">{t.kidLabel}</th>
                <th className="p-1.5">{t.stateLabel}</th>
                <th className="p-1.5">{t.fingerprintLabel}</th>
                <th className="p-1.5">{t.createdAtLabel}</th>
                <th className="p-1.5">{t.expiresAtLabel}</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((key) => (
                <tr
                  key={key.kid}
                  className="border-t border-slate-100 align-top"
                  data-kid={key.kid}
                >
                  <td className="break-all p-1.5 font-mono">{key.kid}</td>
                  <td className="p-1.5">{keyState(t, key)}</td>
                  <td className="break-all p-1.5 font-mono">
                    {key.certificate_fingerprint_sha256}
                  </td>
                  <td className="p-1.5">{time(key.created_at)}</td>
                  <td className="p-1.5">{time(key.expires_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Section>
  )
}

function ImpactSection({ snapshot }: { snapshot: RolloverSnapshot }) {
  const t = useDictionary(xmlFederationKeyDictionary)
  const groups: { title: string; items: ReactNode[] }[] = [
    {
      title: t.impactEntra,
      items: snapshot.entraDomains.map((domain) => (
        <li key={domain.domain} className="flex items-center gap-2">
          <span className="font-mono">{domain.domain}</span>
          <EntraVerificationStatusBadge status={domain.verification.status} />
        </li>
      )),
    },
    {
      title: t.impactWsFed,
      items: snapshot.trusts
        .filter((trust) => trust.kind === 'wsfed')
        .map((trust) => <li key={trust.id}>{trust.label}</li>),
    },
    {
      title: t.impactSaml,
      items: snapshot.trusts
        .filter((trust) => trust.kind === 'saml')
        .map((trust) => <li key={trust.id}>{trust.label}</li>),
    },
  ]
  return (
    <Section title={t.impactHeading} testId="xml-key-impact">
      <p className="text-xs text-slate-600">{t.impactNotice}</p>
      <div className="grid gap-3 text-xs sm:grid-cols-3">
        {groups.map((group) => (
          <div key={group.title} className="min-w-0">
            <h3 className="font-semibold text-slate-700">{group.title}</h3>
            {group.items.length > 0 ? (
              <ul className="mt-1 grid gap-1 break-all">{group.items}</ul>
            ) : (
              <p className="mt-1 text-slate-500">{t.impactNone}</p>
            )}
          </div>
        ))}
      </div>
    </Section>
  )
}

function AfterSection({ keys }: { keys: AdminKey[] }) {
  const t = useDictionary(xmlFederationKeyDictionary)
  const { locale } = useLocale()
  const previous = verifyingKeys(keys)
  const current = activeKey(keys)
  return (
    <Section title={t.recoveryHeading} testId="xml-key-recovery">
      {previous.length > 0 && current ? (
        <div className="grid gap-2 text-xs text-slate-700">
          <h3 className="font-semibold">{t.afterHeading}</h3>
          <p>{t.afterNotice}</p>
          <ul className="list-disc pl-5">
            <li>{t.afterEntra}</li>
            <li>{t.afterSignIn}</li>
            <li>{t.afterTrusts}</li>
          </ul>
          {previous.map((key) => (
            <p key={key.kid} className="font-semibold">
              {t.recoveryDeadline
                .replace('{kid}', key.kid)
                .replace(
                  '{date}',
                  key.expires_at ? new Date(key.expires_at).toLocaleString(locale) : '',
                )}
            </p>
          ))}
        </div>
      ) : null}
      <ul className="list-disc pl-5 text-xs leading-5 text-slate-700">
        <li>{t.recoveryBefore}</li>
        <li>{t.recoveryAfter}</li>
      </ul>
    </Section>
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

function Check({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: (checked: boolean) => void
  label: string
}) {
  return (
    <label className="flex items-start gap-2 text-xs font-medium text-slate-700">
      <input
        type="checkbox"
        className="mt-0.5 size-4"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
      />
      <span className="break-all">{label}</span>
    </label>
  )
}

function Script({
  title,
  value,
  copyLabel,
  onCopy,
}: {
  title?: string
  value: string
  copyLabel: string
  onCopy: (value: string) => Promise<void>
}) {
  return (
    <div className="grid min-w-0 gap-1.5">
      <div className="flex items-center justify-between gap-2">
        {title ? <h3 className="text-xs font-semibold text-slate-800">{title}</h3> : <span />}
        <Button
          type="button"
          variant="outline"
          aria-label={copyLabel}
          onClick={() => void onCopy(value)}
        >
          <IconCopy size={14} aria-hidden="true" />
        </Button>
      </div>
      <pre className="max-h-48 overflow-auto rounded-md bg-slate-50 p-3 text-xs">
        <code>{value}</code>
      </pre>
    </div>
  )
}

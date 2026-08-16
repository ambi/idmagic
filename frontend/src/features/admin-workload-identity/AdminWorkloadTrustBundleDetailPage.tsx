import { IconArrowLeft, IconPlus, IconRefresh } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  createAgentWorkloadBinding,
  deleteAgentWorkloadBinding,
  deleteWorkloadTrustBundle,
  disableAgentWorkloadBinding,
  disableWorkloadTrustBundle,
  enableAgentWorkloadBinding,
  enableWorkloadTrustBundle,
  listAgentWorkloadBindings,
  refreshWorkloadTrustBundleJWKS,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import type {
  AdminAgent,
  AdminAuditEvent,
  AgentWorkloadBinding,
  WorkloadTrustBundle,
} from '../../types'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { ConfirmDialog, editURL, listURL, EnabledStatusBadge } from './AdminWorkloadIdentityShared'
import { AttestationRejectionsCard } from './AttestationRejectionsCard'
import {
  attestationRejections,
  bindingConfirmation,
  bundleConfirmation,
  formatDateTime,
  displayNameForID,
  jwksSource,
  multipleCredentialBindings,
  rejectionsForTrustBundle,
  type Confirmation,
} from './presentation'

// pendingAction は確認ダイアログが閉じるまで保留する破壊的操作。確認文と実行を 1 つに束ねて
// 持つことで、「確認したものと実行するものがずれる」経路を作らない。
type PendingAction = Confirmation & {
  confirmLabel: string
  run: () => Promise<void>
  success: string
}

// AdminWorkloadTrustBundleDetailPage は信頼バンドルの詳細に、配下のバインディングと
// この発行者に対するアテステーション拒否を同居させる。バインディングは所属する信頼バンドルの
// 外では意味を持たないため、専用ルートを与えない。
export function AdminWorkloadTrustBundleDetailPage({
  csrfToken,
  actorUsername,
  trustBundle: initialTrustBundle,
  bindings: initialBindings,
  agents,
  rejectionEvents,
  rejectionsUnavailable = false,
  rejectionsTruncated = false,
}: {
  csrfToken: string
  actorUsername?: string
  trustBundle: WorkloadTrustBundle
  bindings: AgentWorkloadBinding[]
  agents: AdminAgent[]
  rejectionEvents: AdminAuditEvent[]
  rejectionsUnavailable?: boolean
  rejectionsTruncated?: boolean
}) {
  const [bundle, setBundle] = useState(initialTrustBundle)
  const [bindings, setBindings] = useState(initialBindings)
  const [subjectPattern, setSubjectPattern] = useState('')
  const [agentID, setAgentID] = useState(agents[0]?.id ?? '')
  const [pending, setPending] = useState<PendingAction | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminWorkloadIdentityDictionary)
  const { locale } = useLocale()

  const rejections = rejectionsForTrustBundle(attestationRejections(rejectionEvents), bundle.id)

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(success)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setBusy(false)
    }
  }

  async function reloadBindings() {
    setBindings(await listAgentWorkloadBindings(bundle.id))
  }

  // JWKS の再取得だけは成功の文言が結果 (鍵の本数) に依存するため run() を使わない。
  // 到達できなかったことは 200 応答の reachable=false で返るので、エラーとして扱う。
  async function handleRefreshJwks() {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      const result = await refreshWorkloadTrustBundleJWKS(csrfToken, bundle.id)
      if (!result.reachable) {
        setError(t.jwksUnreachableError)
        return
      }
      setBundle((previous) => ({ ...previous, jwks_cached_at: result.jwks_cached_at }))
      setNotice(t.jwksRefreshedNotice.replace('{count}', String(result.key_count ?? 0)))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setBusy(false)
    }
  }

  async function handleAddBinding(event: FormEvent) {
    event.preventDefault()
    await run(async () => {
      await createAgentWorkloadBinding(csrfToken, bundle.id, {
        subject_pattern: subjectPattern,
        agent_id: agentID,
      })
      setSubjectPattern('')
      await reloadBindings()
    }, t.bindingCreatedNotice)
  }

  async function confirmPending() {
    if (!pending) return
    const action = pending
    setPending(null)
    await run(action.run, action.success)
  }

  function askDeleteBundle() {
    setPending({
      ...bundleConfirmation('delete', bundle, bindings.length, t),
      confirmLabel: t.delete,
      success: t.deletedNotice.replace('{name}', bundle.name),
      run: async () => {
        await deleteWorkloadTrustBundle(csrfToken, bundle.id)
        window.location.assign(listURL())
      },
    })
  }

  function askDisableBundle() {
    setPending({
      ...bundleConfirmation('disable', bundle, bindings.length, t),
      confirmLabel: t.disable,
      success: t.disabledNotice.replace('{name}', bundle.name),
      run: async () => {
        await disableWorkloadTrustBundle(csrfToken, bundle.id)
        setBundle((previous) => ({ ...previous, status: 'disabled' }))
      },
    })
  }

  function askDeleteBinding(binding: AgentWorkloadBinding) {
    setPending({
      ...bindingConfirmation('delete', binding.subject_pattern, t),
      confirmLabel: t.delete,
      success: t.bindingDeletedNotice,
      run: async () => {
        await deleteAgentWorkloadBinding(csrfToken, binding.id)
        await reloadBindings()
      },
    })
  }

  function askDisableBinding(binding: AgentWorkloadBinding) {
    setPending({
      ...bindingConfirmation('disable', binding.subject_pattern, t),
      confirmLabel: t.disable,
      success: t.bindingDisabledNotice,
      run: async () => {
        await disableAgentWorkloadBinding(csrfToken, binding.id)
        await reloadBindings()
      },
    })
  }

  return (
    <AdminShell
      active="workload-identity"
      actorUsername={actorUsername}
      title={bundle.name}
      description={t.pageDescription}
      actions={
        <>
          <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.backToList}
          </Button>
          <Button variant="outline" nativeButton={false} render={<a href={editURL(bundle.id)} />}>
            {t.edit}
          </Button>
          <Button variant="outline" disabled={busy} onClick={() => void handleRefreshJwks()}>
            <IconRefresh size={16} aria-hidden="true" />
            {t.refreshJwks}
          </Button>
          {bundle.status === 'enabled' ? (
            <Button
              variant="outline"
              aria-label={t.disableBundleTitle}
              disabled={busy}
              onClick={askDisableBundle}
            >
              {t.disable}
            </Button>
          ) : (
            <Button
              variant="outline"
              aria-label={t.enableBundleAriaLabel}
              disabled={busy}
              onClick={() =>
                run(
                  async () => {
                    await enableWorkloadTrustBundle(csrfToken, bundle.id)
                    setBundle((previous) => ({ ...previous, status: 'enabled' }))
                  },
                  t.enabledNotice.replace('{name}', bundle.name),
                )
              }
            >
              {t.enable}
            </Button>
          )}
          <Button
            variant="destructive"
            aria-label={t.deleteBundleTitle}
            disabled={busy}
            onClick={askDeleteBundle}
          >
            {t.delete}
          </Button>
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <Card className="p-4">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-semibold text-slate-900">{bundle.name}</h2>
          <EnabledStatusBadge status={bundle.status} t={t} />
        </div>
        <dl className="mt-4 grid grid-cols-[minmax(0,14rem)_minmax(0,1fr)] gap-y-2 text-sm">
          <dt className="text-slate-500">{t.trustDomainDetailLabel}</dt>
          <dd className="text-slate-800">{bundle.trust_domain}</dd>
          <dt className="text-slate-500">{t.issuerDetailLabel}</dt>
          <dd className="break-all font-mono text-xs text-slate-800">{bundle.issuer}</dd>
          <dt className="text-slate-500">{t.jwksSourceLabel}</dt>
          <dd className="break-all font-mono text-xs text-slate-800">{jwksSource(bundle, t)}</dd>
          <dt className="text-slate-500">{t.jwksCachedAtLabel}</dt>
          <dd className="text-slate-800">{formatDateTime(bundle.jwks_cached_at, locale)}</dd>
          <dt className="text-slate-500">{t.acceptedAudiencesDetailLabel}</dt>
          <dd className="text-slate-800">{bundle.accepted_audiences.join(', ')}</dd>
          <dt className="text-slate-500">{t.maxTtlDetailLabel}</dt>
          <dd className="text-slate-800">
            {bundle.max_subject_token_ttl_seconds} {t.seconds}
          </dd>
          <dt className="text-slate-500">{t.createdAtLabel}</dt>
          <dd className="text-slate-800">{formatDateTime(bundle.created_at, locale)}</dd>
          <dt className="text-slate-500">{t.updatedAtLabel}</dt>
          <dd className="text-slate-800">{formatDateTime(bundle.updated_at, locale)}</dd>
        </dl>
      </Card>

      <Card className="mt-6 overflow-hidden">
        <div className="border-b border-slate-100 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-900">{t.bindingsTitle}</h2>
          <p className="mt-1 max-w-[80ch] text-xs leading-5 text-slate-500">
            {t.bindingsDescription}
          </p>
        </div>
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">{t.tableHeaderSubjectPattern}</th>
              <th className="px-4 py-3">{t.tableHeaderAgent}</th>
              <th className="px-4 py-3">{t.tableHeaderStatus}</th>
              <th className="px-4 py-3 text-right" />
            </tr>
          </thead>
          <tbody>
            {bindings.map((binding) => (
              <tr key={binding.id} className="border-t border-slate-100">
                <td className="break-all px-4 py-3 font-mono text-xs text-slate-800">
                  {binding.subject_pattern}
                </td>
                <td className="px-4 py-3 text-slate-800">
                  {displayNameForID(binding.agent_id, agents)}
                  {/* 交換は資格情報バインディングの最初の 1 つを採る。規則は決めないが、
                      どれが使われるか定まらない状態そのものは見えるようにする。 */}
                  {multipleCredentialBindings(binding.agent_id, agents) ? (
                    <span className="mt-0.5 block text-xs text-amber-700">
                      {t.multipleCredentialsWarning}
                    </span>
                  ) : null}
                </td>
                <td className="px-4 py-3">
                  <EnabledStatusBadge status={binding.status} t={t} />
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="flex justify-end gap-2">
                    {binding.status === 'enabled' ? (
                      <Button
                        variant="outline"
                        size="sm"
                        aria-label={`${t.disable}: ${binding.subject_pattern}`}
                        disabled={busy}
                        onClick={() => askDisableBinding(binding)}
                      >
                        {t.disable}
                      </Button>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        aria-label={`${t.enable}: ${binding.subject_pattern}`}
                        disabled={busy}
                        onClick={() =>
                          run(async () => {
                            await enableAgentWorkloadBinding(csrfToken, binding.id)
                            await reloadBindings()
                          }, t.bindingEnabledNotice)
                        }
                      >
                        {t.enable}
                      </Button>
                    )}
                    <Button
                      variant="destructive"
                      size="sm"
                      aria-label={`${t.delete}: ${binding.subject_pattern}`}
                      disabled={busy}
                      onClick={() => askDeleteBinding(binding)}
                    >
                      {t.delete}
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {bindings.length === 0 ? (
          <p className="px-4 py-6 text-center text-sm text-slate-500">{t.bindingsEmptyNotice}</p>
        ) : null}
        <form
          className="flex flex-wrap items-end gap-3 border-t border-slate-100 bg-slate-50 p-4"
          onSubmit={handleAddBinding}
        >
          <div className="flex min-w-[22rem] flex-1 flex-col gap-1.5">
            <Label htmlFor="subject_pattern">{t.subjectPatternLabel}</Label>
            <Input
              id="subject_pattern"
              value={subjectPattern}
              placeholder={t.subjectPatternPlaceholder}
              onChange={(event) => setSubjectPattern(event.target.value)}
              required
            />
          </div>
          <div className="flex w-64 flex-col gap-1.5">
            <Label htmlFor="agent_id">{t.agentLabel}</Label>
            <Select
              id="agent_id"
              value={agentID}
              onValueChange={setAgentID}
              options={agents.map((agent) => ({ value: agent.id, label: agent.name }))}
              aria-label={t.agentLabel}
              disabled={agents.length === 0}
            />
          </div>
          <Button type="submit" disabled={busy || agents.length === 0}>
            <IconPlus size={16} aria-hidden="true" />
            {t.addBinding}
          </Button>
          {agents.length === 0 ? (
            <p className="w-full text-xs text-slate-500">{t.noAgentsNotice}</p>
          ) : null}
        </form>
      </Card>

      <AttestationRejectionsCard
        rejections={rejections}
        unavailable={rejectionsUnavailable}
        truncated={rejectionsTruncated}
        t={t}
      />

      {pending ? (
        <ConfirmDialog
          title={pending.title}
          message={pending.message}
          confirmLabel={pending.confirmLabel}
          cancelLabel={t.cancel}
          onCancel={() => setPending(null)}
          onConfirm={() => void confirmPending()}
          busy={busy}
        />
      ) : null}
    </AdminShell>
  )
}

import { IconPlus, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  deleteCrossAppAccessIssuePolicy,
  deleteCrossAppAccessRedeemPolicy,
  listCrossAppAccessIssuePolicies,
  listCrossAppAccessRedeemPolicies,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { CrossAppAccessIssuePolicy, CrossAppAccessRedeemPolicy } from '../../types'
import { adminCrossAppAccessDictionary } from './AdminCrossAppAccessPage.i18n'

// AdminCrossAppAccessPage はアプリ間アクセスの許可の一覧と削除である。作成は専用ルートに置く。
export function AdminCrossAppAccessPage({
  csrfToken,
  actorUsername,
  issuePolicies,
  redeemPolicies,
}: {
  csrfToken: string
  actorUsername?: string
  issuePolicies: CrossAppAccessIssuePolicy[]
  redeemPolicies: CrossAppAccessRedeemPolicy[]
}) {
  const [issueItems, setIssueItems] = useState(issuePolicies)
  const [redeemItems, setRedeemItems] = useState(redeemPolicies)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminCrossAppAccessDictionary)
  const common = useDictionary(commonDictionary)
  const [pending, setPending] = useState<{
    target: string
    action: () => Promise<void>
    onDeleted: () => void
  } | null>(null)
  const [busy, setBusy] = useState(false)

  async function remove(action: () => Promise<void>, onDeleted: () => void) {
    setError('')
    setBusy(true)
    try {
      await action()
      onDeleted()
      setNotice(t.deletedNotice)
      setPending(null)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.deleteFailedError)
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="cross-app-access"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      list={{
        total: issueItems.length + redeemItems.length,
        displayed: issueItems.length + redeemItems.length,
        onRefresh: async () => {
          const [issue, redeem] = await Promise.all([
            listCrossAppAccessIssuePolicies(),
            listCrossAppAccessRedeemPolicies(),
          ])
          setIssueItems(issue)
          setRedeemItems(redeem)
        },
      }}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <section className="flex flex-col gap-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="text-base font-semibold text-slate-900">{t.issueSectionTitle}</h2>
            <p className="mt-0.5 text-xs leading-5 text-slate-500">{t.issueSectionDescription}</p>
          </div>
          <Button
            nativeButton={false}
            render={<a href={tenantURL('/admin/cross-app-access/new-issue')} />}
          >
            <IconPlus size={17} aria-hidden="true" />
            {t.addIssuePolicy}
          </Button>
        </div>
        {issueItems.length === 0 ? (
          <Card className="p-8 text-center text-sm text-slate-500">{t.emptyIssueNotice}</Card>
        ) : (
          issueItems.map((policy) => (
            <Card key={policy.id} className="flex items-start justify-between gap-3 p-4">
              <div className="min-w-0">
                <p className="break-all font-mono text-sm font-semibold text-slate-900">
                  {policy.client_id} → {policy.audience}
                </p>
                <p className="mt-1 break-all text-xs leading-5 text-slate-500">
                  {policy.resources.join(' ')}
                </p>
                <p className="mt-1 break-all text-xs leading-5 text-slate-500">
                  {policy.scopes.join(' ')}
                </p>
              </div>
              <Button
                variant="destructive"
                onClick={() =>
                  setPending({
                    target: `${policy.client_id} → ${policy.audience}`,
                    action: () => deleteCrossAppAccessIssuePolicy(csrfToken, policy.id),
                    onDeleted: () =>
                      setIssueItems((items) => items.filter((item) => item.id !== policy.id)),
                  })
                }
              >
                <IconTrash size={16} aria-hidden="true" />
                {t.delete}
              </Button>
            </Card>
          ))
        )}
      </section>

      <section className="flex flex-col gap-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="text-base font-semibold text-slate-900">{t.redeemSectionTitle}</h2>
            <p className="mt-0.5 text-xs leading-5 text-slate-500">{t.redeemSectionDescription}</p>
          </div>
          <Button
            nativeButton={false}
            render={<a href={tenantURL('/admin/cross-app-access/new-redeem')} />}
          >
            <IconPlus size={17} aria-hidden="true" />
            {t.addRedeemPolicy}
          </Button>
        </div>
        {redeemItems.length === 0 ? (
          <Card className="p-8 text-center text-sm text-slate-500">{t.emptyRedeemNotice}</Card>
        ) : (
          redeemItems.map((policy) => (
            <Card key={policy.id} className="flex items-start justify-between gap-3 p-4">
              <div className="min-w-0">
                <p className="break-all font-mono text-sm font-semibold text-slate-900">
                  {policy.provider_id} → {policy.client_id}
                </p>
                <p className="mt-1 break-all text-xs leading-5 text-slate-500">
                  {policy.scopes.join(' ')}
                </p>
              </div>
              <Button
                variant="destructive"
                onClick={() =>
                  setPending({
                    target: `${policy.provider_id} → ${policy.client_id}`,
                    action: () => deleteCrossAppAccessRedeemPolicy(csrfToken, policy.id),
                    onDeleted: () =>
                      setRedeemItems((items) => items.filter((item) => item.id !== policy.id)),
                  })
                }
              >
                <IconTrash size={16} aria-hidden="true" />
                {t.delete}
              </Button>
            </Card>
          ))
        )}
      </section>
      {pending ? (
        <ConfirmActionDialog
          title={t.delete}
          target={pending.target}
          description={common.deleteResourceDescription}
          confirmLabel={t.delete}
          busy={busy}
          error={error}
          onClose={() => setPending(null)}
          onConfirm={() => void remove(pending.action, pending.onDeleted)}
        />
      ) : null}
    </AdminShell>
  )
}

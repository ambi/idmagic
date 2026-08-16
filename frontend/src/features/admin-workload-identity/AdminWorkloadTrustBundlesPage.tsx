import { IconPlus, IconShieldCheck } from '@tabler/icons-react'
import { AdminShell } from '../../components/AdminShell'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminAuditEvent, WorkloadTrustBundle } from '../../types'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { detailURL, EnabledStatusBadge, newURL } from './AdminWorkloadIdentityShared'
import { AttestationRejectionsCard } from './AttestationRejectionsCard'
import { attestationRejections, formatDateTime } from './presentation'

// AdminWorkloadTrustBundlesPage は信頼バンドルの参照専用一覧。登録は専用ルート (new)、
// 状態変更と削除は詳細画面に置く。一覧から直接消せると、カスケードで消えるバインディングを
// 見ないまま操作できてしまう。
export function AdminWorkloadTrustBundlesPage({
  actorUsername,
  trustBundles,
  rejectionEvents,
  rejectionsUnavailable = false,
  rejectionsTruncated = false,
}: {
  actorUsername?: string
  trustBundles: WorkloadTrustBundle[]
  rejectionEvents: AdminAuditEvent[]
  rejectionsUnavailable?: boolean
  rejectionsTruncated?: boolean
}) {
  const t = useDictionary(adminWorkloadIdentityDictionary)
  const { locale } = useLocale()
  const rejections = attestationRejections(rejectionEvents)

  return (
    <AdminShell
      active="workload-identity"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button nativeButton={false} render={<a href={newURL()} />}>
          <IconPlus size={17} aria-hidden="true" />
          {t.registerTrustBundle}
        </Button>
      }
    >
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">{t.tableHeaderName}</th>
              <th className="px-4 py-3">{t.tableHeaderTrustDomain}</th>
              <th className="px-4 py-3">{t.tableHeaderIssuer}</th>
              <th className="px-4 py-3">{t.tableHeaderStatus}</th>
              <th className="px-4 py-3">{t.tableHeaderJwksCachedAt}</th>
              <th className="px-4 py-3 text-right" />
            </tr>
          </thead>
          <tbody>
            {trustBundles.map((bundle) => (
              <tr key={bundle.id} className="border-t border-slate-100 hover:bg-slate-50">
                <td className="px-4 py-3 font-semibold text-slate-900">{bundle.name}</td>
                <td className="px-4 py-3 text-slate-600">{bundle.trust_domain}</td>
                <td className="break-all px-4 py-3 font-mono text-xs text-slate-600">
                  {bundle.issuer}
                </td>
                <td className="px-4 py-3">
                  <EnabledStatusBadge status={bundle.status} t={t} />
                </td>
                <td className="px-4 py-3 text-xs text-slate-600">
                  {formatDateTime(bundle.jwks_cached_at, locale)}
                </td>
                <td className="px-4 py-3 text-right">
                  <Button
                    variant="outline"
                    nativeButton={false}
                    render={<a href={detailURL(bundle.id)} />}
                  >
                    {t.detail}
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {trustBundles.length === 0 ? (
          <div className="flex min-h-40 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
            <IconShieldCheck size={24} className="text-slate-400" aria-hidden="true" />
            <p className="mt-3">{t.emptyNotice}</p>
          </div>
        ) : null}
      </Card>

      {/* テナント全体の直近の拒否。未登録の発行者による拒否は trustBundleId を持たないため、
          どの詳細画面にも現れない。ここが唯一それを見られる場所になる。 */}
      <AttestationRejectionsCard
        rejections={rejections}
        unavailable={rejectionsUnavailable}
        truncated={rejectionsTruncated}
        trustBundles={trustBundles}
        t={t}
      />
    </AdminShell>
  )
}

import { Alert } from '../../components/ui/alert'
import { Card } from '../../components/ui/card'
import { useLocale } from '../../lib/i18n'
import type { WorkloadTrustBundle } from '../../types'
import {
  displayNameForID,
  formatDateTime,
  rejectionReasonLabel,
  type AttestationRejection,
  type Dictionary,
} from './presentation'

// AttestationRejectionsCard は WorkloadAttestationRejected を理由コード付きで見せる。
// 専用の読み取り経路は持たず、監査検索の結果を渡してもらう。
//
// trustBundles を渡すと、拒否がどの信頼バンドルに帰属したかの列を出す。テナント全体を見る
// 一覧向けで、1 バンドルに絞った詳細画面では渡さない (全行が同じ値になるため)。
export function AttestationRejectionsCard({
  rejections,
  unavailable,
  truncated = false,
  trustBundles,
  t,
}: {
  rejections: AttestationRejection[]
  unavailable: boolean
  truncated?: boolean
  trustBundles?: WorkloadTrustBundle[]
  t: Dictionary
}) {
  const { locale } = useLocale()

  return (
    <Card className="mt-6 overflow-hidden">
      <div className="border-b border-slate-100 px-4 py-3">
        <h2 className="text-sm font-semibold text-slate-900">{t.rejectionsTitle}</h2>
        <p className="mt-1 text-xs leading-5 text-slate-500">{t.rejectionsDescription}</p>
      </div>
      {unavailable ? (
        <div className="p-4">
          <Alert variant="destructive">{t.rejectionsLoadFailedError}</Alert>
        </div>
      ) : rejections.length === 0 ? (
        <p className="px-4 py-6 text-center text-sm text-slate-500">
          {/* 窓を使い切っていた場合、「無い」ではなく「この窓には無い」としか言えない。 */}
          {truncated ? t.rejectionsTruncatedEmptyNotice : t.rejectionsEmptyNotice}
        </p>
      ) : (
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">{t.tableHeaderOccurredAt}</th>
              <th className="px-4 py-3">{t.tableHeaderReason}</th>
              {trustBundles ? <th className="px-4 py-3">{t.tableHeaderName}</th> : null}
            </tr>
          </thead>
          <tbody>
            {rejections.map((rejection) => (
              <tr key={rejection.id} className="border-t border-slate-100">
                <td className="px-4 py-3 text-xs text-slate-600">
                  {formatDateTime(rejection.occurredAt, locale)}
                </td>
                <td className="px-4 py-3 text-slate-800">
                  {rejectionReasonLabel(rejection.reason, t)}
                </td>
                {trustBundles ? (
                  <td className="px-4 py-3 text-xs text-slate-600">
                    {rejection.trustBundleId
                      ? displayNameForID(rejection.trustBundleId, trustBundles)
                      : t.rejectionsUnknownBundle}
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Card>
  )
}

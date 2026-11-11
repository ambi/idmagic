import { useDictionary } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import type { EntraVerificationStatus } from '../../types'
import {
  adminEntraFederationDictionary,
  type AdminEntraFederationDictionary,
} from './AdminEntraFederationPage.i18n'

const styles: Record<EntraVerificationStatus, string> = {
  unverified: 'border-slate-300 bg-slate-50 text-slate-700',
  matched: 'border-emerald-300 bg-emerald-50 text-emerald-800',
  mismatched: 'border-red-300 bg-red-50 text-red-800',
  stale: 'border-amber-300 bg-amber-50 text-amber-900',
}

export function entraStatusLabel(
  t: AdminEntraFederationDictionary,
  status: EntraVerificationStatus,
) {
  return {
    unverified: t.statusUnverified,
    matched: t.statusMatched,
    mismatched: t.statusMismatched,
    stale: t.statusStale,
  }[status]
}

// 保存しただけのドメインを反映済みに見せないため、状態は照合の結果の四つだけを文字で示す。
export function EntraVerificationStatusBadge({ status }: { status: EntraVerificationStatus }) {
  const t = useDictionary(adminEntraFederationDictionary)
  return (
    <span
      className={cn('rounded-full border px-2 py-0.5 text-xs font-medium', styles[status])}
      data-status={status}
    >
      {entraStatusLabel(t, status)}
    </span>
  )
}

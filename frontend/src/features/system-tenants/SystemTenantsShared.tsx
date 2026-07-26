import type { AdminTenant } from '../../types'

export function StatusBadge({ status }: { status: AdminTenant['status'] }) {
  return status === 'active' ? (
    <span className="rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700">
      active
    </span>
  ) : (
    <span className="rounded-md bg-rose-50 px-2 py-0.5 text-xs font-semibold text-rose-700">
      disabled
    </span>
  )
}

export function formatDate(value: string | undefined, locale: 'ja' | 'en'): string {
  if (!value) return '—'
  try {
    return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
  } catch {
    return value
  }
}

import { useDictionary } from '../../lib/i18n'
import type { AdminAgent } from '../../types'
import { adminAgentsDictionary, type AdminAgentsDictionary } from './AdminAgentsPage.i18n'

const STATUS_STYLES: Record<AdminAgent['status'], string> = {
  active: 'bg-emerald-100 text-emerald-700',
  disabled: 'bg-slate-200 text-slate-600',
  killed: 'bg-rose-100 text-rose-700',
}

export function kindLabel(kind: AdminAgent['kind'], t: AdminAgentsDictionary) {
  return kind === 'autonomous' ? t.kindAutonomous : t.kindSupervised
}

function statusLabel(status: AdminAgent['status'], t: AdminAgentsDictionary) {
  return { active: t.statusActive, disabled: t.statusDisabled, killed: t.statusKilled }[status]
}

export function StatusBadge({ status }: { status: AdminAgent['status'] }) {
  const t = useDictionary(adminAgentsDictionary)
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${STATUS_STYLES[status]}`}
    >
      {statusLabel(status, t)}
    </span>
  )
}

export function parseRoles(value: string) {
  return [
    ...new Set(
      value
        .split(',')
        .map((role) => role.trim())
        .filter(Boolean),
    ),
  ]
}

export function optionalValue(value: FormDataEntryValue | null) {
  const normalized = String(value ?? '').trim()
  return normalized || undefined
}

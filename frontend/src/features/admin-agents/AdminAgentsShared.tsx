import { useDictionary } from '../../lib/i18n'
import type { AdminAgent, AdminUser } from '../../types'
import { adminAgentsDictionary, type AdminAgentsDictionary } from './AdminAgentsPage.i18n'

const STATUS_STYLES: Record<AdminAgent['status'], string> = {
  active: 'bg-emerald-100 text-emerald-700',
  disabled: 'bg-slate-200 text-slate-600',
  killed: 'bg-rose-100 text-rose-700',
}

// OwnerSelect は所有者を利用者名で選ばせる。一覧にない現在の所有者も、別の値に見せず ID のまま残す。
export function OwnerSelect({
  id,
  name,
  value,
  onChange,
  users,
  emptyLabel,
  disabled,
}: {
  id: string
  name?: string
  value?: string
  onChange?: (value: string) => void
  users: Pick<AdminUser, 'id' | 'preferred_username'>[]
  emptyLabel?: string
  disabled?: boolean
}) {
  const missing = value && !users.some((user) => user.id === value)
  return (
    <select
      id={id}
      name={name}
      value={value}
      onChange={onChange ? (event) => onChange(event.target.value) : undefined}
      disabled={disabled}
      className="h-9 rounded-md border border-slate-300 bg-white px-2 text-sm"
    >
      {emptyLabel !== undefined ? <option value="">{emptyLabel}</option> : null}
      {missing ? <option value={value}>{value}</option> : null}
      {users.map((user) => (
        <option key={user.id} value={user.id}>
          {user.preferred_username}
        </option>
      ))}
    </select>
  )
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

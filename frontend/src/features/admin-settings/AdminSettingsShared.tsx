import { cn } from '../../lib/utils'
import type { AdminSettings } from '../../types'
import type { AdminSettingsDictionary } from './AdminSettingsPage.i18n'

export function displayNameError(value: string, t: AdminSettingsDictionary): string | null {
  return value.trim() ? null : t.displayNameRequiredError
}

export function passwordPolicyOverride(
  minLength: string,
  maxLength: string,
  historyDepth: string,
): NonNullable<AdminSettings['password_policy_override']> {
  const policy: NonNullable<AdminSettings['password_policy_override']> = {}
  if (minLength.trim()) policy.min_length = Number.parseInt(minLength, 10)
  if (maxLength.trim()) policy.max_length = Number.parseInt(maxLength, 10)
  if (historyDepth.trim()) policy.history_depth = Number.parseInt(historyDepth, 10)
  return policy
}

export function ReadSetting({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="rounded-lg border border-slate-200/80 bg-white/70 px-3 py-2.5">
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className={cn('mt-0.5 text-sm font-medium text-slate-900', mono && 'font-mono')}>
        {value}
      </dd>
    </div>
  )
}

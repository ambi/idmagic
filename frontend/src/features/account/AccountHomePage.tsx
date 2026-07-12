import {
  IconAlertTriangle,
  IconClockHour4,
  IconShieldCheck,
  IconShieldOff,
  IconUser,
} from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { AccountShell } from '../../components/AccountShell'
import { Card } from '../../components/ui/card'
import { useDictionary, useFormatters } from '../../lib/i18n'
import { domainLabelsDictionary } from '../../lib/i18n/domainLabels.i18n'
import { requiredActionLabel, type AccountSummary } from '../../types'
import { accountHomeDictionary } from './AccountHomePage.i18n'

export function formatAccountSummaryDateTime(
  value: string | undefined,
  locale = 'ja',
  empty = '記録なし',
): string {
  if (!value) {
    return empty
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return empty
  }
  return date.toLocaleString(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function AccountHomePage({
  summary,
  isAdmin,
}: {
  summary: AccountSummary
  isAdmin: boolean
}) {
  const displayName = summary.name?.trim() || summary.preferred_username
  const tLabels = useDictionary(domainLabelsDictionary)
  const t = useDictionary(accountHomeDictionary)
  const { formatDateTime } = useFormatters()
  return (
    <AccountShell
      active="home"
      username={summary.preferred_username}
      isAdmin={isAdmin}
      title={t.greeting.replace('{name}', displayName)}
      description={t.description}
    >
      {summary.required_actions.length > 0 ? (
        <Card className="flex items-start gap-3 border-amber-200 bg-amber-50/70 p-4">
          <IconAlertTriangle
            className="mt-0.5 shrink-0 text-amber-600"
            size={20}
            aria-hidden="true"
          />
          <div>
            <p className="text-sm font-semibold text-amber-900">{t.requiredActions}</p>
            <ul className="mt-1.5 flex flex-wrap gap-2">
              {summary.required_actions.map((action) => (
                <li
                  key={action}
                  className="rounded-md bg-amber-100 px-2 py-1 text-xs font-medium text-amber-900"
                >
                  {requiredActionLabel(action, tLabels)}
                </li>
              ))}
            </ul>
          </div>
        </Card>
      ) : null}

      <section className="grid gap-4 sm:grid-cols-2" aria-label={t.accountStatus}>
        <SummaryCard
          icon={summary.mfa_enrolled ? <IconShieldCheck size={20} /> : <IconShieldOff size={20} />}
          tone={summary.mfa_enrolled ? 'ok' : 'warn'}
          label={t.mfa}
          value={summary.mfa_enrolled ? t.enrolled : t.notEnrolled}
        />
        <SummaryCard
          icon={<IconUser size={20} />}
          tone="neutral"
          label={t.emailAddress}
          value={summary.email ?? t.notSet}
          hint={summary.email ? (summary.email_verified ? t.verified : t.unverified) : undefined}
        />
        <SummaryCard
          icon={<IconClockHour4 size={20} />}
          tone="neutral"
          label={t.lastLogin}
          value={summary.last_login_at ? formatDateTime(summary.last_login_at) : t.noRecord}
        />
        <SummaryCard
          icon={<IconUser size={20} />}
          tone="neutral"
          label={t.username}
          value={summary.preferred_username}
        />
      </section>
    </AccountShell>
  )
}

function SummaryCard({
  icon,
  tone,
  label,
  value,
  hint,
}: {
  icon: ReactNode
  tone: 'ok' | 'warn' | 'neutral'
  label: string
  value: string
  hint?: string
}) {
  const toneClass =
    tone === 'ok'
      ? 'bg-emerald-50 text-emerald-700'
      : tone === 'warn'
        ? 'bg-amber-50 text-amber-700'
        : 'bg-slate-100 text-slate-600'
  return (
    <Card className="flex items-start gap-3 p-5">
      <span className={`flex size-10 shrink-0 items-center justify-center rounded-lg ${toneClass}`}>
        {icon}
      </span>
      <div className="min-w-0">
        <p className="text-xs font-medium text-slate-500">{label}</p>
        <p className="mt-1 truncate text-sm font-semibold text-slate-900">{value}</p>
        {hint ? <p className="mt-0.5 text-xs text-slate-500">{hint}</p> : null}
      </div>
    </Card>
  )
}

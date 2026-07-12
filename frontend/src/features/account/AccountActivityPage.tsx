import {
  IconDeviceDesktop,
  IconDeviceLaptop,
  IconInfoCircle,
  IconLogin2,
} from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, revokeAccountSession, revokeOtherAccountSessions } from '../../api'
import { AccountShell } from '../../components/AccountShell'
import { StepUpCancelledError, useStepUpGuard } from '../../components/StepUpDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import type { AccountSession, AccountSignInActivity } from '../../types'
import { useDictionary, useFormatters } from '../../lib/i18n'
import { accountActivityDictionary } from './AccountActivityPage.i18n'

export function formatAccountActivityDateTime(value: string): string {
  return new Date(value).toLocaleString('ja-JP', { dateStyle: 'medium', timeStyle: 'short' })
}

const amrLabels: Record<string, string> = accountActivityDictionary.ja

export function accountActivityMethodSummary(
  amr: string[],
  labels: Record<string, string> = amrLabels,
): string {
  if (amr.length === 0) return labels.unknownMethod
  return amr.map((code) => labels[code] ?? code).join(' + ')
}

function errorMessage(cause: unknown, fallback: string): string {
  return cause instanceof AuthenticationAPIError ? cause.message : fallback
}

function SessionRow({
  session,
  busy,
  onRevoke,
}: {
  session: AccountSession
  busy: boolean
  onRevoke: () => void
}) {
  const t = useDictionary(accountActivityDictionary)
  const { formatDateTime } = useFormatters()
  return (
    <li className="flex items-start justify-between gap-3 px-5 py-4">
      <div className="flex min-w-0 items-start gap-3">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconDeviceDesktop size={18} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-sm font-semibold text-slate-900">{t.session}</p>
            {session.current ? (
              <span className="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700">
                {t.currentSession}
              </span>
            ) : null}
          </div>
          <p className="mt-0.5 text-sm text-slate-600">
            {accountActivityMethodSummary(session.amr, t)}
          </p>
          <p className="mt-1 text-xs text-slate-500">
            {t.started.replace('{date}', formatDateTime(session.started_at))}
          </p>
        </div>
      </div>
      {session.current ? (
        <span className="shrink-0 self-center text-xs text-slate-400">{t.thisDevice}</span>
      ) : (
        <Button
          type="button"
          variant="outline"
          className="h-9 shrink-0 self-center px-3 text-xs"
          disabled={busy}
          onClick={onRevoke}
        >
          {busy ? t.ending : t.end}
        </Button>
      )}
    </li>
  )
}

function ActivityRow({ activity }: { activity: AccountSignInActivity }) {
  const t = useDictionary(accountActivityDictionary)
  const { formatDateTime } = useFormatters()
  return (
    <li className="flex items-start gap-3 px-5 py-4">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
        <IconLogin2 size={18} aria-hidden="true" />
      </span>
      <div className="min-w-0">
        <p className="text-sm font-semibold text-slate-900">{t.signIn}</p>
        <p className="mt-0.5 text-sm text-slate-600">
          {accountActivityMethodSummary(activity.amr, t)}
        </p>
        <p className="mt-1 text-xs text-slate-500">{formatDateTime(activity.occurred_at)}</p>
      </div>
    </li>
  )
}

export function AccountActivityPage({
  csrfToken,
  username,
  isAdmin,
  activities,
  sessions: initialSessions,
}: {
  csrfToken: string
  username: string
  activities: AccountSignInActivity[]
  sessions: AccountSession[]
  isAdmin: boolean
}) {
  const t = useDictionary(accountActivityDictionary)
  const [sessions, setSessions] = useState(initialSessions)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [busyOthers, setBusyOthers] = useState(false)
  const [error, setError] = useState('')
  const { guard, dialog } = useStepUpGuard(csrfToken)

  async function handleRevoke(id: string) {
    setBusyId(id)
    setError('')
    try {
      await revokeAccountSession(csrfToken, id)
      setSessions((current) => current.filter((session) => session.id !== id))
    } catch (cause) {
      setError(errorMessage(cause, t.revokeFailed))
    } finally {
      setBusyId(null)
    }
  }

  async function handleRevokeOthers() {
    setBusyOthers(true)
    setError('')
    try {
      await guard(() => revokeOtherAccountSessions(csrfToken))
      setSessions((current) => current.filter((session) => session.current))
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(errorMessage(cause, t.revokeOthersFailed))
    } finally {
      setBusyOthers(false)
    }
  }

  return (
    <AccountShell
      active="activity"
      username={username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      <SessionsSection
        sessions={sessions}
        busyId={busyId}
        busyOthers={busyOthers}
        onRevoke={handleRevoke}
        onRevokeOthers={handleRevokeOthers}
      />

      <ActivityHistorySection activities={activities} />

      <div className="flex items-start gap-3 rounded-xl bg-slate-50 p-3.5 text-xs leading-5 text-slate-600">
        <IconInfoCircle className="mt-0.5 shrink-0 text-slate-500" size={17} aria-hidden="true" />
        <p>{t.info}</p>
      </div>
      {dialog}
    </AccountShell>
  )
}

export function SessionsSection({
  sessions,
  busyId,
  busyOthers,
  onRevoke,
  onRevokeOthers,
}: {
  sessions: AccountSession[]
  busyId: string | null
  busyOthers: boolean
  onRevoke: (id: string) => void
  onRevokeOthers: () => void
}) {
  const t = useDictionary(accountActivityDictionary)
  const otherCount = sessions.filter((session) => !session.current).length

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-900">{t.activeSessions}</h2>
        {otherCount > 0 ? (
          <Button
            type="button"
            variant="outline"
            className="h-9 px-3 text-xs"
            disabled={busyOthers}
            onClick={onRevokeOthers}
          >
            {busyOthers ? t.ending : t.endOther}
          </Button>
        ) : null}
      </div>
      <Card className="overflow-hidden p-0">
        {sessions.length === 0 ? (
          <div className="flex items-center gap-3 px-5 py-8 text-sm text-slate-600">
            <IconDeviceLaptop size={20} className="text-slate-400" aria-hidden="true" />
            {t.noSessions}
          </div>
        ) : (
          <ul className="divide-y divide-slate-100">
            {sessions.map((session) => (
              <SessionRow
                key={session.id}
                session={session}
                busy={busyId === session.id}
                onRevoke={() => onRevoke(session.id)}
              />
            ))}
          </ul>
        )}
      </Card>
    </section>
  )
}

export function ActivityHistorySection({ activities }: { activities: AccountSignInActivity[] }) {
  const t = useDictionary(accountActivityDictionary)
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-sm font-semibold text-slate-900">{t.history}</h2>
      <Card className="overflow-hidden p-0">
        {activities.length === 0 ? (
          <div className="flex items-center gap-3 px-5 py-8 text-sm text-slate-600">
            <IconDeviceLaptop size={20} className="text-slate-400" aria-hidden="true" />
            {t.noHistory}
          </div>
        ) : (
          <ul className="divide-y divide-slate-100">
            {activities.map((activity) => (
              <ActivityRow
                key={`${activity.occurred_at}-${activity.amr.join('-')}`}
                activity={activity}
              />
            ))}
          </ul>
        )}
      </Card>
    </section>
  )
}

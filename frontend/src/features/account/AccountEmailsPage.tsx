import { IconCircleCheck, IconCircleDashed, IconMail } from '@tabler/icons-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import { AuthenticationAPIError, requestEmailChange } from '../../api'
import { AccountShell } from '../../components/AccountShell'
import { StepUpCancelledError, useStepUpGuard } from '../../components/StepUpDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { accountEmailsDictionary } from './AccountEmailsPage.i18n'

export function AccountEmailsPage({
  csrfToken,
  email,
  emailVerified,
  isAdmin,
}: {
  csrfToken: string
  email?: string
  emailVerified: boolean
  isAdmin: boolean
}) {
  const t = useDictionary(accountEmailsDictionary)
  const [newEmail, setNewEmail] = useState('')
  const [editing, setEditing] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [sentTo, setSentTo] = useState('')
  const { guard, dialog } = useStepUpGuard(csrfToken)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError('')
    setSentTo('')
    const target = newEmail.trim()
    try {
      await guard(() => requestEmailChange(csrfToken, target))
      setSentTo(target)
      setNewEmail('')
      setEditing(false)
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.requestFailed)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AccountEmailsPresentation
      email={email}
      emailVerified={emailVerified}
      isAdmin={isAdmin}
      newEmail={newEmail}
      editing={editing}
      submitting={submitting}
      error={error}
      sentTo={sentTo}
      dialog={dialog}
      onStartEdit={() => setEditing(true)}
      onCancelEdit={() => {
        setNewEmail('')
        setEditing(false)
      }}
      onNewEmailChange={setNewEmail}
      onSubmit={handleSubmit}
    />
  )
}

export function AccountEmailsPresentation({
  email,
  emailVerified,
  isAdmin,
  newEmail,
  editing,
  submitting,
  error,
  sentTo,
  dialog,
  onStartEdit,
  onCancelEdit,
  onNewEmailChange,
  onSubmit,
}: {
  email?: string
  emailVerified: boolean
  isAdmin: boolean
  newEmail: string
  editing: boolean
  submitting: boolean
  error: string
  sentTo: string
  dialog: ReactNode
  onStartEdit: () => void
  onCancelEdit: () => void
  onNewEmailChange: (value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  const t = useDictionary(accountEmailsDictionary)
  return (
    <AccountShell
      active="emails"
      username={email ?? 'account'}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      <Card className="flex items-start gap-3 p-5">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconMail size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium text-slate-500">{t.current}</p>
          <p className="mt-1 truncate text-sm font-semibold text-slate-900">{email ?? t.notSet}</p>
          {email ? (
            <span
              className={`mt-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                emailVerified ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700'
              }`}
            >
              {emailVerified ? (
                <IconCircleCheck size={13} aria-hidden="true" />
              ) : (
                <IconCircleDashed size={13} aria-hidden="true" />
              )}
              {emailVerified ? t.verified : t.unverified}
            </span>
          ) : null}
        </div>
        {!editing ? (
          <Button type="button" variant="outline" onClick={onStartEdit}>
            {t.change}
          </Button>
        ) : null}
      </Card>

      {sentTo ? (
        <Alert variant="success">
          <span className="font-mono">{sentTo}</span> {t.sent.replace('{email}', sentTo)}
        </Alert>
      ) : null}
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      {editing ? (
        <Card className="p-5">
          <form onSubmit={onSubmit} className="grid gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="new-email">{t.newEmail}</Label>
              <Input
                id="new-email"
                type="email"
                value={newEmail}
                required
                autoComplete="email"
                placeholder="you@example.com"
                onChange={(event) => onNewEmailChange(event.target.value)}
              />
              <p className="text-xs text-slate-500">{t.help}</p>
            </div>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={submitting || newEmail.trim().length === 0}>
                {submitting ? t.sending : t.send}
              </Button>
              <Button type="button" variant="ghost" disabled={submitting} onClick={onCancelEdit}>
                {t.cancel}
              </Button>
            </div>
          </form>
        </Card>
      ) : null}
      {dialog}
    </AccountShell>
  )
}

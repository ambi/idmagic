import { IconAlertCircle, IconArrowRight, IconAt, IconCircleCheck } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, requestPasswordReset } from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { passwordRecoveryDictionary } from './PasswordRecoveryPages.i18n'

export function ForgotPasswordPage({ csrfToken }: { csrfToken: string }) {
  const t = useDictionary(passwordRecoveryDictionary)
  const [submitting, setSubmitting] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const email = String(new FormData(event.currentTarget).get('email') ?? '')
    setSubmitting(true)
    setError('')
    try {
      await requestPasswordReset(csrfToken, email)
      setSubmitted(true)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.networkError)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthShell>
      <div className="flex flex-col gap-7">
        <header className="flex flex-col gap-2.5">
          <p className="eyebrow">{t.forgotEyebrow}</p>
          <h2 className="page-title">{t.forgotTitle}</h2>
          <p className="page-description">{t.forgotDescription}</p>
        </header>
        {submitted ? (
          <Alert className="flex gap-3 border-emerald-200 bg-emerald-50" aria-live="polite">
            <IconCircleCheck className="mt-0.5 text-emerald-600" size={19} aria-hidden="true" />
            <p className="text-sm text-emerald-900">{t.resetSent}</p>
          </Alert>
        ) : null}
        {error ? (
          <Alert className="flex gap-3" aria-live="polite">
            <IconAlertCircle className="mt-0.5 text-red-600" size={19} aria-hidden="true" />
            <p className="text-sm text-red-800">{error}</p>
          </Alert>
        ) : null}
        <ForgotPasswordFormPresentation
          submitting={submitting}
          submitted={submitted}
          onSubmit={handleSubmit}
        />
        <a className="text-center text-sm font-medium text-blue-700 hover:underline" href="/login">
          {t.returnToSignIn}
        </a>
      </div>
    </AuthShell>
  )
}

export function ForgotPasswordFormPresentation({
  submitting,
  submitted,
  onSubmit,
}: {
  submitting: boolean
  submitted: boolean
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  const t = useDictionary(passwordRecoveryDictionary)
  return (
    <form onSubmit={onSubmit}>
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-2">
          <Label htmlFor="email">{t.emailAddress}</Label>
          <div className="relative">
            <IconAt
              className="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400"
              size={18}
            />
            <Input
              id="email"
              name="email"
              type="email"
              className="pl-10"
              autoComplete="email"
              required
              autoFocus
              disabled={submitting || submitted}
            />
          </div>
        </div>
        <Button type="submit" size="lg" className="w-full" disabled={submitting || submitted}>
          {submitting ? t.sending : t.sendResetLink}
          <IconArrowRight size={18} aria-hidden="true" />
        </Button>
      </div>
    </form>
  )
}

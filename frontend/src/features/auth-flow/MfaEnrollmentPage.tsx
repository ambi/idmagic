import { IconAlertCircle, IconArrowRight, IconShieldLock } from '@tabler/icons-react'
import { type FormEvent, useEffect, useState } from 'react'
import {
  AuthenticationAPIError,
  confirmMfaEnrollment,
  continueBrowserFlow,
  startMfaEnrollment,
  type MfaEnrollmentStart,
} from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { mfaEnrollmentPageDictionary } from './MfaEnrollmentPage.i18n'

export function MfaEnrollmentPage({
  csrfToken,
  returnTo,
}: {
  csrfToken: string
  returnTo?: string
}) {
  const t = useDictionary(mfaEnrollmentPageDictionary)
  const [enrollment, setEnrollment] = useState<MfaEnrollmentStart>()
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    void startMfaEnrollment(csrfToken)
      .then(setEnrollment)
      .catch((cause: unknown) =>
        setError(cause instanceof AuthenticationAPIError ? cause.message : t.startFailed),
      )
  }, [csrfToken, t.startFailed])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!enrollment) return
    setSubmitting(true)
    setError('')
    try {
      continueBrowserFlow(
        await confirmMfaEnrollment(csrfToken, enrollment.secret, code.trim(), returnTo),
      )
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.completeFailed)
      setSubmitting(false)
    }
  }

  return (
    <AuthShell>
      <div className="flex flex-col gap-6">
        <header className="flex flex-col gap-2.5">
          <p className="eyebrow">{t.eyebrow}</p>
          <h2 className="page-title">{t.title}</h2>
          <p className="page-description">{t.description}</p>
        </header>
        {error ? (
          <Alert className="flex gap-3" aria-live="polite">
            <IconAlertCircle size={19} aria-hidden="true" />
            <p>{error}</p>
          </Alert>
        ) : null}
        {enrollment ? (
          <form onSubmit={handleSubmit} className="grid gap-5">
            <div className="rounded-xl bg-slate-50 p-4 text-sm">
              <p>{t.setupInstruction}</p>
              <code className="mt-2 block break-all font-mono font-semibold">
                {enrollment.secret}
              </code>
              <p className="mt-2 text-xs text-slate-500">
                {t.account.replace('{account}', enrollment.account_name)}
              </p>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="enrollment-code">{t.verificationCode}</Label>
              <Input
                id="enrollment-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                pattern="[0-9]{6}"
                maxLength={6}
                required
                value={code}
                onChange={(event) => setCode(event.target.value.replace(/\D/g, ''))}
              />
            </div>
            <Button type="submit" size="lg" disabled={submitting}>
              {submitting ? t.enrolling : t.submit}
              <IconArrowRight size={18} aria-hidden="true" />
            </Button>
          </form>
        ) : null}
        <div className="flex gap-3 rounded-xl bg-slate-50 p-3.5 text-xs text-slate-600">
          <IconShieldLock size={17} aria-hidden="true" />
          <p>{t.securityNote}</p>
        </div>
      </div>
    </AuthShell>
  )
}

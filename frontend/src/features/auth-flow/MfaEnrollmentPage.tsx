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

export function MfaEnrollmentPage({
  csrfToken,
  returnTo,
}: {
  csrfToken: string
  returnTo?: string
}) {
  const [enrollment, setEnrollment] = useState<MfaEnrollmentStart>()
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    void startMfaEnrollment(csrfToken)
      .then(setEnrollment)
      .catch((cause: unknown) =>
        setError(
          cause instanceof AuthenticationAPIError ? cause.message : '登録を開始できませんでした。',
        ),
      )
  }, [csrfToken])

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
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : '登録を完了できませんでした。',
      )
      setSubmitting(false)
    }
  }

  return (
    <AuthShell>
      <div className="flex flex-col gap-6">
        <header className="flex flex-col gap-2.5">
          <p className="eyebrow">MFA 登録が必要です</p>
          <h2 className="page-title">認証アプリを登録</h2>
          <p className="page-description">
            管理者が承認した登録手続きです。完了するまでアプリやマイページにはアクセスできません。
          </p>
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
              <p>認証アプリに次のキーを登録してください。</p>
              <code className="mt-2 block break-all font-mono font-semibold">
                {enrollment.secret}
              </code>
              <p className="mt-2 text-xs text-slate-500">アカウント: {enrollment.account_name}</p>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="enrollment-code">確認コード</Label>
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
              {submitting ? '登録中…' : '登録してログインを続行'}
              <IconArrowRight size={18} aria-hidden="true" />
            </Button>
          </form>
        ) : null}
        <div className="flex gap-3 rounded-xl bg-slate-50 p-3.5 text-xs text-slate-600">
          <IconShieldLock size={17} aria-hidden="true" />
          <p>この登録は一度だけ利用できる管理者承認に基づいています。</p>
        </div>
      </div>
    </AuthShell>
  )
}

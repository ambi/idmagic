import { IconCircleCheck, IconMail } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, confirmEmailChange, tenantURL } from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { useDictionary } from '../../lib/i18n'
import { emailVerifyPageDictionary } from './EmailVerifyPage.i18n'

export function EmailVerifyPage({ csrfToken, token }: { csrfToken: string; token: string }) {
  const t = useDictionary(emailVerifyPageDictionary)
  const [state, setState] = useState<'idle' | 'submitting' | 'done'>('idle')
  const [error, setError] = useState('')

  async function handleConfirm() {
    setState('submitting')
    setError('')
    try {
      await confirmEmailChange(csrfToken, token)
      setState('done')
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.confirmationFailed)
      setState('idle')
    }
  }

  return (
    <AuthShell aside={false}>
      <div className="grid gap-6">
        <header className="grid gap-2">
          <span className="flex size-11 items-center justify-center rounded-xl bg-blue-50 text-blue-700">
            <IconMail size={22} aria-hidden="true" />
          </span>
          <h1 className="text-xl font-semibold text-slate-900">{t.title}</h1>
          <p className="text-sm text-slate-600">{t.description}</p>
        </header>

        {error ? <Alert variant="destructive">{error}</Alert> : null}

        <EmailVerificationAction token={token} state={state} onConfirm={handleConfirm} />
      </div>
    </AuthShell>
  )
}

export function EmailVerificationAction({
  token,
  state,
  onConfirm,
}: {
  token: string
  state: 'idle' | 'submitting' | 'done'
  onConfirm: () => void
}) {
  const t = useDictionary(emailVerifyPageDictionary)
  if (state === 'done') {
    return (
      <Alert variant="success" className="flex items-start gap-2">
        <IconCircleCheck className="mt-0.5 shrink-0" size={18} aria-hidden="true" />
        <span>
          {t.confirmed}{' '}
          <a href={tenantURL('/account')} className="font-medium underline">
            {t.returnToAccount}
          </a>
        </span>
      </Alert>
    )
  }
  if (!token) {
    return <Alert variant="destructive">{t.invalidLink}</Alert>
  }
  return (
    <Button type="button" onClick={onConfirm} disabled={state === 'submitting'}>
      {state === 'submitting' ? t.verifying : t.confirmEmail}
    </Button>
  )
}

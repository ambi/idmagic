import {
  IconAlertCircle,
  IconArrowRight,
  IconFingerprint,
  IconKey,
  IconLifebuoy,
  IconShieldCheck,
} from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  continueBrowserFlow,
  loginWithPasskey,
  submitRecoveryCode,
  submitTOTP,
} from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'

type SecondFactorMethod = 'totp' | 'webauthn' | 'recovery_code'

const methodOrder: SecondFactorMethod[] = ['totp', 'webauthn', 'recovery_code']

const methodLabels: Record<SecondFactorMethod, string> = {
  totp: '認証アプリ',
  webauthn: 'パスキー',
  recovery_code: 'リカバリコード',
}

export function availableSecondFactorMethods(methods?: string[]): SecondFactorMethod[] {
  const available = methodOrder.filter((method) => (methods ?? ['totp']).includes(method))
  return available.length > 0 ? available : ['totp']
}

// TotpPage は password 認証後の第二要素ステップ。利用できる method (認証アプリ / パスキー /
// リカバリコード) を選択して検証する (wi-26 / ADR-087)。
export function TotpPage({
  csrfToken,
  returnTo,
  secondFactorMethods,
}: {
  csrfToken: string
  returnTo?: string
  secondFactorMethods?: string[]
}) {
  const methods = availableSecondFactorMethods(secondFactorMethods)
  const [method, setMethod] = useState<SecondFactorMethod>(methods[0])
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  function fail(cause: unknown, fallback: string) {
    setError(cause instanceof AuthenticationAPIError ? cause.message : fallback)
    setSubmitting(false)
  }

  async function handleTotp(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError('')
    try {
      continueBrowserFlow(await submitTOTP(csrfToken, code.trim(), returnTo))
    } catch (cause) {
      fail(cause, '認証サービスに接続できませんでした。')
    }
  }

  async function handleRecovery(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError('')
    try {
      continueBrowserFlow(await submitRecoveryCode(csrfToken, code.trim(), returnTo))
    } catch (cause) {
      fail(cause, '認証サービスに接続できませんでした。')
    }
  }

  async function handlePasskey() {
    setSubmitting(true)
    setError('')
    try {
      continueBrowserFlow(await loginWithPasskey(csrfToken, returnTo))
    } catch (cause) {
      if (cause instanceof DOMException) {
        fail(cause, 'パスキー認証がキャンセルされました。')
        return
      }
      fail(cause, 'パスキー認証に失敗しました。')
    }
  }

  function selectMethod(next: SecondFactorMethod) {
    setMethod(next)
    setCode('')
    setError('')
  }

  return (
    <AuthShell>
      <div className="flex flex-col gap-7">
        <header className="flex flex-col gap-2.5">
          <p className="eyebrow">二要素認証</p>
          <h2 className="page-title">本人確認</h2>
          <p className="page-description">
            サインインを完了するために、二段階目の本人確認を行ってください。
          </p>
        </header>

        {error ? (
          <Alert className="flex gap-3" aria-live="polite">
            <IconAlertCircle
              className="mt-0.5 shrink-0 text-red-600"
              size={19}
              aria-hidden="true"
            />
            <div>
              <p className="font-semibold">確認できません</p>
              <p className="mt-1 text-sm leading-5 text-red-800">{error}</p>
            </div>
          </Alert>
        ) : null}

        {methods.length > 1 ? (
          <div className="flex flex-wrap gap-2" role="tablist" aria-label="本人確認の方法">
            {methods.map((option) => (
              <Button
                key={option}
                type="button"
                variant={option === method ? 'default' : 'outline'}
                className="h-9 px-3 text-xs"
                aria-pressed={option === method}
                disabled={submitting}
                onClick={() => selectMethod(option)}
              >
                {methodLabels[option]}
              </Button>
            ))}
          </div>
        ) : null}

        {method === 'totp' ? (
          <form onSubmit={handleTotp}>
            <div className="flex flex-col gap-5">
              <div className="flex flex-col gap-2">
                <Label htmlFor="code">確認コード</Label>
                <div className="relative">
                  <IconKey
                    aria-hidden="true"
                    className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400"
                    size={18}
                  />
                  <Input
                    id="code"
                    name="code"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    pattern="[0-9]{6}"
                    maxLength={6}
                    placeholder="000000"
                    className="pl-10"
                    required
                    autoFocus
                    disabled={submitting}
                    value={code}
                    onChange={(event) => setCode(event.target.value.replace(/\D/g, ''))}
                  />
                </div>
              </div>
              <Button type="submit" size="lg" className="mt-1 w-full" disabled={submitting}>
                {submitting ? '確認しています…' : 'コードを確認'}
                <IconArrowRight size={18} aria-hidden="true" />
              </Button>
            </div>
          </form>
        ) : null}

        {method === 'webauthn' ? (
          <div className="flex flex-col gap-4">
            <p className="text-sm text-slate-600">
              登録済みのパスキー (指紋・顔認証・セキュリティキー) で本人確認します。
            </p>
            <Button
              type="button"
              size="lg"
              className="w-full"
              disabled={submitting}
              onClick={handlePasskey}
            >
              <IconFingerprint size={18} aria-hidden="true" />
              {submitting ? '認証しています…' : 'パスキーで認証'}
            </Button>
          </div>
        ) : null}

        {method === 'recovery_code' ? (
          <form onSubmit={handleRecovery}>
            <div className="flex flex-col gap-5">
              <div className="flex flex-col gap-2">
                <Label htmlFor="recovery">リカバリコード</Label>
                <div className="relative">
                  <IconLifebuoy
                    aria-hidden="true"
                    className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400"
                    size={18}
                  />
                  <Input
                    id="recovery"
                    name="recovery"
                    autoComplete="one-time-code"
                    placeholder="xxxxxxxxxx"
                    className="pl-10 font-mono tracking-wider"
                    required
                    autoFocus
                    disabled={submitting}
                    value={code}
                    onChange={(event) => setCode(event.target.value)}
                  />
                </div>
                <p className="text-xs text-slate-500">
                  認証アプリやパスキーが使えない場合に、保存済みのリカバリコードを 1 つ入力します。
                </p>
              </div>
              <Button type="submit" size="lg" className="mt-1 w-full" disabled={submitting}>
                {submitting ? '確認しています…' : 'リカバリコードを確認'}
                <IconArrowRight size={18} aria-hidden="true" />
              </Button>
            </div>
          </form>
        ) : null}

        <div className="flex items-start gap-3 rounded-xl bg-slate-50 p-3.5 text-xs leading-5 text-slate-600">
          <IconShieldCheck
            className="mt-0.5 shrink-0 text-slate-500"
            size={17}
            aria-hidden="true"
          />
          <p>パスキーは端末に紐づき、フィッシングに強い認証方法です。</p>
        </div>
      </div>
    </AuthShell>
  )
}

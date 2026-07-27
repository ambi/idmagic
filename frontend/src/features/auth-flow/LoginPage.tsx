import {
  IconAlertCircle,
  IconArrowRight,
  IconAt,
  IconEye,
  IconEyeOff,
  IconLock,
  IconShieldLock,
} from '@tabler/icons-react'
import { type FormEvent, useEffect, useState } from 'react'
import { tenantURL } from '../../api/core'
import {
  AuthenticationAPIError,
  continueBrowserFlow,
  listFederatedLoginProviders,
  login,
  type FederatedLoginProvider,
} from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { localizedErrorMessage, useDictionary, useLocale } from '../../lib/i18n'
import { loginPageDictionary } from './LoginPage.i18n'

export function LoginPage({ csrfToken, returnTo }: { csrfToken: string; returnTo?: string }) {
  const t = useDictionary(loginPageDictionary)
  const tCommon = useDictionary(commonDictionary)
  const { locale } = useLocale()
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [providers, setProviders] = useState<FederatedLoginProvider[]>([])

  useEffect(() => {
    let active = true
    listFederatedLoginProviders()
      .then((items) => {
        if (active) setProviders(items)
      })
      .catch(() => {
        // Password authentication remains available if provider discovery is unavailable.
      })
    return () => {
      active = false
    }
  }, [])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    setSubmitting(true)
    setError('')
    try {
      const result = await login(
        csrfToken,
        String(form.get('username') ?? ''),
        String(form.get('password') ?? ''),
        returnTo,
      )
      continueBrowserFlow(result)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? localizedErrorMessage(locale, cause.code, cause.message)
          : tCommon.networkError,
      )
      setSubmitting(false)
    }
  }

  return (
    <AuthShell>
      <div className="flex flex-col gap-7">
        <header className="flex flex-col gap-2.5">
          <p className="eyebrow">{t.eyebrow}</p>
          <h2 className="page-title">{t.title}</h2>
          <p className="page-description">{t.description}</p>
        </header>

        {error ? (
          <Alert className="flex gap-3" aria-live="polite">
            <IconAlertCircle
              className="mt-0.5 shrink-0 text-red-600"
              size={19}
              aria-hidden="true"
            />
            <div>
              <p className="font-semibold">{t.errorTitle}</p>
              <p className="mt-1 text-sm leading-5 text-red-800">{error}</p>
            </div>
          </Alert>
        ) : null}

        <LoginFormPresentation
          submitting={submitting}
          showPassword={showPassword}
          onSubmit={handleSubmit}
          onTogglePassword={() => setShowPassword((visible) => !visible)}
        />

        {providers.length > 0 ? (
          <section className="flex flex-col gap-3" aria-label={t.alternativeSignIn}>
            <div className="flex items-center gap-3 text-xs text-slate-500">
              <span className="h-px flex-1 bg-slate-200" />
              <span>{t.alternativeSignIn}</span>
              <span className="h-px flex-1 bg-slate-200" />
            </div>
            {providers.map((provider) => {
              const parameters = new URLSearchParams({ provider_id: provider.id })
              if (returnTo) parameters.set('return_to', returnTo)
              return (
                <a
                  key={provider.id}
                  className="flex h-11 items-center justify-center rounded-md border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-800 transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/30"
                  href={`${tenantURL('/api/auth/federation/start')}?${parameters.toString()}`}
                >
                  {t.continueWith.replace('{provider}', provider.display_name)}
                </a>
              )
            })}
          </section>
        ) : null}

        <div className="flex items-start gap-3 rounded-xl bg-slate-50 p-3.5 text-xs leading-5 text-slate-600">
          <IconShieldLock className="mt-0.5 shrink-0 text-slate-500" size={17} aria-hidden="true" />
          <p>{t.securityNote}</p>
        </div>
      </div>
    </AuthShell>
  )
}

export function LoginFormPresentation({
  submitting,
  showPassword,
  onSubmit,
  onTogglePassword,
}: {
  submitting: boolean
  showPassword: boolean
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  onTogglePassword: () => void
}) {
  const t = useDictionary(loginPageDictionary)
  return (
    <form onSubmit={onSubmit}>
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-2">
          <Label htmlFor="username">{t.usernameLabel}</Label>
          <div className="relative">
            <IconAt
              aria-hidden="true"
              className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400"
              size={18}
            />
            <Input
              id="username"
              name="username"
              placeholder={t.usernamePlaceholder}
              className="pl-10"
              autoComplete="username"
              spellCheck={false}
              required
              autoFocus
              disabled={submitting}
            />
          </div>
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="password">{t.passwordLabel}</Label>
          <div className="relative">
            <IconLock
              aria-hidden="true"
              className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400"
              size={18}
            />
            <Input
              id="password"
              type={showPassword ? 'text' : 'password'}
              name="password"
              placeholder={t.passwordPlaceholder}
              className="px-10"
              autoComplete="current-password"
              required
              disabled={submitting}
            />
            <button
              type="button"
              onClick={onTogglePassword}
              className="absolute right-2.5 top-1/2 flex size-8 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/30"
              aria-label={showPassword ? t.hidePassword : t.showPassword}
              aria-pressed={showPassword}
            >
              {showPassword ? (
                <IconEyeOff size={18} aria-hidden="true" />
              ) : (
                <IconEye size={18} aria-hidden="true" />
              )}
            </button>
          </div>
        </div>
        <Button
          type="submit"
          size="lg"
          className="tenant-primary-cta mt-1 w-full"
          disabled={submitting}
        >
          {submitting ? t.submitting : t.submit}
          <IconArrowRight size={18} aria-hidden="true" />
        </Button>
        <div className="flex justify-center">
          <a
            className="text-xs font-medium text-blue-700 hover:underline"
            href={tenantURL('/forgot_password')}
          >
            {t.forgotPassword}
          </a>
        </div>
      </div>
    </form>
  )
}

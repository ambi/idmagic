import {
  IconDeviceDesktopCheck,
  IconInfoCircle,
  IconKeyboard,
  IconShieldCheck,
  IconX,
} from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, continueBrowserFlow, submitDevice } from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { devicePageDictionary } from './DevicePage.i18n'

export function DevicePage({ csrfToken, userCode }: { csrfToken: string; userCode: string }) {
  const t = useDictionary(devicePageDictionary)
  const normalizedCode = userCode.replace(/-/g, '').toUpperCase()
  const [code, setCode] = useState(normalizedCode)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleDevice(action: 'allow' | 'deny') {
    setSubmitting(true)
    setError('')
    try {
      continueBrowserFlow(await submitDevice(csrfToken, code, action))
    } catch (cause) {
      if (cause instanceof AuthenticationAPIError && cause.code === 'authentication_required') {
        window.location.assign('/status?state=authentication-required')
        return
      }
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.deviceError)
      setSubmitting(false)
    }
  }

  return (
    <AuthShell asideTitle={t.asideTitle} asideText={t.asideText}>
      <div className="flex flex-col gap-7">
        <header className="flex flex-col gap-2.5">
          <div className="mb-1 flex size-12 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
            <IconDeviceDesktopCheck size={25} aria-hidden="true" />
          </div>
          <p className="eyebrow">{t.eyebrow}</p>
          <h2 className="page-title">{t.title}</h2>
          <p className="page-description">{t.description}</p>
        </header>

        <DeviceCodeFormPresentation
          code={code}
          error={error}
          submitting={submitting}
          onCodeChange={setCode}
          onSubmit={handleDevice}
        />
      </div>
    </AuthShell>
  )
}

export function normalizeDeviceCode(value: string): string {
  return value
    .replace(/[^a-z0-9]/gi, '')
    .slice(0, 8)
    .toUpperCase()
}

export function DeviceCodeFormPresentation({
  code,
  error,
  submitting,
  onCodeChange,
  onSubmit,
}: {
  code: string
  error: string
  submitting: boolean
  onCodeChange: (code: string) => void
  onSubmit: (action: 'allow' | 'deny') => void
}) {
  const t = useDictionary(devicePageDictionary)
  const isComplete = code.length === 8
  return (
    <form onSubmit={(event) => event.preventDefault()}>
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <Label htmlFor="user-code">{t.codeLabel}</Label>
            <span className="text-xs tabular-nums text-slate-500">{code.length} / 8</span>
          </div>
          <div className="relative">
            <IconKeyboard
              className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-slate-400"
              size={19}
              aria-hidden="true"
            />
            <Input
              id="user-code"
              value={code}
              onChange={(event) => onCodeChange(normalizeDeviceCode(event.currentTarget.value))}
              inputMode="text"
              autoComplete="one-time-code"
              spellCheck={false}
              aria-describedby="user-code-hint"
              className="h-16 px-12 text-center font-mono text-xl font-bold tracking-[0.32em] uppercase sm:text-2xl"
              disabled={submitting}
            />
          </div>
          <p id="user-code-hint" className="text-xs leading-5 text-slate-500">
            {t.codeHintPrefix}
            <span className="font-mono font-semibold">ABCD-EFGH</span>
          </p>
        </div>
        <div className="flex gap-3 rounded-xl border border-blue-100 bg-blue-50/60 p-3.5 text-xs leading-5 text-blue-950">
          <IconInfoCircle className="mt-0.5 shrink-0 text-blue-700" size={17} aria-hidden="true" />
          <p>{t.warningNote}</p>
        </div>
        {error ? (
          <p role="alert" className="text-sm font-medium text-red-700">
            {error}
          </p>
        ) : null}
        <div className="flex flex-col gap-2.5">
          <Button
            type="button"
            size="lg"
            className="tenant-primary-cta"
            disabled={!isComplete || submitting}
            onClick={() => onSubmit('allow')}
          >
            <IconShieldCheck size={18} aria-hidden="true" />
            {submitting ? t.processing : t.approve}
          </Button>
          <Button
            type="button"
            size="lg"
            variant="ghost"
            disabled={!isComplete || submitting}
            onClick={() => onSubmit('deny')}
          >
            <IconX size={17} aria-hidden="true" />
            {t.deny}
          </Button>
        </div>
      </div>
    </form>
  )
}

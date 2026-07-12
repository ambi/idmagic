import {
  IconArrowRight,
  IconCircleCheck,
  IconDeviceMobile,
  IconFingerprint,
  IconKey,
  IconLifebuoy,
  IconShieldLock,
  IconTrash,
} from '@tabler/icons-react'
import { QRCodeSVG } from 'qrcode.react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  confirmTotpEnrollment,
  generateRecoveryCodes,
  isWebAuthnSupported,
  registerPasskey,
  removePasskey,
  removeTotpFactor,
  revokeRecoveryCodes,
  startTotpEnrollment,
  tenantURL,
} from '../../api'
import { AccountShell } from '../../components/AccountShell'
import { StepUpCancelledError, useStepUpGuard } from '../../components/StepUpDialog'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import type {
  AccountSecurity,
  RecoveryCodeStatus,
  TotpEnrollmentStart,
  WebAuthnCredentialSummary,
} from '../../types'
import { useDictionary } from '../../lib/i18n'
import { accountSecurityDictionary } from './AccountSecurityPage.i18n'

export function formatAccountSecurityDateTime(value?: string): string {
  if (!value) return '記録なし'
  return new Date(value).toLocaleString('ja-JP', { dateStyle: 'medium', timeStyle: 'short' })
}

function errorMessage(cause: unknown, fallback: string): string {
  return cause instanceof AuthenticationAPIError ? cause.message : fallback
}

export function AccountSecurityPage({
  csrfToken,
  username,
  isAdmin,
  security,
}: {
  csrfToken: string
  username: string
  isAdmin: boolean
  security: AccountSecurity
}) {
  const t = useDictionary(accountSecurityDictionary)
  const [enrolled, setEnrolled] = useState(security.totp_enrolled)
  const [enrollment, setEnrollment] = useState<TotpEnrollmentStart | null>(null)
  const [enrollCode, setEnrollCode] = useState('')
  const [removeCode, setRemoveCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [passkeys, setPasskeys] = useState<WebAuthnCredentialSummary[]>(
    security.webauthn_credentials ?? [],
  )
  const [passkeyLabel, setPasskeyLabel] = useState('')
  const [recovery, setRecovery] = useState<RecoveryCodeStatus>(
    security.recovery_codes ?? { total: 0, remaining: 0 },
  )
  const [generatedCodes, setGeneratedCodes] = useState<string[] | null>(null)
  const { guard, dialog } = useStepUpGuard(csrfToken)

  async function handleRegisterPasskey() {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await registerPasskey(csrfToken, passkeyLabel)
      setPasskeyLabel('')
      // 登録直後は最新一覧を取得するためページを再読み込みする (loader が再取得する)。
      window.location.reload()
    } catch (cause) {
      if (cause instanceof DOMException) {
        setError(t.passkeyCancelled)
      } else {
        setError(errorMessage(cause, t.passkeyRegisterFailed))
      }
      setBusy(false)
    }
  }

  async function handleRemovePasskey(credentialId: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await guard(() => removePasskey(csrfToken, credentialId))
      setPasskeys((current) => current.filter((c) => c.credential_id !== credentialId))
      setNotice(t.passkeyRemoved)
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(errorMessage(cause, t.passkeyRemoveFailed))
    } finally {
      setBusy(false)
    }
  }

  async function handleGenerateRecovery() {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      const result = await guard(() => generateRecoveryCodes(csrfToken))
      setGeneratedCodes(result.codes)
      setRecovery({
        generated_at: result.generated_at,
        total: result.codes.length,
        remaining: result.codes.length,
      })
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(errorMessage(cause, t.recoveryGenerateFailed))
    } finally {
      setBusy(false)
    }
  }

  async function handleRevokeRecovery() {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await guard(() => revokeRecoveryCodes(csrfToken))
      setGeneratedCodes(null)
      setRecovery({ total: 0, remaining: 0 })
      setNotice(t.recoveryRevoked)
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(errorMessage(cause, t.recoveryRevokeFailed))
    } finally {
      setBusy(false)
    }
  }

  async function handleStart() {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      setEnrollment(await startTotpEnrollment(csrfToken))
      setEnrollCode('')
    } catch (cause) {
      setError(errorMessage(cause, t.totpStartFailed))
    } finally {
      setBusy(false)
    }
  }

  async function handleConfirm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!enrollment) return
    setBusy(true)
    setError('')
    try {
      await confirmTotpEnrollment(csrfToken, enrollment.secret, enrollCode.trim())
      setEnrolled(true)
      setEnrollment(null)
      setEnrollCode('')
      setNotice(t.totpEnrolled)
    } catch (cause) {
      setError(errorMessage(cause, t.totpEnrollFailed))
    } finally {
      setBusy(false)
    }
  }

  async function handleRemove(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      await guard(() => removeTotpFactor(csrfToken, removeCode.trim()))
      setEnrolled(false)
      setRemoveCode('')
      setNotice(t.totpRemoved)
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(errorMessage(cause, t.totpRemoveFailed))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AccountShell
      active="security"
      username={username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      <Toast message={notice} onDismiss={() => setNotice('')} />
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      <PasswordCard passwordChangedAt={security.password_changed_at} />

      <TotpCard
        enrolled={enrolled}
        enrollment={enrollment}
        enrollCode={enrollCode}
        removeCode={removeCode}
        busy={busy}
        onStart={handleStart}
        onConfirm={handleConfirm}
        onCancel={() => {
          setEnrollment(null)
          setEnrollCode('')
          setError('')
        }}
        onEnrollCodeChange={setEnrollCode}
        onRemoveCodeChange={setRemoveCode}
        onRemove={handleRemove}
      />

      <PasskeysCard
        passkeys={passkeys}
        passkeyLabel={passkeyLabel}
        busy={busy}
        onPasskeyLabelChange={setPasskeyLabel}
        onRegister={handleRegisterPasskey}
        onRemove={handleRemovePasskey}
      />

      <RecoveryCodesCard
        recovery={recovery}
        generatedCodes={generatedCodes}
        busy={busy}
        onGenerate={handleGenerateRecovery}
        onRevoke={handleRevokeRecovery}
      />

      <div className="flex items-start gap-3 rounded-xl bg-slate-50 p-3.5 text-xs leading-5 text-slate-600">
        <IconShieldLock className="mt-0.5 shrink-0 text-slate-500" size={17} aria-hidden="true" />
        <p>{t.hint}</p>
      </div>
      {dialog}
    </AccountShell>
  )
}

function PasswordCard({ passwordChangedAt }: { passwordChangedAt?: string }) {
  const t = useDictionary(accountSecurityDictionary)
  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconKey size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900">{t.password}</p>
          <p className="mt-1 text-sm text-slate-600">
            {t.changedAt.replace('{date}', formatAccountSecurityDateTime(passwordChangedAt))}
          </p>
        </div>
      </div>
      <div>
        <Button asChild variant="outline">
          <a href={tenantURL('/account/password')}>
            {t.changePassword}
            <IconArrowRight size={16} aria-hidden="true" />
          </a>
        </Button>
      </div>
    </Card>
  )
}

export function TotpEnrollmentForm({
  enrollment,
  enrollCode,
  busy,
  onConfirm,
  onCancel,
  onEnrollCodeChange,
}: {
  enrollment: TotpEnrollmentStart
  enrollCode: string
  busy: boolean
  onConfirm: (event: FormEvent<HTMLFormElement>) => void
  onCancel: () => void
  onEnrollCodeChange: (value: string) => void
}) {
  const t = useDictionary(accountSecurityDictionary)
  return (
    <form onSubmit={onConfirm} className="grid gap-4 border-t border-slate-100 pt-4">
      <div className="flex flex-col items-center gap-3 border-b border-slate-100 pb-4">
        <p className="text-center text-sm text-slate-700">{t.scanQR}</p>
        <div className="rounded-xl border border-slate-200 bg-white p-3">
          <QRCodeSVG
            value={enrollment.otpauth_uri}
            size={176}
            level="M"
            marginSize={0}
            title={t.qrTitle}
          />
        </div>
      </div>
      <details className="rounded-lg bg-slate-50 px-3.5 py-3 text-sm text-slate-600">
        <summary className="cursor-pointer font-medium text-slate-700">{t.cannotScan}</summary>
        <div className="mt-3 grid gap-1.5">
          <Label htmlFor="totp-secret">{t.setupKey}</Label>
          <Input
            id="totp-secret"
            readOnly
            value={enrollment.secret}
            className="font-mono tracking-wider"
            onFocus={(event) => event.target.select()}
          />
          <p className="mt-1 text-xs text-slate-500">{t.setupHelp}</p>
          <p className="mt-2 break-all text-xs text-slate-400">{enrollment.otpauth_uri}</p>
        </div>
      </details>
      <div className="grid gap-1.5">
        <Label htmlFor="totp-code">{t.totpCode}</Label>
        <Input
          id="totp-code"
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="[0-9]{6}"
          maxLength={6}
          required
          placeholder="123456"
          value={enrollCode}
          className="font-mono tracking-[0.3em]"
          onChange={(event) => onEnrollCodeChange(event.target.value.replace(/\D/g, ''))}
        />
      </div>
      <div className="flex gap-2">
        <Button type="submit" disabled={busy || enrollCode.trim().length !== 6}>
          {busy ? t.enrolling : t.completeEnrollment}
        </Button>
        <Button type="button" variant="ghost" disabled={busy} onClick={onCancel}>
          {t.cancel}
        </Button>
      </div>
    </form>
  )
}

export function TotpRemovalForm({
  removeCode,
  busy,
  onSubmit,
  onRemoveCodeChange,
}: {
  removeCode: string
  busy: boolean
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  onRemoveCodeChange: (value: string) => void
}) {
  const t = useDictionary(accountSecurityDictionary)
  return (
    <form onSubmit={onSubmit} className="grid gap-4 border-t border-slate-100 pt-4">
      <div className="grid gap-1.5">
        <Label htmlFor="remove-code">{t.removalCode}</Label>
        <Input
          id="remove-code"
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="[0-9]{6}"
          maxLength={6}
          required
          placeholder="123456"
          value={removeCode}
          className="font-mono tracking-[0.3em]"
          onChange={(event) => onRemoveCodeChange(event.target.value.replace(/\D/g, ''))}
        />
        <p className="text-xs text-slate-500">{t.removalHelp}</p>
      </div>
      <div>
        <Button
          type="submit"
          variant="destructive"
          disabled={busy || removeCode.trim().length !== 6}
        >
          {busy ? t.removing : t.removeTotp}
        </Button>
      </div>
    </form>
  )
}

function TotpCard({
  enrolled,
  enrollment,
  enrollCode,
  removeCode,
  busy,
  onStart,
  onConfirm,
  onCancel,
  onEnrollCodeChange,
  onRemoveCodeChange,
  onRemove,
}: {
  enrolled: boolean
  enrollment: TotpEnrollmentStart | null
  enrollCode: string
  removeCode: string
  busy: boolean
  onStart: () => void
  onConfirm: (event: FormEvent<HTMLFormElement>) => void
  onCancel: () => void
  onEnrollCodeChange: (value: string) => void
  onRemoveCodeChange: (value: string) => void
  onRemove: (event: FormEvent<HTMLFormElement>) => void
}) {
  const t = useDictionary(accountSecurityDictionary)
  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconDeviceMobile size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900">{t.totp}</p>
          <p className="mt-1 text-sm text-slate-600">{t.totpDescription}</p>
          <span
            className={`mt-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
              enrolled ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-600'
            }`}
          >
            {enrolled ? <IconCircleCheck size={13} aria-hidden="true" /> : null}
            {enrolled ? t.configured : t.notConfigured}
          </span>
        </div>
      </div>

      {!enrolled && !enrollment ? (
        <div>
          <Button type="button" onClick={onStart} disabled={busy}>
            {busy ? t.preparing : t.setUpTotp}
          </Button>
        </div>
      ) : null}

      {!enrolled && enrollment ? (
        <TotpEnrollmentForm
          enrollment={enrollment}
          enrollCode={enrollCode}
          busy={busy}
          onConfirm={onConfirm}
          onCancel={onCancel}
          onEnrollCodeChange={onEnrollCodeChange}
        />
      ) : null}

      {enrolled ? (
        <TotpRemovalForm
          removeCode={removeCode}
          busy={busy}
          onSubmit={onRemove}
          onRemoveCodeChange={onRemoveCodeChange}
        />
      ) : null}
    </Card>
  )
}

export function PasskeyList({
  passkeys,
  busy,
  onRemove,
}: {
  passkeys: WebAuthnCredentialSummary[]
  busy: boolean
  onRemove: (credentialId: string) => void
}) {
  const t = useDictionary(accountSecurityDictionary)
  if (passkeys.length === 0) {
    return <p className="border-t border-slate-100 pt-4 text-sm text-slate-500">{t.noPasskeys}</p>
  }
  return (
    <ul className="flex flex-col gap-2 border-t border-slate-100 pt-4">
      {passkeys.map((passkey) => (
        <li
          key={passkey.credential_id}
          className="flex items-center justify-between gap-3 rounded-lg border border-slate-200 px-3.5 py-2.5"
        >
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-slate-800">
              {passkey.label ?? t.passkey}
            </p>
            <p className="mt-0.5 text-xs text-slate-500">
              {t.registered.replace('{date}', formatAccountSecurityDateTime(passkey.created_at))}
              {passkey.last_used_at
                ? t.lastUsed.replace('{date}', formatAccountSecurityDateTime(passkey.last_used_at))
                : ''}
            </p>
          </div>
          <Button
            type="button"
            variant="ghost"
            className="h-8 shrink-0 px-2 text-red-600 hover:bg-red-50"
            disabled={busy}
            onClick={() => onRemove(passkey.credential_id)}
          >
            <IconTrash size={16} aria-hidden="true" />
            {t.remove}
          </Button>
        </li>
      ))}
    </ul>
  )
}

export function PasskeyRegisterForm({
  passkeyLabel,
  busy,
  onLabelChange,
  onRegister,
}: {
  passkeyLabel: string
  busy: boolean
  onLabelChange: (value: string) => void
  onRegister: () => void
}) {
  const t = useDictionary(accountSecurityDictionary)
  if (!isWebAuthnSupported()) {
    return <p className="text-sm text-slate-500">{t.unsupported}</p>
  }
  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor="passkey-label">{t.passkeyName}</Label>
      <div className="flex gap-2">
        <Input
          id="passkey-label"
          placeholder={t.passkeyExample}
          maxLength={64}
          value={passkeyLabel}
          disabled={busy}
          onChange={(event) => onLabelChange(event.target.value)}
        />
        <Button type="button" className="shrink-0" onClick={onRegister} disabled={busy}>
          {busy ? t.register : t.registerPasskey}
        </Button>
      </div>
    </div>
  )
}

function PasskeysCard({
  passkeys,
  passkeyLabel,
  busy,
  onPasskeyLabelChange,
  onRegister,
  onRemove,
}: {
  passkeys: WebAuthnCredentialSummary[]
  passkeyLabel: string
  busy: boolean
  onPasskeyLabelChange: (value: string) => void
  onRegister: () => void
  onRemove: (credentialId: string) => void
}) {
  const t = useDictionary(accountSecurityDictionary)
  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconFingerprint size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900">{t.passkeys}</p>
          <p className="mt-1 text-sm text-slate-600">{t.passkeysDescription}</p>
        </div>
      </div>

      <PasskeyList passkeys={passkeys} busy={busy} onRemove={onRemove} />
      <PasskeyRegisterForm
        passkeyLabel={passkeyLabel}
        busy={busy}
        onLabelChange={onPasskeyLabelChange}
        onRegister={onRegister}
      />
    </Card>
  )
}

export function RecoveryCodesPanel({
  recovery,
  generatedCodes,
  busy,
  onGenerate,
  onRevoke,
}: {
  recovery: RecoveryCodeStatus
  generatedCodes: string[] | null
  busy: boolean
  onGenerate: () => void
  onRevoke: () => void
}) {
  const t = useDictionary(accountSecurityDictionary)
  return (
    <>
      {generatedCodes ? (
        <div className="flex flex-col gap-3 border-t border-slate-100 pt-4">
          <Alert>{t.codesWarning}</Alert>
          <ul className="grid grid-cols-2 gap-2 rounded-lg bg-slate-50 p-3 font-mono text-sm text-slate-800">
            {generatedCodes.map((code) => (
              <li key={code} className="tracking-wider">
                {code}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-4">
        <Button type="button" onClick={onGenerate} disabled={busy}>
          {busy ? t.processing : recovery.total > 0 ? t.regenerate : t.generate}
        </Button>
        {recovery.total > 0 ? (
          <Button type="button" variant="outline" onClick={onRevoke} disabled={busy}>
            {t.revokeAll}
          </Button>
        ) : null}
      </div>
      {recovery.total > 0 ? (
        <p className="text-xs text-slate-500">{t.regenerationWarning}</p>
      ) : null}
    </>
  )
}

function RecoveryCodesCard({
  recovery,
  generatedCodes,
  busy,
  onGenerate,
  onRevoke,
}: {
  recovery: RecoveryCodeStatus
  generatedCodes: string[] | null
  busy: boolean
  onGenerate: () => void
  onRevoke: () => void
}) {
  const t = useDictionary(accountSecurityDictionary)
  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconLifebuoy size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900">{t.recoveryCodes}</p>
          <p className="mt-1 text-sm text-slate-600">{t.recoveryDescription}</p>
          <span className="mt-2 inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
            {t.remaining
              .replace('{remaining}', String(recovery.remaining))
              .replace('{total}', String(recovery.total))}
          </span>
        </div>
      </div>

      <RecoveryCodesPanel
        recovery={recovery}
        generatedCodes={generatedCodes}
        busy={busy}
        onGenerate={onGenerate}
        onRevoke={onRevoke}
      />
    </Card>
  )
}

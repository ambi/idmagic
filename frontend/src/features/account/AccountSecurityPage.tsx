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
        setError('パスキーの登録がキャンセルされました。')
      } else {
        setError(errorMessage(cause, 'パスキーを登録できませんでした。'))
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
      setNotice('パスキーを解除しました。')
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(errorMessage(cause, 'パスキーを解除できませんでした。'))
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
      setError(errorMessage(cause, 'リカバリコードを生成できませんでした。'))
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
      setNotice('リカバリコードを失効しました。')
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(errorMessage(cause, 'リカバリコードを失効できませんでした。'))
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
      setError(errorMessage(cause, '認証アプリの登録を開始できませんでした。'))
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
      setNotice('認証アプリを登録しました。次回サインインから確認コードが必要になります。')
    } catch (cause) {
      setError(errorMessage(cause, '認証アプリを登録できませんでした。'))
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
      setNotice('認証アプリを解除しました。')
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(errorMessage(cause, '認証アプリを解除できませんでした。'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AccountShell
      active="security"
      username={username}
      isAdmin={isAdmin}
      title="セキュリティ"
      description="パスワードと二段階認証 (認証アプリ) を管理します。"
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
        <p>
          二段階認証を有効にすると、パスワードが漏れても認証アプリやパスキーがなければサインイン
          できません。
        </p>
      </div>
      {dialog}
    </AccountShell>
  )
}

function PasswordCard({ passwordChangedAt }: { passwordChangedAt?: string }) {
  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconKey size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900">パスワード</p>
          <p className="mt-1 text-sm text-slate-600">
            最終変更: {formatAccountSecurityDateTime(passwordChangedAt)}
          </p>
        </div>
      </div>
      <div>
        <Button asChild variant="outline">
          <a href={tenantURL('/account/password')}>
            パスワードを変更
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
  return (
    <form onSubmit={onConfirm} className="grid gap-4 border-t border-slate-100 pt-4">
      <div className="flex flex-col items-center gap-3 border-b border-slate-100 pb-4">
        <p className="text-center text-sm text-slate-700">
          認証アプリ (Google Authenticator など) で、この QR コードをスキャンしてください。
        </p>
        <div className="rounded-xl border border-slate-200 bg-white p-3">
          <QRCodeSVG
            value={enrollment.otpauth_uri}
            size={176}
            level="M"
            marginSize={0}
            title="認証アプリ登録用の QR コード"
          />
        </div>
      </div>
      <details className="rounded-lg bg-slate-50 px-3.5 py-3 text-sm text-slate-600">
        <summary className="cursor-pointer font-medium text-slate-700">
          QR コードをスキャンできない場合
        </summary>
        <div className="mt-3 grid gap-1.5">
          <Label htmlFor="totp-secret">セットアップキー</Label>
          <Input
            id="totp-secret"
            readOnly
            value={enrollment.secret}
            className="font-mono tracking-wider"
            onFocus={(event) => event.target.select()}
          />
          <p className="mt-1 text-xs text-slate-500">
            認証アプリに手動でこのキーを入力してください (時間ベース / 6 桁 / 30 秒)。
          </p>
          <p className="mt-2 break-all text-xs text-slate-400">{enrollment.otpauth_uri}</p>
        </div>
      </details>
      <div className="grid gap-1.5">
        <Label htmlFor="totp-code">認証アプリに表示された 6 桁コード</Label>
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
          {busy ? '登録中…' : '登録を完了'}
        </Button>
        <Button type="button" variant="ghost" disabled={busy} onClick={onCancel}>
          キャンセル
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
  return (
    <form onSubmit={onSubmit} className="grid gap-4 border-t border-slate-100 pt-4">
      <div className="grid gap-1.5">
        <Label htmlFor="remove-code">解除するには現在の 6 桁コードを入力</Label>
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
        <p className="text-xs text-slate-500">
          解除すると二段階認証が無効になります。共有端末では特に注意してください。
        </p>
      </div>
      <div>
        <Button
          type="submit"
          variant="destructive"
          disabled={busy || removeCode.trim().length !== 6}
        >
          {busy ? '解除中…' : '認証アプリを解除'}
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
  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconDeviceMobile size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900">認証アプリ (TOTP)</p>
          <p className="mt-1 text-sm text-slate-600">
            Google Authenticator などの認証アプリで生成する確認コードを、サインインの
            二段階目に使います。
          </p>
          <span
            className={`mt-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
              enrolled ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-600'
            }`}
          >
            {enrolled ? <IconCircleCheck size={13} aria-hidden="true" /> : null}
            {enrolled ? '設定済み' : '未設定'}
          </span>
        </div>
      </div>

      {!enrolled && !enrollment ? (
        <div>
          <Button type="button" onClick={onStart} disabled={busy}>
            {busy ? '準備中…' : '認証アプリを設定'}
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
  if (passkeys.length === 0) {
    return (
      <p className="border-t border-slate-100 pt-4 text-sm text-slate-500">
        登録済みのパスキーはありません。
      </p>
    )
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
              {passkey.label ?? 'パスキー'}
            </p>
            <p className="mt-0.5 text-xs text-slate-500">
              登録: {formatAccountSecurityDateTime(passkey.created_at)}
              {passkey.last_used_at
                ? ` / 最終利用: ${formatAccountSecurityDateTime(passkey.last_used_at)}`
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
            解除
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
  if (!isWebAuthnSupported()) {
    return <p className="text-sm text-slate-500">このブラウザはパスキーに対応していません。</p>
  }
  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor="passkey-label">パスキーの名前 (任意)</Label>
      <div className="flex gap-2">
        <Input
          id="passkey-label"
          placeholder="例: MacBook Touch ID"
          maxLength={64}
          value={passkeyLabel}
          disabled={busy}
          onChange={(event) => onLabelChange(event.target.value)}
        />
        <Button type="button" className="shrink-0" onClick={onRegister} disabled={busy}>
          {busy ? '登録中…' : 'パスキーを登録'}
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
  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconFingerprint size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900">パスキー (WebAuthn)</p>
          <p className="mt-1 text-sm text-slate-600">
            指紋・顔認証・セキュリティキーで、フィッシングに強い二段階認証を行います。複数の
            端末を登録できます。
          </p>
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
  return (
    <>
      {generatedCodes ? (
        <div className="flex flex-col gap-3 border-t border-slate-100 pt-4">
          <Alert>
            これらのコードは今だけ表示されます。安全な場所に保存してください。各コードは 1 回だけ
            使えます。
          </Alert>
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
          {busy
            ? '処理中…'
            : recovery.total > 0
              ? 'リカバリコードを再生成'
              : 'リカバリコードを生成'}
        </Button>
        {recovery.total > 0 ? (
          <Button type="button" variant="outline" onClick={onRevoke} disabled={busy}>
            すべて失効
          </Button>
        ) : null}
      </div>
      {recovery.total > 0 ? (
        <p className="text-xs text-slate-500">再生成すると既存のコードはすべて無効になります。</p>
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
  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <IconLifebuoy size={20} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900">リカバリコード</p>
          <p className="mt-1 text-sm text-slate-600">
            認証アプリやパスキーを使えないときに、二段階目の本人確認に使う一度きりのコードです。
          </p>
          <span className="mt-2 inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
            残り {recovery.remaining} / {recovery.total} 個
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

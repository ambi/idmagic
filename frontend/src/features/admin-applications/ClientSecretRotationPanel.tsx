import { IconAlertTriangle } from '@tabler/icons-react'
import { useState } from 'react'
import { rotateApplicationClientSecret } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import { CopyableField, messageOf } from './AdminApplicationsShared'

const GRACE_DAY_OPTIONS = [0, 1, 7, 14, 30] as const

// ClientSecretRotationPanel はシークレット再発行という破壊的操作を、OIDC 設定本体
// とは切り離した独立ブロックとして見せる。事故防止のため「猶予期間を選ぶ →
// ローテーション → 確認 → 実行」の明示的な 2 段階にし、1 クリックでは適用しない。
export function ClientSecretRotationPanel({
  applicationID,
  csrfToken,
  onError,
}: {
  applicationID: string
  csrfToken: string
  onError: (message: string) => void
}) {
  const [graceDays, setGraceDays] = useState<number>(7)
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [secret, setSecret] = useState('')
  const [copied, setCopied] = useState(false)
  const t = useDictionary(adminApplicationsDictionary)

  function graceOptionLabel(days: number): string {
    return days === 0
      ? t.secretRotationGraceImmediate
      : t.secretRotationGraceDays.replace('{days}', String(days))
  }

  const confirmMessage =
    graceDays === 0
      ? t.secretRotationConfirmImmediateMessage
      : t.secretRotationConfirmDaysMessage.replace('{days}', String(graceDays))

  async function rotate() {
    setBusy(true)
    onError('')
    try {
      const result = await rotateApplicationClientSecret(csrfToken, applicationID, graceDays)
      setSecret(result.client_secret)
      setCopied(false)
      setConfirming(false)
    } catch (cause) {
      onError(messageOf(cause, t.secretRotationError))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="rounded-xl border border-amber-300 bg-amber-50/60 p-5">
      <div className="flex items-start gap-2.5">
        <IconAlertTriangle
          size={18}
          className="mt-0.5 shrink-0 text-amber-700"
          aria-hidden="true"
        />
        <div>
          <h3 className="text-sm font-semibold text-amber-950">{t.secretRotationHeading}</h3>
          <p className="mt-1 text-xs leading-5 text-amber-800">{t.secretRotationDescription}</p>
        </div>
      </div>

      <div className="mt-4 flex flex-wrap items-end gap-3">
        <div className="grid gap-1.5">
          <Label htmlFor="secret-rotation-grace" className="text-xs text-amber-900">
            {t.secretRotationGraceLabel}
          </Label>
          <select
            id="secret-rotation-grace"
            value={graceDays}
            onChange={(event) => {
              setGraceDays(Number(event.target.value))
              setConfirming(false)
            }}
            disabled={busy}
            className="h-10 rounded-lg border border-amber-300 bg-white px-2 text-sm text-slate-800"
          >
            {GRACE_DAY_OPTIONS.map((days) => (
              <option key={days} value={days}>
                {graceOptionLabel(days)}
              </option>
            ))}
          </select>
        </div>
        {!confirming ? (
          <Button
            type="button"
            variant="outline"
            className="h-10 border-amber-300 text-amber-900 hover:border-amber-400 hover:bg-amber-100"
            disabled={busy}
            onClick={() => setConfirming(true)}
          >
            {t.secretRotationButton}
          </Button>
        ) : null}
      </div>

      {confirming ? (
        <Alert
          variant="destructive"
          className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
        >
          <span className="text-sm">{confirmMessage}</span>
          <div className="flex shrink-0 gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={() => setConfirming(false)}
            >
              {t.secretRotationCancelButton}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={busy}
              onClick={() => void rotate()}
            >
              {t.secretRotationConfirmButton}
            </Button>
          </div>
        </Alert>
      ) : null}

      {secret ? (
        <div className="mt-4 grid gap-2 border-t border-amber-200 pt-4">
          <CopyableField label={t.secretRotationNewSecretLabel} value={secret} />
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input type="checkbox" checked={copied} onChange={(e) => setCopied(e.target.checked)} />
            {t.secretRotationCopiedLabel}
          </label>
          {copied ? (
            <Button type="button" variant="outline" onClick={() => setSecret('')}>
              {t.secretRotationCloseButton}
            </Button>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

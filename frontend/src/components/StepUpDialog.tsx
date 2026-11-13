// ステップアップの再認可 (RFC 9470)。機微な操作が直近の再認証を求めて拒否されたとき、
// 確認を出し、利用者が押したボタンでポップアップの再認可を開き、成立後に元の操作を一回だけ
// 再試行する共通フック。ポップアップは利用者の操作の中で同期的に開かないとブラウザーが
// 遮断するので、確認のボタンを挟む。
import { useState } from 'react'
import {
  isStepUpRequired,
  openReauthorizationWindow,
  ReauthorizationCancelledError,
  reauthorize,
  type StepUpChallenge,
} from '../api'
import { Alert } from './ui/alert'
import { Button } from './ui/button'
import { commonDictionary } from '../lib/i18n/common.i18n'
import { useDictionary } from '../lib/i18n'

// ユーザーが再認可の確認を閉じた (キャンセルした) ことを表す。呼び出し側はこれを
// 「エラー表示なしの中断」として握りつぶす。
export class StepUpCancelledError extends Error {
  constructor() {
    super('step-up cancelled')
    this.name = 'StepUpCancelledError'
  }
}

interface PendingStepUp {
  challenge: StepUpChallenge
  resolve: () => void
  reject: (cause: unknown) => void
}

// useStepUpGuard は guard() でラップした操作が再認証を求められたら確認を開き、再認可の
// 成立後に同じ操作を再試行してその結果を呼び出し元へ返す。dialog を JSX に描画する。
export function useStepUpGuard() {
  const [pending, setPending] = useState<PendingStepUp | null>(null)

  async function guard<T>(action: () => Promise<T>): Promise<T> {
    try {
      return await action()
    } catch (cause) {
      if (!isStepUpRequired(cause)) throw cause
      await new Promise<void>((resolve, reject) => {
        setPending({ challenge: cause.challenge, resolve, reject })
      })
      return action()
    }
  }

  const dialog = pending ? (
    <ReauthorizationPrompt
      challenge={pending.challenge}
      onAuthenticated={() => {
        const current = pending
        setPending(null)
        current.resolve()
      }}
      onCancel={() => {
        const current = pending
        setPending(null)
        current.reject(new StepUpCancelledError())
      }}
    />
  ) : null

  return { guard, dialog }
}

interface ReauthorizationPromptProps {
  challenge: StepUpChallenge
  onAuthenticated: () => void
  onCancel: () => void
}

function ReauthorizationPrompt({
  challenge,
  onAuthenticated,
  onCancel,
}: ReauthorizationPromptProps) {
  const t = useDictionary(commonDictionary)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  function handleContinue() {
    // ポップアップはクリックの中で同期的に開く。開けなければ reauthorize が画面全体で再認可する。
    const popup = openReauthorizationWindow()
    setBusy(true)
    setError('')
    reauthorize(challenge, popup)
      .then(onAuthenticated)
      .catch((cause) => {
        setBusy(false)
        setError(
          cause instanceof ReauthorizationCancelledError ? t.stepUpNotCompleted : t.stepUpFailure,
        )
      })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="step-up-title"
      onKeyDown={(event) => {
        if (event.key === 'Escape' && !busy) onCancel()
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget && !busy) onCancel()
      }}
    >
      <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl">
        <h2 id="step-up-title" className="text-base font-semibold text-slate-900">
          {t.stepUpTitle}
        </h2>
        <p className="mt-1 text-sm text-slate-600">{t.stepUpDescription}</p>

        {error ? (
          <Alert variant="destructive" className="mt-4">
            {error}
          </Alert>
        ) : null}

        <div className="mt-4 flex justify-end gap-2">
          <Button type="button" variant="ghost" disabled={busy} onClick={onCancel}>
            {t.cancel}
          </Button>
          <Button type="button" autoFocus disabled={busy} onClick={handleContinue}>
            {busy ? t.stepUpWaiting : t.stepUpContinue}
          </Button>
        </div>
      </div>
    </div>
  )
}

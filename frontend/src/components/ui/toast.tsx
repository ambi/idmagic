import { IconCheck, IconX } from '@tabler/icons-react'
import { useEffect, useRef } from 'react'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { useDictionary } from '../../lib/i18n'

// Toast は操作成功メッセージを、レイアウトを押し下げない固定表示のオーバーレイで
// 短時間だけ見せる (wi-126 §5)。message が空なら何も描画しない。既定で数秒後に
// 自動で消え、閉じるボタンでも消せる。エラーは持続表示したいので従来どおり
// インラインの Alert を使い、本コンポーネントは成功系の通知に用いる。
const AUTO_DISMISS_MS = 4000

export function Toast({
  message,
  onDismiss,
  durationMs = AUTO_DISMISS_MS,
}: {
  message: string
  onDismiss: () => void
  durationMs?: number
}) {
  const t = useDictionary(commonDictionary)
  // onDismiss は呼び出し側でインライン関数になりがちなので ref に逃がし、
  // message が変わったときだけ自動消去タイマーを張り直す (親の再描画で
  // タイマーがリセットされ続けて消えなくなるのを防ぐ)。
  const dismissRef = useRef(onDismiss)
  dismissRef.current = onDismiss

  useEffect(() => {
    if (!message) return
    const timer = window.setTimeout(() => dismissRef.current(), durationMs)
    return () => window.clearTimeout(timer)
  }, [message, durationMs])

  if (!message) return null

  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex justify-center px-4 sm:inset-x-auto sm:right-4 sm:justify-end">
      <div
        role="status"
        aria-live="polite"
        className="pointer-events-auto flex items-center gap-2 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900 shadow-lg"
      >
        <IconCheck size={18} aria-hidden="true" />
        <span>{message}</span>
        <button
          type="button"
          onClick={onDismiss}
          aria-label={t.close}
          className="ml-1 rounded p-0.5 text-emerald-700 transition-colors hover:bg-emerald-100"
        >
          <IconX size={16} aria-hidden="true" />
        </button>
      </div>
    </div>
  )
}

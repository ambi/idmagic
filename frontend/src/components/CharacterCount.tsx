import { codePointLength } from '../lib/form'
import { useDictionary } from '../lib/i18n'
import { cn } from '../lib/utils'
import { fieldErrorDictionary } from './FieldError.i18n'

// 残りがこの割合を切ったら、支援技術に残りの文字数を知らせる。
const NEAR_LIMIT = 0.9

/**
 * 上限 500 以上の長い欄に、入力した文字数と上限を示す。数え方は検証と同じコードポイント数である。
 * 打つたびに読み上げると入力を妨げるので、数字の表示は読み上げず、上限に近づいたときと
 * 超えたときだけ `aria-live` で知らせる。超過のエラー文は FieldError が別に示す。
 */
export function CharacterCount({
  value,
  max,
  className,
}: {
  value: string
  max: number
  className?: string
}) {
  const t = useDictionary(fieldErrorDictionary)
  const count = codePointLength(value.trim())
  const over = count > max
  const announcement = over
    ? t.characterCountOver.replace('{count}', String(count - max))
    : count >= max * NEAR_LIMIT
      ? t.characterCountNear.replace('{count}', String(max - count))
      : ''
  return (
    <p
      className={cn(
        'text-right text-xs tabular-nums',
        over ? 'text-destructive' : 'text-slate-500',
        className,
      )}
    >
      <span aria-hidden="true">
        {t.characterCount.replace('{count}', String(count)).replace('{max}', String(max))}
      </span>
      <span className="sr-only" aria-live="polite">
        {announcement}
      </span>
    </p>
  )
}

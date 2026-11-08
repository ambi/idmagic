import { IconCheck, IconCopy } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { Button } from './ui/button'
import { useDictionary } from '../lib/i18n'
import { copyableValueDictionary } from './CopyableValue.i18n'

// CopyableValue は変更できない値 (client_id / secret 等) を入力欄ではなくテキストとして
// 表示し、コピーボタンだけを添える。フォームに見せないことで「編集不可」を明示する。
export function CopyableValue({
  value,
  copyLabel,
  multiline = false,
  id,
}: {
  value: string
  copyLabel: string
  multiline?: boolean
  id?: string
}) {
  const t = useDictionary(copyableValueDictionary)
  const [result, setResult] = useState<{
    value: string
    status: 'idle' | 'copying' | 'copied' | 'failed'
  }>({ value, status: 'idle' })
  const status = result.value === value ? result.status : 'idle'
  useEffect(() => {
    if (result.status !== 'copied') return
    const timer = window.setTimeout(
      () => setResult((current) => (current === result ? { ...current, status: 'idle' } : current)),
      1500,
    )
    return () => window.clearTimeout(timer)
  }, [result])

  async function copy() {
    setResult({ value, status: 'copying' })
    let next: 'copied' | 'failed'
    try {
      if (!navigator.clipboard) throw new Error('Clipboard unavailable')
      await navigator.clipboard.writeText(value)
      next = 'copied'
    } catch {
      next = 'failed'
    }
    setResult((current) => (current.value === value ? { value, status: next } : current))
  }
  return (
    <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
      {multiline ? (
        <pre
          id={id}
          className="min-w-0 flex-1 overflow-x-auto rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-800"
        >
          {value}
        </pre>
      ) : (
        <code
          id={id}
          className="min-w-0 flex-1 break-all rounded-md bg-slate-50 px-3 py-2 font-mono text-xs text-slate-800"
        >
          {value}
        </code>
      )}
      <Button
        type="button"
        variant="outline"
        className="size-9 shrink-0 px-0"
        aria-label={copyLabel}
        disabled={status === 'copying'}
        onClick={() => void copy()}
      >
        {status === 'copied' ? (
          <IconCheck size={16} className="text-emerald-600" aria-hidden="true" />
        ) : (
          <IconCopy size={16} aria-hidden="true" />
        )}
      </Button>
      {status === 'copied' ? (
        <span role="status" className="w-full text-xs text-slate-600">
          {t.copied}
        </span>
      ) : null}
      {status === 'failed' ? (
        <span role="alert" className="w-full text-xs text-red-700">
          {t.failed}
        </span>
      ) : null}
    </div>
  )
}
